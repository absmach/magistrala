// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains websocket-adapter main function to start the websocket-adapter service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"

	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	thingsClient "github.com/mainflux/mainflux/internal/clients/grpc/things"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	pstracing "github.com/mainflux/mainflux/pkg/messaging/tracing"
	"github.com/mainflux/mainflux/things/policies"
	adapter "github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/api"
	"github.com/mainflux/mainflux/ws/tracing"
)

const (
	svcName        = "ws-adapter"
	envPrefix      = "MF_WS_ADAPTER_"
	envPrefixHttp  = "MF_WS_ADAPTER_HTTP_"
	defSvcHttpPort = "8190"
)

type config struct {
	LogLevel      string `env:"MF_WS_ADAPTER_LOG_LEVEL"   envDefault:"info"`
	BrokerURL     string `env:"MF_BROKER_URL"             envDefault:"nats://localhost:4222"`
	JaegerURL     string `env:"MF_JAEGER_URL"             envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry bool   `env:"MF_SEND_TELEMETRY"         envDefault:"true"`
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

	tc, tcHandler, err := thingsClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer internal.Close(logger, tcHandler)
	logger.Info("Successfully connected to things grpc server " + tcHandler.Secure())

	tp, err := jaegerClient.NewProvider(svcName, cfg.JaegerURL)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to init Jaeger: %s", err))
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	nps, err := brokers.NewPubSub(cfg.BrokerURL, "", logger)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to connect to message broker: %s", err))
	}
	nps = pstracing.NewPubSub(tracer, nps)
	defer nps.Close()

	svc := newService(tc, nps, logger, tracer)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Fatal(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("WS adapter service terminated: %s", err))
	}
}

func newService(tc policies.ThingsServiceClient, nps messaging.PubSub, logger mflog.Logger, tracer trace.Tracer) adapter.Service {
	svc := adapter.New(tc, nps)
	svc = tracing.New(tracer, svc)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("ws_adapter", "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	return svc
}
