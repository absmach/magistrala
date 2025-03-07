// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains http-adapter main function to start the http-adapter service.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/mgate"
	mgatehttp "github.com/absmach/mgate/pkg/http"
	"github.com/absmach/mgate/pkg/session"
	"github.com/absmach/supermq"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	adapter "github.com/absmach/supermq/http"
	httpapi "github.com/absmach/supermq/http/api"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authn/authsvc"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/supermq/pkg/messaging/brokers/tracing"
	msgevents "github.com/absmach/supermq/pkg/messaging/events"
	"github.com/absmach/supermq/pkg/messaging/handler"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName           = "http_adapter"
	envPrefix         = "SMQ_HTTP_ADAPTER_"
	envPrefixClients  = "SMQ_CLIENTS_GRPC_"
	envPrefixChannels = "SMQ_CHANNELS_GRPC_"
	envPrefixAuth     = "SMQ_AUTH_GRPC_"
	defSvcHTTPPort    = "80"
	targetHTTPPort    = "81"
	targetHTTPHost    = "http://localhost"
)

type config struct {
	LogLevel      string  `env:"SMQ_HTTP_ADAPTER_LOG_LEVEL"   envDefault:"info"`
	BrokerURL     string  `env:"SMQ_MESSAGE_BROKER_URL"       envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"SMQ_JAEGER_URL"               envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry bool    `env:"SMQ_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID    string  `env:"SMQ_HTTP_ADAPTER_INSTANCE_ID" envDefault:""`
	TraceRatio    float64 `env:"SMQ_JAEGER_TRACE_RATIO"       envDefault:"1.0"`
	ESURL         string  `env:"SMQ_ES_URL"                   envDefault:"nats://localhost:4222"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
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
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	clientsClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&clientsClientCfg, env.Options{Prefix: envPrefixClients}); err != nil {
		logger.Error(fmt.Sprintf("failed to load clients gRPC client configuration : %s", err))
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

	pub, err := brokers.NewPublisher(ctx, cfg.BrokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pub.Close()
	pub = brokerstracing.NewPublisher(httpServerConfig, tracer, pub)

	pub, err = msgevents.NewPublisherMiddleware(ctx, pub, cfg.ESURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create event store middleware: %s", err))
		exitCode = 1
		return
	}

	svc := newService(pub, authn, clientsClient, channelsClient, logger, tracer)
	targetServerCfg := server.Config{Port: targetHTTPPort}

	hs := httpserver.NewServer(ctx, cancel, svcName, targetServerCfg, httpapi.MakeHandler(logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return proxyHTTP(ctx, httpServerConfig, logger, svc)
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("HTTP adapter service terminated: %s", err))
	}
}

func newService(pub messaging.Publisher, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, logger *slog.Logger, tracer trace.Tracer) session.Handler {
	svc := adapter.NewHandler(pub, authn, clients, channels, logger)
	svc = handler.NewTracing(tracer, svc)
	svc = handler.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = handler.MetricsMiddleware(svc, counter, latency)
	return svc
}

func proxyHTTP(ctx context.Context, cfg server.Config, logger *slog.Logger, sessionHandler session.Handler) error {
	config := mgate.Config{
		Address:    fmt.Sprintf("%s:%s", "", cfg.Port),
		Target:     fmt.Sprintf("%s:%s", targetHTTPHost, targetHTTPPort),
		PathPrefix: "/",
	}
	if cfg.CertFile != "" || cfg.KeyFile != "" {
		tlsCert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return err
		}
		config.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{tlsCert},
		}
	}
	mp, err := mgatehttp.NewProxy(config, sessionHandler, logger)
	if err != nil {
		return err
	}
	http.HandleFunc("/", mp.ServeHTTP)

	errCh := make(chan error)
	switch {
	case cfg.CertFile != "" || cfg.KeyFile != "":
		go func() {
			errCh <- mp.Listen(ctx)
		}()
		logger.Info(fmt.Sprintf("%s service HTTPS server listening at %s:%s with TLS cert %s and key %s", svcName, cfg.Host, cfg.Port, cfg.CertFile, cfg.KeyFile))
	default:
		go func() {
			errCh <- mp.Listen(ctx)
		}()
		logger.Info(fmt.Sprintf("%s service HTTP server listening at %s:%s without TLS", svcName, cfg.Host, cfg.Port))
	}

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy HTTP shutdown at %s", config.Target))
		return nil
	case err := <-errCh:
		return err
	}
}
