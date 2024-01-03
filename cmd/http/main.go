// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains http-adapter main function to start the http-adapter service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/absmach/magistrala"
	adapter "github.com/absmach/magistrala/http"
	"github.com/absmach/magistrala/http/api"
	"github.com/absmach/magistrala/internal"
	jaegerclient "github.com/absmach/magistrala/internal/clients/jaeger"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/messaging/handler"
	"github.com/absmach/magistrala/pkg/uuid"
	mproxy "github.com/absmach/mproxy/pkg/http"
	"github.com/absmach/mproxy/pkg/session"
	"github.com/caarlos0/env/v10"
	chclient "github.com/mainflux/callhome/pkg/client"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "http_adapter"
	envPrefix      = "MG_HTTP_ADAPTER_"
	envPrefixAuthz = "MG_THINGS_AUTH_GRPC_"
	defSvcHTTPPort = "80"
	targetHTTPPort = "81"
	targetHTTPHost = "http://localhost"
)

type config struct {
	LogLevel      string  `env:"MG_HTTP_ADAPTER_LOG_LEVEL"   envDefault:"info"`
	BrokerURL     string  `env:"MG_MESSAGE_BROKER_URL"       envDefault:"nats://localhost:4222"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"               envDefault:"http://localhost:14268/api/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID    string  `env:"MG_HTTP_ADAPTER_INSTANCE_ID" envDefault:""`
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
		log.Fatalf("failed to init logger: %s", err)
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
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authConfig := auth.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuthz}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authClient, authHandler, err := auth.SetupAuthz(authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()

	logger.Info("Successfully connected to things grpc server " + authHandler.Secure())

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

	svc := newService(pub, authClient, logger, tracer)
	targetServerCfg := server.Config{Port: targetHTTPPort}

	hs := httpserver.New(ctx, cancel, svcName, targetServerCfg, api.MakeHandler(cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
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

func newService(pub messaging.Publisher, tc magistrala.AuthzServiceClient, logger mglog.Logger, tracer trace.Tracer) session.Handler {
	svc := adapter.NewHandler(pub, logger, tc)
	svc = handler.NewTracing(tracer, svc)
	svc = handler.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = handler.MetricsMiddleware(svc, counter, latency)
	return svc
}

func proxyHTTP(ctx context.Context, cfg server.Config, logger mglog.Logger, sessionHandler session.Handler) error {
	address := fmt.Sprintf("%s:%s", "", cfg.Port)
	target := fmt.Sprintf("%s:%s", targetHTTPHost, targetHTTPPort)
	mp, err := mproxy.NewProxy(address, target, sessionHandler, logger)
	if err != nil {
		return err
	}
	http.HandleFunc("/", mp.Handler)

	errCh := make(chan error)
	switch {
	case cfg.CertFile != "" || cfg.KeyFile != "":
		go func() {
			errCh <- mp.ListenTLS(cfg.CertFile, cfg.KeyFile)
		}()
		logger.Info(fmt.Sprintf("%s service https server listening at %s:%s with TLS cert %s and key %s", svcName, cfg.Host, cfg.Port, cfg.CertFile, cfg.KeyFile))
	default:
		go func() {
			errCh <- mp.Listen()
		}()
		logger.Info(fmt.Sprintf("%s service http server listening at %s:%s without TLS", svcName, cfg.Host, cfg.Port))
	}

	select {
	case <-ctx.Done():
		logger.Info(fmt.Sprintf("proxy HTTP shutdown at %s", target))
		return nil
	case err := <-errCh:
		return err
	}
}
