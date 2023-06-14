// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains smpp-notifier main function to start the smpp-notifier service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/notifiers"
	"github.com/mainflux/mainflux/consumers/notifiers/api"
	notifierPg "github.com/mainflux/mainflux/consumers/notifiers/postgres"
	"github.com/mainflux/mainflux/internal"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/users/policies"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	mfsmpp "github.com/mainflux/mainflux/consumers/notifiers/smpp"
	"github.com/mainflux/mainflux/consumers/notifiers/tracing"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	pstracing "github.com/mainflux/mainflux/pkg/messaging/tracing"
	"github.com/mainflux/mainflux/pkg/ulid"
)

const (
	svcName        = "smpp-notifier"
	envPrefix      = "MF_SMPP_NOTIFIER_"
	envPrefixHttp  = "MF_SMPP_NOTIFIER_HTTP_"
	defDB          = "subscriptions"
	defSvcHttpPort = "9014"
)

type config struct {
	LogLevel      string `env:"MF_SMPP_NOTIFIER_LOG_LEVEL"   envDefault:"info"`
	From          string `env:"MF_SMPP_NOTIFIER_FROM_ADDR"   envDefault:""`
	ConfigPath    string `env:"MF_SMPP_NOTIFIER_CONFIG_PATH" envDefault:"/config.toml"`
	BrokerURL     string `env:"MF_BROKER_URL"                envDefault:"nats://localhost:4222"`
	JaegerURL     string `env:"MF_JAEGER_URL"                envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry bool   `env:"MF_SEND_TELEMETRY"            envDefault:"true"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	dbConfig := pgClient.Config{Name: defDB}
	db, err := pgClient.SetupWithConfig(envPrefix, *notifierPg.Migration(), dbConfig)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer db.Close()

	smppConfig := mfsmpp.Config{}
	if err := env.Parse(&smppConfig); err != nil {
		logger.Fatal(fmt.Sprintf("failed to load SMPP configuration from environment : %s", err))
	}

	tp, err := jaegerClient.NewProvider(svcName, cfg.JaegerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to connect to message broker: %s", err))
	}
	pubSub = pstracing.NewPubSub(tracer, pubSub)
	defer pubSub.Close()

	auth, authHandler, err := authClient.Setup(envPrefix, svcName, cfg.JaegerURL)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	svc := newService(db, tracer, auth, cfg, smppConfig, logger)
	if err = consumers.Start(ctx, svcName, pubSub, svc, cfg.ConfigPath, logger); err != nil {
		logger.Fatal(fmt.Sprintf("failed to create Postgres writer: %s", err))
	}

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Fatal(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("SMPP notifier service terminated: %s", err))
	}

}

func newService(db *sqlx.DB, tracer trace.Tracer, auth policies.AuthServiceClient, c config, sc mfsmpp.Config, logger mflog.Logger) notifiers.Service {
	database := notifierPg.NewDatabase(db, tracer)
	repo := tracing.New(tracer, notifierPg.New(database))
	idp := ulid.New()
	notifier := mfsmpp.New(sc)
	svc := notifiers.New(auth, repo, idp, notifier, c.From)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("notifier", "smpp")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
