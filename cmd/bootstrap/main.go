// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	r "github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/bootstrap"
	api "github.com/mainflux/mainflux/bootstrap/api"
	bootstrapPg "github.com/mainflux/mainflux/bootstrap/postgres"
	rediscons "github.com/mainflux/mainflux/bootstrap/redis/consumer"
	redisprod "github.com/mainflux/mainflux/bootstrap/redis/producer"
	"github.com/mainflux/mainflux/internal"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "bootstrap"
	envPrefix      = "MF_BOOTSTRAP_"
	envPrefixES    = "MF_BOOTSTRAP_ES_"
	envPrefixHttp  = "MF_BOOTSTRAP_HTTP_"
	defDB          = "bootstrap"
	defSvcHttpPort = "8180"
)

type config struct {
	LogLevel       string `env:"MF_BOOTSTRAP_LOG_LEVEL"        envDefault:"info"`
	EncKey         string `env:"MF_BOOTSTRAP_ENCRYPT_KEY"      envDefault:"12345678910111213141516171819202"`
	ESConsumerName string `env:"MF_BOOTSTRAP_EVENT_CONSUMER"   envDefault:"bootstrap"`
	ThingsURL      string `env:"MF_THINGS_URL"                 envDefault:"http://localhost"`
	JaegerURL      string `env:"MF_JAEGER_URL"                 envDefault:"localhost:6831"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatal(err.Error())
	}

	// Create new postgres client
	dbConfig := pgClient.Config{Name: defDB}

	db, err := pgClient.SetupWithConfig(envPrefix, *bootstrapPg.Migration(), dbConfig)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer db.Close()

	// Create new redis client for bootstrap event store
	esClient, err := redisClient.Setup(envPrefixES)
	if err != nil {
		log.Fatalf("failed to setup %s bootstrap event store redis client : %s", svcName, err.Error())
	}
	defer esClient.Close()

	// Create new auth grpc client api
	auth, authHandler, err := authClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		log.Fatal(err)
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	// Create new service
	svc := newService(auth, db, logger, esClient, cfg)

	// Create an new HTTP server
	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, bootstrap.NewConfigReader([]byte(cfg.EncKey)), logger), logger)

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
		log.Fatalf(err.Error())
	}
	defer thingsESClient.Close()

	go subscribeToThingsES(svc, thingsESClient, cfg.ESConsumerName, logger)

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Bootstrap service terminated: %s", err))
	}
}

func newService(auth mainflux.AuthServiceClient, db *sqlx.DB, logger logger.Logger, esClient *r.Client, cfg config) bootstrap.Service {
	repoConfig := bootstrapPg.NewConfigRepository(db, logger)

	config := mfsdk.Config{
		ThingsURL: cfg.ThingsURL,
	}

	sdk := mfsdk.NewSDK(config)

	svc := bootstrap.New(auth, repoConfig, sdk, []byte(cfg.EncKey))
	svc = redisprod.NewEventStoreMiddleware(svc, esClient)
	svc = api.NewLoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}

func subscribeToThingsES(svc bootstrap.Service, client *r.Client, consumer string, logger logger.Logger) {
	eventStore := rediscons.NewEventStore(svc, client, consumer, logger)
	logger.Info("Subscribed to Redis Event Store")
	if err := eventStore.Subscribe(context.Background(), "mainflux.things"); err != nil {
		logger.Warn(fmt.Sprintf("Bootstrap service failed to subscribe to event sourcing: %s", err))
	}
}
