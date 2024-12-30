// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains timescale-writer main function to start the timescale-writer service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/consumers"
	consumertracing "github.com/absmach/supermq/consumers/tracing"
	httpapi "github.com/absmach/supermq/consumers/writers/api"
	"github.com/absmach/supermq/consumers/writers/timescale"
	smqlog "github.com/absmach/supermq/logger"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/supermq/pkg/messaging/brokers/tracing"
	pgclient "github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "timescaledb-writer"
	envPrefixDB    = "SMQ_TIMESCALE_"
	envPrefixHTTP  = "SMQ_TIMESCALE_WRITER_HTTP_"
	defDB          = "messages"
	defSvcHTTPPort = "9012"
)

type config struct {
	LogLevel      string  `env:"SMQ_TIMESCALE_WRITER_LOG_LEVEL"    envDefault:"info"`
	ConfigPath    string  `env:"SMQ_TIMESCALE_WRITER_CONFIG_PATH"  envDefault:"/config.toml"`
	BrokerURL     string  `env:"SMQ_MESSAGE_BROKER_URL"            envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"SMQ_JAEGER_URL"                    envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry bool    `env:"SMQ_SEND_TELEMETRY"                envDefault:"true"`
	InstanceID    string  `env:"SMQ_TIMESCALE_WRITER_INSTANCE_ID"  envDefault:""`
	TraceRatio    float64 `env:"SMQ_JAEGER_TRACE_RATIO"            envDefault:"1.0"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s service configuration : %s", svcName, err)
	}

	logger, err := smqlog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer smqlog.ExitWithError(&exitCode)

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
	db, err := pgclient.Setup(dbConfig, *timescale.Migration())
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
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

	repo := newService(db, logger)
	repo = consumertracing.NewBlocking(tracer, repo, httpServerConfig)

	pubSub, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pubSub.Close()
	pubSub = brokerstracing.NewPubSub(httpServerConfig, tracer, pubSub)

	if err = consumers.Start(ctx, svcName, pubSub, repo, cfg.ConfigPath, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to create Timescale writer: %s", err))
		exitCode = 1
		return
	}

	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svcName, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Timescale writer service terminated: %s", err))
	}
}

func newService(db *sqlx.DB, logger *slog.Logger) consumers.BlockingConsumer {
	svc := timescale.New(db)
	svc = httpapi.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics("timescale", "message_writer")
	svc = httpapi.MetricsMiddleware(svc, counter, latency)
	return svc
}
