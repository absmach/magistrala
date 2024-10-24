// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains coap-adapter main function to start the coap-adapter service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/coap"
	"github.com/absmach/magistrala/coap/api"
	"github.com/absmach/magistrala/coap/tracing"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/grpcclient"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	coapserver "github.com/absmach/magistrala/pkg/server/coap"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
)

const (
	svcName           = "coap_adapter"
	envPrefix         = "MG_COAP_ADAPTER_"
	envPrefixHTTP     = "MG_COAP_ADAPTER_HTTP_"
	envPrefixClients   = "MG_CLIENTS_AUTH_GRPC_"
	envPrefixChannels = "MG_CHANNELS_GRPC_"
	defSvcHTTPPort    = "5683"
	defSvcCoAPPort    = "5683"
)

type config struct {
	LogLevel      string  `env:"MG_COAP_ADAPTER_LOG_LEVEL"   envDefault:"info"`
	BrokerURL     string  `env:"MG_MESSAGE_BROKER_URL"       envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"               envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID    string  `env:"MG_COAP_ADAPTER_INSTANCE_ID" envDefault:""`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"       envDefault:"1.0"`
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

	coapServerConfig := server.Config{Port: defSvcCoAPPort}
	if err := env.ParseWithOptions(&coapServerConfig, env.Options{Prefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s CoAP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	clientsClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&clientsClientCfg, env.Options{Prefix: envPrefixClients}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	clientsClient, clientsHandler, err := grpcclient.SetupClientsClient(ctx, clientsClientCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer clientsHandler.Close()

	logger.Info("Clients service gRPC client successfully connected to clients gRPC server " + clientsHandler.Secure())

	channelsClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&channelsClientCfg, env.Options{Prefix: envPrefixChannels}); err != nil {
		logger.Error(fmt.Sprintf("failed to load channels gRPC client configuration : %s", err))
		exitCode = 1
		return
	}

	channelsClient, channelsHandler, err := grpcclient.SetupChannelsClient(ctx, channelsClientCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer channelsHandler.Close()
	logger.Info("Channels service gRPC client successfully connected to channels gRPC server " + channelsHandler.Secure())

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

	nps, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer nps.Close()
	nps = brokerstracing.NewPubSub(coapServerConfig, tracer, nps)

	svc := coap.New(clientsClient, channelsClient, nps)

	svc = tracing.New(tracer, svc)

	svc = api.LoggingMiddleware(svc, logger)

	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(cfg.InstanceID), logger)

	cs := coapserver.NewServer(ctx, cancel, svcName, coapServerConfig, api.MakeCoAPHandler(svc, logger), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})
	g.Go(func() error {
		return cs.Start()
	})
	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs, cs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("CoAP adapter service terminated: %s", err))
	}
}
