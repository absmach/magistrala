// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains cassandra-writer main function to start the cassandra-writer service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gocql/gocql"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	consumertracing "github.com/mainflux/mainflux/consumers/tracing"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/cassandra"
	"github.com/mainflux/mainflux/internal"
	cassandraclient "github.com/mainflux/mainflux/internal/clients/cassandra"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	brokerstracing "github.com/mainflux/mainflux/pkg/messaging/brokers/tracing"
	"github.com/mainflux/mainflux/pkg/uuid"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "cassandra-writer"
	envPrefixDB    = "MF_CASSANDRA_"
	envPrefixHTTP  = "MF_CASSANDRA_WRITER_HTTP_"
	defSvcHTTPPort = "9004"
)

type config struct {
	LogLevel      string `env:"MF_CASSANDRA_WRITER_LOG_LEVEL"     envDefault:"info"`
	ConfigPath    string `env:"MF_CASSANDRA_WRITER_CONFIG_PATH"   envDefault:"/config.toml"`
	BrokerURL     string `env:"MF_BROKER_URL"                     envDefault:"nats://localhost:4222"`
	JaegerURL     string `env:"MF_JAEGER_URL"                     envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry bool   `env:"MF_SEND_TELEMETRY"                 envDefault:"true"`
	InstanceID    string `env:"MF_CASSANDRA_WRITER_INSTANCE_ID"   envDefault:""`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new cassandra writer service configurations
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

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	// Create new to cassandra client
	csdSession, err := cassandraclient.SetupDB(envPrefixDB, cassandra.Table)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer csdSession.Close()

	tp, err := jaegerclient.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID)
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

	// Create new cassandra-writer repo
	repo := newService(csdSession, logger)
	repo = consumertracing.NewBlocking(tracer, repo, httpServerConfig)

	// Create new pub sub broker
	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pubSub.Close()
	pubSub = brokerstracing.NewPubSub(httpServerConfig, tracer, pubSub)

	// Start new consumer
	if err := consumers.Start(ctx, svcName, pubSub, repo, cfg.ConfigPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Cassandra writer: %s", err))
		exitCode = 1
		return
	}

	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svcName, cfg.InstanceID), logger)

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
		logger.Error(fmt.Sprintf("Cassandra writer service terminated: %s", err))
	}
}

func newService(session *gocql.Session, logger mflog.Logger) consumers.BlockingConsumer {
	repo := cassandra.New(session)
	repo = api.LoggingMiddleware(repo, logger)
	counter, latency := internal.MakeMetrics("cassandra", "message_writer")
	repo = api.MetricsMiddleware(repo, counter, latency)
	return repo
}
