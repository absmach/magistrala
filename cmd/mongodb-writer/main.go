// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/mongodb"
	"github.com/mainflux/mainflux/internal"
	mongoClient "github.com/mainflux/mainflux/internal/clients/mongo"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "mongodb-writer"
	envPrefix      = "MF_MONGO_WRITER_"
	envPrefixDB    = "MF_MONGO_WRITER_DB_"
	envPrefixHttp  = "MF_MONGO_WRITER_HTTP_"
	defSvcHttpPort = "8180"
)

type config struct {
	LogLevel   string `env:"MF_MONGO_WRITER_LOG_LEVEL"     envDefault:"info"`
	ConfigPath string `env:"MF_MONGO_WRITER_CONFIG_PATH"   envDefault:"/config.toml"`
	BrokerURL  string `env:"MF_BROKER_URL"                 envDefault:"nats://localhost:4222"`
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
		log.Fatal(err)
	}

	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		log.Fatalf("failed to connect to message broker: %s", err.Error())
	}
	defer pubSub.Close()

	db, err := mongoClient.Setup(envPrefixDB)
	if err != nil {
		log.Fatalf("failed to setup mongo database : %s", err.Error())
	}

	repo := newService(db, logger)

	if err := consumers.Start(svcName, pubSub, repo, cfg.ConfigPath, logger); err != nil {
		log.Fatalf("failed to start MongoDB writer: %s", err.Error())
	}

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svcName), logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("MongoDB writer service terminated: %s", err))
	}
}

func newService(db *mongo.Database, logger logger.Logger) consumers.Consumer {
	repo := mongodb.New(db)
	repo = api.LoggingMiddleware(repo, logger)
	counter, latency := internal.MakeMetrics("mongodb", "message_writer")
	repo = api.MetricsMiddleware(repo, counter, latency)
	return repo
}
