// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains bootstrap main function to start the bootstrap service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/api"
	"github.com/mainflux/mainflux/bootstrap/events/consumer"
	"github.com/mainflux/mainflux/bootstrap/events/producer"
	bootstrappg "github.com/mainflux/mainflux/bootstrap/postgres"
	"github.com/mainflux/mainflux/bootstrap/tracing"
	"github.com/mainflux/mainflux/internal"
	authclient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	"github.com/mainflux/mainflux/internal/clients/jaeger"
	pgclient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/postgres"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/events/redis"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "bootstrap"
	envPrefixDB    = "MF_BOOTSTRAP_DB_"
	envPrefixHTTP  = "MF_BOOTSTRAP_HTTP_"
	defDB          = "bootstrap"
	defSvcHTTPPort = "9013"

	thingsStream = "mainflux.things"
)

type config struct {
	LogLevel       string `env:"MF_BOOTSTRAP_LOG_LEVEL"        envDefault:"info"`
	EncKey         string `env:"MF_BOOTSTRAP_ENCRYPT_KEY"      envDefault:"12345678910111213141516171819202"`
	ESConsumerName string `env:"MF_BOOTSTRAP_EVENT_CONSUMER"   envDefault:"bootstrap"`
	ThingsURL      string `env:"MF_THINGS_URL"                 envDefault:"http://localhost:9000"`
	JaegerURL      string `env:"MF_JAEGER_URL"                 envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry  bool   `env:"MF_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID     string `env:"MF_BOOTSTRAP_INSTANCE_ID"      envDefault:""`
	ESURL          string `env:"MF_BOOTSTRAP_ES_URL"           envDefault:"redis://localhost:6379/0"`
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

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	// Create new postgres client
	dbConfig := pgclient.Config{Name: defDB}
	if err := dbConfig.LoadEnv(envPrefixDB); err != nil {
		logger.Fatal(err.Error())
	}
	db, err := pgclient.SetupWithConfig(envPrefixDB, *bootstrappg.Migration(), dbConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	// Create new auth grpc client api
	auth, authHandler, err := authclient.Setup(svcName)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	tp, err := jaeger.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	// Create new service
	svc, err := newService(ctx, auth, db, tracer, logger, cfg, dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s service: %s", svcName, err))
		exitCode = 1
		return
	}

	if err = subscribeToThingsES(ctx, svc, cfg, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to things event store: %s", err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, bootstrap.NewConfigReader([]byte(cfg.EncKey)), logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	// Start servers
	g.Go(func() error {
		return hs.Start()
	})
	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Bootstrap service terminated: %s", err))
	}
}

func newService(ctx context.Context, auth mainflux.AuthServiceClient, db *sqlx.DB, tracer trace.Tracer, logger mflog.Logger, cfg config, dbConfig pgclient.Config) (bootstrap.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)

	repoConfig := bootstrappg.NewConfigRepository(database, logger)

	config := mfsdk.Config{
		ThingsURL: cfg.ThingsURL,
	}

	sdk := mfsdk.NewSDK(config)

	svc := bootstrap.New(auth, repoConfig, sdk, []byte(cfg.EncKey))

	var err error
	svc, err = producer.NewEventStoreMiddleware(ctx, svc, cfg.ESURL)
	if err != nil {
		return nil, err
	}
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc, nil
}

func subscribeToThingsES(ctx context.Context, svc bootstrap.Service, cfg config, logger mflog.Logger) error {
	subscriber, err := redis.NewSubscriber(cfg.ESURL, thingsStream, cfg.ESConsumerName, logger)
	if err != nil {
		return err
	}

	handler := consumer.NewEventHandler(svc)

	logger.Info("Subscribed to Redis Event Store")

	return subscriber.Subscribe(ctx, handler)
}
