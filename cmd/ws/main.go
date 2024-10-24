// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains websocket-adapter main function to start the websocket-adapter service.
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
	grpcChannelsV1 "github.com/absmach/magistrala/internal/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/authn/authsvc"
	"github.com/absmach/magistrala/pkg/grpcclient"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/ws"
	"github.com/absmach/magistrala/ws/api"
	"github.com/absmach/magistrala/ws/tracing"
	"github.com/absmach/mgate/pkg/session"
	"github.com/absmach/mgate/pkg/websockets"
	"github.com/caarlos0/env/v11"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName           = "ws-adapter"
	envPrefixHTTP     = "MG_WS_ADAPTER_HTTP_"
	envPrefixClients  = "MG_CLIENTS_AUTH_GRPC_"
	envPrefixChannels = "MG_CHANNELS_GRPC_"
	envPrefixAuth     = "MG_AUTH_GRPC_"
	defSvcHTTPPort    = "8190"
	targetWSPort      = "8191"
	targetWSHost      = "localhost"
)

type config struct {
	LogLevel      string  `env:"MG_WS_ADAPTER_LOG_LEVEL"    envDefault:"info"`
	BrokerURL     string  `env:"MG_MESSAGE_BROKER_URL"      envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"              envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"          envDefault:"true"`
	InstanceID    string  `env:"MG_WS_ADAPTER_INSTANCE_ID"  envDefault:""`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"      envDefault:"1.0"`
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

	targetServerConfig := server.Config{
		Port: targetWSPort,
		Host: targetWSHost,
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

	authnCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&authnCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC client configuration : %s", err))
		exitCode = 1
		return
	}

	authn, authnHandler, err := authsvc.NewAuthentication(ctx, authnCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authnHandler.Close()
	logger.Info("authn successfully connected to auth gRPC server " + authnHandler.Secure())

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
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
		logger.Error(fmt.Sprintf("Failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer nps.Close()
	nps = brokerstracing.NewPubSub(targetServerConfig, tracer, nps)

	svc := newService(clientsClient, channelsClient, nps, logger, tracer)

	hs := httpserver.NewServer(ctx, cancel, svcName, targetServerConfig, api.MakeHandler(ctx, svc, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		g.Go(func() error {
			return hs.Start()
		})
		handler := ws.NewHandler(nps, logger, authn, clientsClient, channelsClient)
		return proxyWS(ctx, httpServerConfig, targetServerConfig, logger, handler)
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("WS adapter service terminated: %s", err))
	}
}

func newService(clientsClient grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, nps messaging.PubSub, logger *slog.Logger, tracer trace.Tracer) ws.Service {
	svc := ws.New(clientsClient, channels, nps)
	svc = tracing.New(tracer, svc)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics("ws_adapter", "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	return svc
}

func proxyWS(ctx context.Context, hostConfig, targetConfig server.Config, logger *slog.Logger, handler session.Handler) error {
	target := fmt.Sprintf("ws://%s:%s", targetConfig.Host, targetConfig.Port)
	address := fmt.Sprintf("%s:%s", hostConfig.Host, hostConfig.Port)
	wp, err := websockets.NewProxy(address, target, logger, handler)
	if err != nil {
		return err
	}

	errCh := make(chan error)

	go func() {
		if hostConfig.CertFile != "" && hostConfig.KeyFile != "" {
			logger.Info(fmt.Sprintf("ws-adapter service http server listening at %s:%s with TLS", hostConfig.Host, hostConfig.Port))
			errCh <- wp.ListenTLS(hostConfig.CertFile, hostConfig.KeyFile)
		} else {
			logger.Info(fmt.Sprintf("ws-adapter service http server listening at %s:%s without TLS", hostConfig.Host, hostConfig.Port))
			errCh <- wp.Listen()
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy MQTT WS shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}
}
