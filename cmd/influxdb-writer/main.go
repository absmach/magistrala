// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/influxdb"
	"github.com/mainflux/mainflux/internal"
	influxDBClient "github.com/mainflux/mainflux/internal/clients/influxdb"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	"golang.org/x/sync/errgroup"
)

const (
	svcName           = "influxdb-writer"
	envPrefix         = "MF_INFLUX_WRITER_"
	envPrefixHttp     = "MF_INFLUX_WRITER_HTTP_"
	envPrefixInfluxdb = "MF_INFLUXDB_"
	defSvcHttpPort    = "8180"
)

type config struct {
	LogLevel   string `env:"MF_INFLUX_WRITER_LOG_LEVEL"     envDefault:"info"`
	ConfigPath string `env:"MF_INFLUX_WRITER_CONFIG_PATH"   envDefault:"/config.toml"`
	BrokerURL  string `env:"MF_BROKER_URL"                  envDefault:"nats://localhost:4222"`
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
		log.Fatalf(err.Error())
	}

	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		log.Fatalf("failed to connect to message broker: %s", err.Error())
	}
	defer pubSub.Close()

	influxDBConfig := influxDBClient.Config{}
	if err := env.Parse(&influxDBConfig, env.Options{Prefix: envPrefixInfluxdb}); err != nil {
		log.Fatalf("failed to load InfluxDB client configuration from environment variable : %s", err.Error())
	}
	client, err := influxDBClient.Connect(influxDBConfig)
	if err != nil {
		log.Fatalf("failed to connect to InfluxDB : %s", err.Error())
	}
	defer client.Close()

	repo := newService(client, influxDBConfig.DbName, logger)

	if err := consumers.Start(svcName, pubSub, repo, cfg.ConfigPath, logger); err != nil {
		log.Fatalf("failed to start InfluxDB writer: %s", err.Error())
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
		logger.Error(fmt.Sprintf("InfluxDB reader service terminated: %s", err))
	}
}

func newService(client influxdata.Client, dbName string, logger logger.Logger) consumers.Consumer {
	repo := influxdb.New(client, dbName)
	repo = api.LoggingMiddleware(repo, logger)
	counter, latency := internal.MakeMetrics("influxdb", "message_writer")
	repo = api.MetricsMiddleware(repo, counter, latency)
	return repo
}
