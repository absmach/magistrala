// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains clients main function to start the clients service.
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
	"github.com/absmach/magistrala/channels"
	grpcapi "github.com/absmach/magistrala/channels/api/grpc"
	httpapi "github.com/absmach/magistrala/channels/api/http"
	"github.com/absmach/magistrala/channels/events"
	"github.com/absmach/magistrala/channels/middleware"
	"github.com/absmach/magistrala/channels/postgres"
	pChannels "github.com/absmach/magistrala/channels/private"
	"github.com/absmach/magistrala/channels/tracing"
	grpcChannelsV1 "github.com/absmach/magistrala/internal/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	grpcGroupsV1 "github.com/absmach/magistrala/internal/grpc/groups/v1"
	mglog "github.com/absmach/magistrala/logger"
	authsvcAuthn "github.com/absmach/magistrala/pkg/authn/authsvc"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	authsvcAuthz "github.com/absmach/magistrala/pkg/authz/authsvc"
	"github.com/absmach/magistrala/pkg/grpcclient"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/policies/spicedb"
	pg "github.com/absmach/magistrala/pkg/postgres"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	grpcserver "github.com/absmach/magistrala/pkg/server/grpc"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/sid"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	svcName          = "channels"
	envPrefixDB      = "MG_CHANNELS_DB_"
	envPrefixHTTP    = "MG_CHANNELS_HTTP_"
	envPrefixGRPC    = "MG_CHANNELS_GRPC_"
	envPrefixAuth    = "MG_AUTH_GRPC_"
	envPrefixClients = "MG_CLIENTS_AUTH_GRPC_"
	envPrefixGroups  = "MG_GROUPS_GRPC_"
	defDB            = "channels"
	defSvcHTTPPort   = "9005"
	defSvcGRPCPort   = "7005"
)

type config struct {
	LogLevel            string  `env:"MG_CHANNELS_LOG_LEVEL"           envDefault:"info"`
	InstanceID          string  `env:"MG_CHANNELS_INSTANCE_ID"         envDefault:""`
	JaegerURL           url.URL `env:"MG_JAEGER_URL"                   envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool    `env:"MG_SEND_TELEMETRY"               envDefault:"true"`
	ESURL               string  `env:"MG_ES_URL"                       envDefault:"nats://localhost:4222"`
	TraceRatio          float64 `env:"MG_JAEGER_TRACE_RATIO"           envDefault:"1.0"`
	SpicedbHost         string  `env:"MG_SPICEDB_HOST"                 envDefault:"localhost"`
	SpicedbPort         string  `env:"MG_SPICEDB_PORT"                 envDefault:"50051"`
	SpicedbPreSharedKey string  `env:"MG_SPICEDB_PRE_SHARED_KEY"       envDefault:"12345678"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new channels configuration
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	var logger *slog.Logger
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

	// Create new database for clients
	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	migrations, err := postgres.Migration()
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	db, err := pgclient.Setup(dbConfig, *migrations)
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

	policyEvaluator, policyService, err := newSpiceDBPolicyServiceEvaluator(cfg, logger)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy service are successfully connected to SpiceDB gRPC server")

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
	logger.Info("AuthN  successfully connected to auth gRPC server " + authnClient.Secure())

	authz, authzClient, err := authsvcAuthz.NewAuthorization(ctx, grpcCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authzClient.Close()
	logger.Info("AuthZ  successfully connected to auth gRPC server " + authzClient.Secure())

	thgrpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&thgrpcCfg, env.Options{Prefix: envPrefixClients}); err != nil {
		logger.Error(fmt.Sprintf("failed to load clients gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	clientsClient, clientsHandler, err := grpcclient.SetupClientsClient(ctx, thgrpcCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to clients gRPC server: %s", err))
		exitCode = 1
		return
	}
	defer clientsHandler.Close()
	logger.Info("Clients gRPC client successfully connected to clients gRPC server " + clientsHandler.Secure())

	groupsgRPCCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&groupsgRPCCfg, env.Options{Prefix: envPrefixGroups}); err != nil {
		logger.Error(fmt.Sprintf("failed to load groups gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	groupsClient, groupsHandler, err := grpcclient.SetupGroupsClient(ctx, groupsgRPCCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to groups gRPC server: %s", err))
		exitCode = 1
		return
	}
	defer groupsHandler.Close()
	logger.Info("Groups gRPC client successfully connected to groups gRPC server " + groupsHandler.Secure())

	svc, psvc, err := newService(ctx, db, dbConfig, authz, policyEvaluator, policyService, cfg.ESURL, tracer, clientsClient, groupsClient, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create services: %s", err))
		exitCode = 1
		return
	}

	grpcServerConfig := server.Config{Port: defSvcGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	registerChannelsServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcChannelsV1.RegisterChannelsServiceServer(srv, grpcapi.NewServer(psvc))
	}

	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerChannelsServer, logger)

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	mux := chi.NewRouter()
	httpSvc := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, authn, mux, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	// Start all servers
	g.Go(func() error {
		return httpSvc.Start()
	})

	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSvc)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}

func newService(ctx context.Context, db *sqlx.DB, dbConfig pgclient.Config, authz mgauthz.Authorization,
	pe policies.Evaluator, ps policies.Service, esURL string, tracer trace.Tracer, clientsClient grpcClientsV1.ClientsServiceClient,
	groupsClient grpcGroupsV1.GroupsServiceClient, logger *slog.Logger,
) (channels.Service, pChannels.Service, error) {
	database := pg.NewDatabase(db, dbConfig, tracer)
	repo := postgres.NewRepository(database)

	idp := uuid.New()
	sidp, err := sid.New()
	if err != nil {
		return nil, nil, err
	}

	svc, err := channels.New(repo, ps, idp, clientsClient, groupsClient, sidp)
	if err != nil {
		return nil, nil, err
	}

	svc, err = events.NewEventStoreMiddleware(ctx, svc, esURL)
	if err != nil {
		return nil, nil, err
	}

	svc = tracing.New(svc, tracer)

	counter, latency := prometheus.MakeMetrics("channels", "api")
	svc = middleware.MetricsMiddleware(svc, counter, latency)

	svc, err = middleware.AuthorizationMiddleware(svc, repo, authz, channels.NewOperationPermissionMap(), channels.NewRolesOperationPermissionMap(), channels.NewExternalOperationPermissionMap())
	if err != nil {
		return nil, nil, err
	}
	svc = middleware.LoggingMiddleware(svc, logger)

	psvc := pChannels.New(repo, pe, ps)
	return svc, psvc, err
}

func newSpiceDBPolicyServiceEvaluator(cfg config, logger *slog.Logger) (policies.Evaluator, policies.Service, error) {
	client, err := authzed.NewClientWithExperimentalAPIs(
		fmt.Sprintf("%s:%s", cfg.SpicedbHost, cfg.SpicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(cfg.SpicedbPreSharedKey),
	)
	if err != nil {
		return nil, nil, err
	}
	ps := spicedb.NewPolicyService(client, logger)

	pe := spicedb.NewPolicyEvaluator(client, logger)
	return pe, ps, nil
}
