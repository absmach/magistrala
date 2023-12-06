// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains smpp-notifier main function to start the smpp-notifier service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/consumers"
	"github.com/absmach/magistrala/consumers/notifiers"
	"github.com/absmach/magistrala/consumers/notifiers/api"
	notifierpg "github.com/absmach/magistrala/consumers/notifiers/postgres"
	mgsmpp "github.com/absmach/magistrala/consumers/notifiers/smpp"
	"github.com/absmach/magistrala/consumers/notifiers/tracing"
	"github.com/absmach/magistrala/internal"
	jaegerclient "github.com/absmach/magistrala/internal/clients/jaeger"
	pgclient "github.com/absmach/magistrala/internal/clients/postgres"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/ulid"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v10"
	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "smpp-notifier"
	envPrefixDB    = "MG_SMPP_NOTIFIER_DB_"
	envPrefixHTTP  = "MG_SMPP_NOTIFIER_HTTP_"
	envPrefixAuth  = "MG_AUTH_GRPC_"
	defDB          = "subscriptions"
	defSvcHTTPPort = "9014"
)

type config struct {
	LogLevel      string  `env:"MG_SMPP_NOTIFIER_LOG_LEVEL"     envDefault:"info"`
	From          string  `env:"MG_SMPP_NOTIFIER_FROM_ADDR"     envDefault:""`
	ConfigPath    string  `env:"MG_SMPP_NOTIFIER_CONFIG_PATH"   envDefault:"/config.toml"`
	BrokerURL     string  `env:"MG_MESSAGE_BROKER_URL"          envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"                  envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"              envDefault:"true"`
	InstanceID    string  `env:"MG_SMPP_NOTIFIER_INSTANCE_ID"   envDefault:""`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"          envDefault:"1.0"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mglog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	var exitCode int
	defer mglog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s Postgres configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	db, err := pgclient.Setup(dbConfig, *notifierpg.Migration())
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer db.Close()

	smppConfig := mgsmpp.Config{}
	if err := env.Parse(&smppConfig); err != nil {
		logger.Error(fmt.Sprintf("failed to load SMPP configuration from environment : %s", err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	pubSub, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pubSub.Close()
	pubSub = brokerstracing.NewPubSub(httpServerConfig, tracer, pubSub)

	authConfig := auth.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authClient, authHandler, err := auth.Setup(authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	svc := newService(db, tracer, authClient, cfg, smppConfig, logger)
	if err = consumers.Start(ctx, svcName, pubSub, svc, cfg.ConfigPath, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to create Postgres writer: %s", err))
		exitCode = 1
		return
	}

	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
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

func newService(db *sqlx.DB, tracer trace.Tracer, authClient magistrala.AuthServiceClient, c config, sc mgsmpp.Config, logger mglog.Logger) notifiers.Service {
	database := notifierpg.NewDatabase(db, tracer)
	repo := tracing.New(tracer, notifierpg.New(database))
	idp := ulid.New()
	notifier := mgsmpp.New(sc)
	svc := notifiers.New(authClient, repo, idp, notifier, c.From)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("notifier", "smpp")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
