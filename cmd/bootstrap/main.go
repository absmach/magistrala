// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains bootstrap main function to start the bootstrap service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/api"
	"github.com/absmach/magistrala/bootstrap/events/consumer"
	"github.com/absmach/magistrala/bootstrap/events/producer"
	bootstrappg "github.com/absmach/magistrala/bootstrap/postgres"
	"github.com/absmach/magistrala/bootstrap/tracing"
	"github.com/absmach/magistrala/internal"
	"github.com/absmach/magistrala/internal/clients/jaeger"
	pgclient "github.com/absmach/magistrala/internal/clients/postgres"
	"github.com/absmach/magistrala/internal/postgres"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/events/store"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v10"
	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "bootstrap"
	envPrefixDB    = "MG_BOOTSTRAP_DB_"
	envPrefixHTTP  = "MG_BOOTSTRAP_HTTP_"
	envPrefixAuth  = "MG_AUTH_GRPC_"
	defDB          = "bootstrap"
	defSvcHTTPPort = "9013"

	thingsStream = "magistrala.things"
	streamID     = "magistrala.bootstrap"
)

type config struct {
	LogLevel       string  `env:"MG_BOOTSTRAP_LOG_LEVEL"        envDefault:"info"`
	EncKey         string  `env:"MG_BOOTSTRAP_ENCRYPT_KEY"      envDefault:"12345678910111213141516171819202"`
	ESConsumerName string  `env:"MG_BOOTSTRAP_EVENT_CONSUMER"   envDefault:"bootstrap"`
	ThingsURL      string  `env:"MG_THINGS_URL"                 envDefault:"http://localhost:9000"`
	JaegerURL      url.URL `env:"MG_JAEGER_URL"                 envDefault:"http://localhost:14268/api/traces"`
	SendTelemetry  bool    `env:"MG_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID     string  `env:"MG_BOOTSTRAP_INSTANCE_ID"      envDefault:""`
	ESURL          string  `env:"MG_ES_URL"                     envDefault:"nats://localhost:4222"`
	TraceRatio     float64 `env:"MG_JAEGER_TRACE_RATIO"         envDefault:"1.0"`
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

	// Create new postgres client
	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Fatal(err.Error())
	}
	db, err := pgclient.Setup(dbConfig, *bootstrappg.Migration())
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

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

	tp, err := jaeger.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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
	svc, err := newService(ctx, authClient, db, tracer, logger, cfg, dbConfig)
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
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, bootstrap.NewConfigReader([]byte(cfg.EncKey)), logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
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

func newService(ctx context.Context, authClient magistrala.AuthServiceClient, db *sqlx.DB, tracer trace.Tracer, logger mglog.Logger, cfg config, dbConfig pgclient.Config) (bootstrap.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)

	repoConfig := bootstrappg.NewConfigRepository(database, logger)

	config := mgsdk.Config{
		ThingsURL: cfg.ThingsURL,
	}

	sdk := mgsdk.NewSDK(config)

	svc := bootstrap.New(authClient, repoConfig, sdk, []byte(cfg.EncKey))

	publisher, err := store.NewPublisher(ctx, cfg.ESURL, streamID)
	if err != nil {
		return nil, err
	}

	svc = producer.NewEventStoreMiddleware(svc, publisher)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc, nil
}

func subscribeToThingsES(ctx context.Context, svc bootstrap.Service, cfg config, logger mglog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, cfg.ESURL, thingsStream, cfg.ESConsumerName, logger)
	if err != nil {
		return err
	}

	handler := consumer.NewEventHandler(svc)

	logger.Info("Subscribed to Redis Event Store")

	return subscriber.Subscribe(ctx, handler)
}
