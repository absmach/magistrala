// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/influxdb"
	"github.com/mainflux/mainflux/internal"
	influxDBClient "github.com/mainflux/mainflux/internal/clients/influxdb"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
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
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to connect to message broker: %s", err))
	}
	defer pubSub.Close()

	influxDBConfig := influxDBClient.Config{}
	if err := env.Parse(&influxDBConfig, env.Options{Prefix: envPrefixInfluxdb}); err != nil {
		logger.Fatal(fmt.Sprintf("failed to load InfluxDB client configuration from environment variable : %s", err))
	}
	influxDBConfig.DBUrl = fmt.Sprintf("%s://%s:%s", influxDBConfig.Protocol, influxDBConfig.Host, influxDBConfig.Port)
	repocfg := influxdb.RepoConfig{
		Bucket: influxDBConfig.Bucket,
		Org:    influxDBConfig.Org,
	}

	client, err := influxDBClient.Connect(influxDBConfig, ctx)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to connect to InfluxDB : %s", err))
	}
	defer client.Close()

	repo := newService(client, repocfg, logger)

	if err := consumers.Start(svcName, pubSub, repo, cfg.ConfigPath, logger); err != nil {
		logger.Fatal(fmt.Sprintf("failed to start InfluxDB writer: %s", err))
	}

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Fatal(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
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

func newService(client influxdb2.Client, repocfg influxdb.RepoConfig, logger mflog.Logger) consumers.Consumer {
	repo := influxdb.New(client, repocfg, true)
	repo = api.LoggingMiddleware(repo, logger)
	counter, latency := internal.MakeMetrics("influxdb", "message_writer")
	repo = api.MetricsMiddleware(repo, counter, latency)
	return repo
}
