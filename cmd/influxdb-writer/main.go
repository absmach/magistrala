// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains influxdb-writer main function to start the influxdb-writer service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/consumers"
	consumertracing "github.com/absmach/magistrala/consumers/tracing"
	"github.com/absmach/magistrala/consumers/writers/api"
	"github.com/absmach/magistrala/consumers/writers/influxdb"
	influxdbclient "github.com/absmach/magistrala/internal/clients/influxdb"
	"github.com/absmach/magistrala/internal/clients/jaeger"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v10"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "influxdb-writer"
	envPrefixHTTP  = "MG_INFLUX_WRITER_HTTP_"
	envPrefixDB    = "MG_INFLUXDB_"
	defSvcHTTPPort = "9006"
)

type config struct {
	LogLevel      string  `env:"MG_INFLUX_WRITER_LOG_LEVEL"     envDefault:"info"`
	ConfigPath    string  `env:"MG_INFLUX_WRITER_CONFIG_PATH"   envDefault:"/config.toml"`
	BrokerURL     string  `env:"MG_MESSAGE_BROKER_URL"          envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"                  envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"              envDefault:"true"`
	InstanceID    string  `env:"MG_INFLUX_WRITER_INSTANCE_ID"   envDefault:""`
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
		log.Fatalf("failed to init logger: %s", err.Error())
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

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	tp, err := jaeger.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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

	influxDBConfig := influxdbclient.Config{}
	if err := env.ParseWithOptions(&influxDBConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(fmt.Sprintf("failed to load InfluxDB client configuration from environment variable : %s", err))
		exitCode = 1
		return
	}
	influxDBConfig.DBUrl = fmt.Sprintf("%s://%s:%s", influxDBConfig.Protocol, influxDBConfig.Host, influxDBConfig.Port)

	repocfg := influxdb.RepoConfig{
		Bucket: influxDBConfig.Bucket,
		Org:    influxDBConfig.Org,
	}

	client, err := influxdbclient.Connect(ctx, influxDBConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to InfluxDB : %s", err))
		exitCode = 1
		return
	}
	defer client.Close()

	repo := influxdb.NewAsync(client, repocfg)
	repo = consumertracing.NewAsync(tracer, repo, httpServerConfig)

	// Start consuming and logging errors.
	go func(log *slog.Logger) {
		for err := range repo.Errors() {
			if err != nil {
				log.Error(err.Error())
			}
		}
	}(logger)

	if err := consumers.Start(ctx, svcName, pubSub, repo, cfg.ConfigPath, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to start InfluxDB writer: %s", err))
		exitCode = 1
		return
	}

	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svcName, cfg.InstanceID), logger)

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
		logger.Error(fmt.Sprintf("InfluxDB reader service terminated: %s", err))
	}
}
