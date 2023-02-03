// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/cassandra"
	"github.com/mainflux/mainflux/internal"
	cassandraClient "github.com/mainflux/mainflux/internal/clients/cassandra"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "cassandra-writer"
	envPrefix      = "MF_CASSANDRA_WRITER_"
	envPrefixHttp  = "MF_CASSANDRA_WRITER_HTTP_"
	defSvcHttpPort = "8180"
)

type config struct {
	LogLevel   string `env:"MF_CASSANDRA_WRITER_LOG_LEVEL"     envDefault:"info"`
	ConfigPath string `env:"MF_CASSANDRA_WRITER_CONFIG_PATH"   envDefault:"/config.toml"`
	BrokerURL  string `env:"MF_BROKER_URL"                     envDefault:"nats://localhost:4222"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new cassandra writer service configurations
	cfg := config{}

	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s service configuration : %s", svcName, err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Create new to cassandra client
	csdSession, err := cassandraClient.SetupDB(envPrefix, cassandra.Table)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer csdSession.Close()

	// Create new cassandra-writer repo
	repo := newService(csdSession, logger)

	// Create new pub sub broker
	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		log.Fatalf("failed to connect to message broker: %s", err.Error())
	}
	defer pubSub.Close()

	// Start new consumer
	if err := consumers.Start(svcName, pubSub, repo, cfg.ConfigPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Cassandra writer: %s", err))
	}

	// Create new http server
	httpServerConfig := server.Config{Port: defSvcHttpPort}

	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefix, AltPrefix: envPrefixHttp}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}

	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svcName), logger)

	// Start servers
	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Cassandra writer service terminated: %s", err))
	}

}

func newService(session *gocql.Session, logger logger.Logger) consumers.Consumer {
	repo := cassandra.New(session)
	repo = api.LoggingMiddleware(repo, logger)
	counter, latency := internal.MakeMetrics("cassandra", "message_writer")
	repo = api.MetricsMiddleware(repo, counter, latency)
	return repo
}
