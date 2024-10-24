// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains certs main function to start the certs service.
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
	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/certs/api"
	pki "github.com/absmach/magistrala/certs/pki/amcerts"
	"github.com/absmach/magistrala/certs/tracing"
	mglog "github.com/absmach/magistrala/logger"
	authsvcAuthn "github.com/absmach/magistrala/pkg/authn/authsvc"
	"github.com/absmach/magistrala/pkg/grpcclient"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/prometheus"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/pkg/server"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "certs"
	envPrefixDB    = "MG_CERTS_DB_"
	envPrefixHTTP  = "MG_CERTS_HTTP_"
	envPrefixAuth  = "MG_AUTH_GRPC_"
	defDB          = "certs"
	defSvcHTTPPort = "9019"
)

type config struct {
	LogLevel      string  `env:"MG_CERTS_LOG_LEVEL"        envDefault:"info"`
	ClientsURL    string  `env:"MG_CLIENTS_URL"            envDefault:"http://localhost:9000"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"             envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"         envDefault:"true"`
	InstanceID    string  `env:"MG_CERTS_INSTANCE_ID"      envDefault:""`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"     envDefault:"1.0"`

	// Sign and issue certificates without 3rd party PKI
	SignCAPath    string `env:"MG_CERTS_SIGN_CA_PATH"        envDefault:"ca.crt"`
	SignCAKeyPath string `env:"MG_CERTS_SIGN_CA_KEY_PATH"    envDefault:"ca.key"`

	// Amcerts SDK settings
	SDKHost         string `env:"MG_CERTS_SDK_HOST"             envDefault:""`
	SDKCertsURL     string `env:"MG_CERTS_SDK_CERTS_URL"        envDefault:"http://localhost:9010"`
	TLSVerification bool   `env:"MG_CERTS_SDK_TLS_VERIFICATION" envDefault:"false"`
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

	if cfg.SDKHost == "" {
		logger.Error("No host specified for PKI engine")
		exitCode = 1
		return
	}

	pkiclient, err := pki.NewAgent(cfg.SDKHost, cfg.SDKCertsURL, cfg.TLSVerification)
	if err != nil {
		logger.Error("failed to configure client for PKI engine")
		exitCode = 1
		return
	}

	grpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&grpcCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	authn, authnClient, err := authsvcAuthn.NewAuthentication(ctx, grpcCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authnClient.Close()
	logger.Info("AuthN successfully connected to auth gRPC server " + authnClient.Secure())

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	svc := newService(tracer, logger, cfg, pkiclient)

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, authn, logger, cfg.InstanceID), logger)

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
		logger.Error(fmt.Sprintf("Certs service terminated: %s", err))
	}
}

func newService(tracer trace.Tracer, logger *slog.Logger, cfg config, pkiAgent pki.Agent) certs.Service {
	config := mgsdk.Config{
		ClientsURL: cfg.ClientsURL,
	}
	sdk := mgsdk.NewSDK(config)
	svc := certs.New(sdk, pkiAgent)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc
}
