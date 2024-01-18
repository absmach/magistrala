// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains postgres-writer main function to start the postgres-writer service.
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
	writerpg "github.com/absmach/magistrala/consumers/writers/postgres"
	"github.com/absmach/magistrala/internal"
	jaegerclient "github.com/absmach/magistrala/internal/clients/jaeger"
	pgclient "github.com/absmach/magistrala/internal/clients/postgres"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v10"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "postgres-writer"
	envPrefixDB    = "MG_POSTGRES_"
	envPrefixHTTP  = "MG_POSTGRES_WRITER_HTTP_"
	defDB          = "messages"
	defSvcHTTPPort = "9010"
)

type config struct {
	LogLevel      string  `env:"MG_POSTGRES_WRITER_LOG_LEVEL"     envDefault:"info"`
	ConfigPath    string  `env:"MG_POSTGRES_WRITER_CONFIG_PATH"   envDefault:"/config.toml"`
	BrokerURL     string  `env:"MG_MESSAGE_BROKER_URL"            envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"                    envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"                envDefault:"true"`
	InstanceID    string  `env:"MG_POSTGRES_WRITER_INSTANCE_ID"   envDefault:""`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"            envDefault:"1.0"`
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

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s Postgres configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	db, err := pgclient.Setup(dbConfig, *writerpg.Migration())
	if err != nil {
		logger.Error(err.Error())
	}
	defer db.Close()

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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

	repo := newService(db, logger)
	repo = consumertracing.NewBlocking(tracer, repo, httpServerConfig)

	if err = consumers.Start(ctx, svcName, pubSub, repo, cfg.ConfigPath, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to create Postgres writer: %s", err))
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
		logger.Error(fmt.Sprintf("Postgres writer service terminated: %s", err))
	}
}

func newService(db *sqlx.DB, logger *slog.Logger) consumers.BlockingConsumer {
	svc := writerpg.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("postgres", "message_writer")
	svc = api.MetricsMiddleware(svc, counter, latency)
	return svc
}
