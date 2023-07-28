// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains bootstrap main function to start the bootstrap service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"go.opentelemetry.io/otel/trace"

	bootstrapPg "github.com/mainflux/mainflux/bootstrap/postgres"
	rediscons "github.com/mainflux/mainflux/bootstrap/redis/consumer"
	redisprod "github.com/mainflux/mainflux/bootstrap/redis/producer"
	"github.com/mainflux/mainflux/bootstrap/tracing"
	"github.com/mainflux/mainflux/internal"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/postgres"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users/policies"
	"golang.org/x/sync/errgroup"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/api"
)

const (
	svcName        = "bootstrap"
	envPrefix      = "MF_BOOTSTRAP_"
	envPrefixES    = "MF_BOOTSTRAP_ES_"
	envPrefixHttp  = "MF_BOOTSTRAP_HTTP_"
	defDB          = "bootstrap"
	defSvcHttpPort = "9013"
)

type config struct {
	LogLevel       string `env:"MF_BOOTSTRAP_LOG_LEVEL"        envDefault:"info"`
	EncKey         string `env:"MF_BOOTSTRAP_ENCRYPT_KEY"      envDefault:"12345678910111213141516171819202"`
	ESConsumerName string `env:"MF_BOOTSTRAP_EVENT_CONSUMER"   envDefault:"bootstrap"`
	ThingsURL      string `env:"MF_THINGS_URL"                 envDefault:"http://localhost:9000"`
	JaegerURL      string `env:"MF_JAEGER_URL"                 envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry  bool   `env:"MF_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID     string `env:"MF_BOOTSTRAP_INSTANCE_ID"      envDefault:""`
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

	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID, err = uuid.New().ID()
		if err != nil {
			log.Fatalf("Failed to generate instanceID: %s", err)
		}
	}

	// Create new postgres client
	dbConfig := pgClient.Config{Name: defDB}

	db, err := pgClient.SetupWithConfig(envPrefix, *bootstrapPg.Migration(), dbConfig)
	if err != nil {
		logger.Fatal(err.Error())
	}
	var exitCode int
	defer mflog.ExitWithError(&exitCode)
	defer db.Close()

	// Create new redis client for bootstrap event store
	esClient, err := redisClient.Setup(envPrefixES)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup %s bootstrap event store redis client : %s", svcName, err))
		exitCode = 1
		return
	}
	defer esClient.Close()

	// Create new auth grpc client api
	auth, authHandler, err := authClient.Setup(envPrefix, svcName)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	tp, err := jaegerClient.NewProvider(svcName, cfg.JaegerURL, instanceID)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to init Jaeger: %s", err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	// Create new service
	svc := newService(auth, db, tracer, logger, esClient, cfg)

	// Create an new HTTP server
	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, bootstrap.NewConfigReader([]byte(cfg.EncKey)), logger, instanceID), logger)

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

	// Subscribe to things event store
	thingsESClient, err := redisClient.Setup(envPrefixES)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer thingsESClient.Close()

	go subscribeToThingsES(ctx, svc, thingsESClient, cfg.ESConsumerName, logger)

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Bootstrap service terminated: %s", err))
	}
}

func newService(auth policies.AuthServiceClient, db *sqlx.DB, tracer trace.Tracer, logger mflog.Logger, esClient *redis.Client, cfg config) bootstrap.Service {
	database := postgres.NewDatabase(db, tracer)
	repoConfig := bootstrapPg.NewConfigRepository(database, logger)

	config := mfsdk.Config{
		ThingsURL: cfg.ThingsURL,
	}

	sdk := mfsdk.NewSDK(config)

	svc := bootstrap.New(auth, repoConfig, sdk, []byte(cfg.EncKey))
	svc = redisprod.NewEventStoreMiddleware(svc, esClient)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc
}

func subscribeToThingsES(ctx context.Context, svc bootstrap.Service, client *redis.Client, consumer string, logger mflog.Logger) {
	eventStore := rediscons.NewEventStore(svc, client, consumer, logger)
	logger.Info("Subscribed to Redis Event Store")
	if err := eventStore.Subscribe(ctx, "mainflux.things"); err != nil {
		logger.Warn(fmt.Sprintf("Bootstrap service failed to subscribe to event sourcing: %s", err))
	}
}
