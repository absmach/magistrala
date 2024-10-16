// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains invitations main function to start the invitations service.
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
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/invitations/api"
	"github.com/absmach/magistrala/invitations/middleware"
	invitationspg "github.com/absmach/magistrala/invitations/postgres"
	mglog "github.com/absmach/magistrala/logger"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authsvcAuthn "github.com/absmach/magistrala/pkg/authn/authsvc"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	authsvcAuthz "github.com/absmach/magistrala/pkg/authz/authsvc"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/postgres"
	clientspg "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/pkg/server"
	"github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "invitations"
	envPrefixDB    = "MG_INVITATIONS_DB_"
	envPrefixHTTP  = "MG_INVITATIONS_HTTP_"
	envPrefixAuth  = "MG_AUTH_GRPC_"
	defDB          = "invitations"
	defSvcHTTPPort = "9020"
)

type config struct {
	LogLevel      string  `env:"MG_INVITATIONS_LOG_LEVEL"      envDefault:"info"`
	UsersURL      string  `env:"MG_USERS_URL"                  envDefault:"http://localhost:9002"`
	DomainsURL    string  `env:"MG_DOMAINS_URL"                envDefault:"http://localhost:8189"`
	InstanceID    string  `env:"MG_INVITATIONS_INSTANCE_ID"    envDefault:""`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"                 envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"         envDefault:"1.0"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"             envDefault:"true"`
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

	dbConfig := clientspg.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s database configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	db, err := clientspg.Setup(dbConfig, *invitationspg.Migration())
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	authClientCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&authClientCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth configuration : %s", err.Error()))
		exitCode = 1
		return
	}
	tokenClient, tokenHandler, err := grpcclient.SetupTokenClient(ctx, authClientCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer tokenHandler.Close()
	logger.Info("Token service client successfully connected to auth gRPC server " + tokenHandler.Secure())

	authn, authnHandler, err := authsvcAuthn.NewAuthentication(ctx, authClientCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authnHandler.Close()
	logger.Info("Authn successfully connected to auth gRPC server " + authnHandler.Secure())

	authz, authzHandler, err := authsvcAuthz.NewAuthorization(ctx, authClientCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authzHandler.Close()
	logger.Info("Authz successfully connected to auth gRPC server " + authzHandler.Secure())

	tp, err := jaeger.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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

	svc, err := newService(db, dbConfig, authn, authz, tokenClient, tracer, cfg, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s service: %s", svcName, err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	httpSvr := http.NewServer(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return httpSvr.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSvr)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}

func newService(db *sqlx.DB, dbConfig clientspg.Config, authn mgauthn.Authentication, authz mgauthz.Authorization, token magistrala.TokenServiceClient, tracer trace.Tracer, conf config, logger *slog.Logger) (invitations.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	repo := invitationspg.NewRepository(database)

	config := mgsdk.Config{
		UsersURL:   conf.UsersURL,
		DomainsURL: conf.DomainsURL,
	}
	sdk := mgsdk.NewSDK(config)

	svc := invitations.NewService(authn, authz, token, repo, sdk)
	svc = middleware.Tracing(svc, tracer)
	svc = middleware.Logging(logger, svc)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.Metrics(counter, latency, svc)

	return svc, nil
}
