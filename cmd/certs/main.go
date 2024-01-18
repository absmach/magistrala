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
	vault "github.com/absmach/magistrala/certs/pki"
	certspg "github.com/absmach/magistrala/certs/postgres"
	"github.com/absmach/magistrala/certs/tracing"
	"github.com/absmach/magistrala/internal"
	jaegerclient "github.com/absmach/magistrala/internal/clients/jaeger"
	pgclient "github.com/absmach/magistrala/internal/clients/postgres"
	"github.com/absmach/magistrala/internal/postgres"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/auth"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v10"
	"github.com/jmoiron/sqlx"
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
	ThingsURL     string  `env:"MG_THINGS_URL"             envDefault:"http://localhost:9000"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"             envDefault:"http://localhost:14268/api/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"         envDefault:"true"`
	InstanceID    string  `env:"MG_CERTS_INSTANCE_ID"      envDefault:""`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"     envDefault:"1.0"`

	// Sign and issue certificates without 3rd party PKI
	SignCAPath    string `env:"MG_CERTS_SIGN_CA_PATH"        envDefault:"ca.crt"`
	SignCAKeyPath string `env:"MG_CERTS_SIGN_CA_KEY_PATH"    envDefault:"ca.key"`

	// 3rd party PKI API access settings
	PkiHost  string `env:"MG_CERTS_VAULT_HOST"    envDefault:""`
	PkiPath  string `env:"MG_VAULT_PKI_INT_PATH"  envDefault:"pki_int"`
	PkiRole  string `env:"MG_VAULT_CA_ROLE_NAME"  envDefault:"magistrala"`
	PkiToken string `env:"MG_VAULT_TOKEN"         envDefault:""`
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

	if cfg.PkiHost == "" {
		logger.Error("No host specified for PKI engine")
		exitCode = 1
		return
	}

	pkiclient, err := vault.NewVaultClient(cfg.PkiToken, cfg.PkiHost, cfg.PkiPath, cfg.PkiRole)
	if err != nil {
		logger.Error("failed to configure client for PKI engine")
		exitCode = 1
		return
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
	}
	db, err := pgclient.Setup(dbConfig, *certspg.Migration())
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	authConfig := auth.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authClient, authHandler, err := auth.Setup(authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()

	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

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

	svc := newService(authClient, db, tracer, logger, cfg, dbConfig, pkiclient)

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger, cfg.InstanceID), logger)

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

func newService(authClient magistrala.AuthServiceClient, db *sqlx.DB, tracer trace.Tracer, logger *slog.Logger, cfg config, dbConfig pgclient.Config, pkiAgent vault.Agent) certs.Service {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	certsRepo := certspg.NewRepository(database, logger)
	config := mgsdk.Config{
		ThingsURL: cfg.ThingsURL,
	}
	sdk := mgsdk.NewSDK(config)
	svc := certs.New(authClient, certsRepo, sdk, pkiAgent)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc
}
