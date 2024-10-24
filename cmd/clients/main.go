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
	"time"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/clients"
	grpcapi "github.com/absmach/magistrala/clients/api/grpc"
	httpapi "github.com/absmach/magistrala/clients/api/http"
	"github.com/absmach/magistrala/clients/cache"
	"github.com/absmach/magistrala/clients/events"
	"github.com/absmach/magistrala/clients/middleware"
	"github.com/absmach/magistrala/clients/postgres"
	pClients "github.com/absmach/magistrala/clients/private"
	"github.com/absmach/magistrala/clients/tracing"
	redisclient "github.com/absmach/magistrala/internal/clients/redis"
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
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	svcName            = "clients"
	envPrefixDB        = "MG_CLIENTS_DB_"
	envPrefixHTTP      = "MG_CLIENTS_HTTP_"
	envPrefixGRPC      = "MG_CLIENTS_AUTH_GRPC_"
	envPrefixAuth      = "MG_AUTH_GRPC_"
	envPrefixChannels  = "MG_CHANNELS_GRPC_"
	envPrefixGroups    = "MG_GROUPS_GRPC_"
	defDB              = "clients"
	defSvcHTTPPort     = "9000"
	defSvcAuthGRPCPort = "7000"
)

type config struct {
	InstanceID          string        `env:"MG_CLIENTS_INSTANCE_ID"        envDefault:""`
	LogLevel            string        `env:"MG_CLIENTS_LOG_LEVEL"          envDefault:"info"`
	StandaloneID        string        `env:"MG_CLIENTS_STANDALONE_ID"      envDefault:""`
	StandaloneToken     string        `env:"MG_CLIENTS_STANDALONE_TOKEN"   envDefault:""`
	CacheURL            string        `env:"MG_CLIENTS_CACHE_URL"          envDefault:"redis://localhost:6379/0"`
	CacheKeyDuration    time.Duration `env:"MG_CLIENTS_CACHE_KEY_DURATION" envDefault:"10m"`
	JaegerURL           url.URL       `env:"MG_JAEGER_URL"                 envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool          `env:"MG_SEND_TELEMETRY"             envDefault:"true"`
	ESURL               string        `env:"MG_ES_URL"                     envDefault:"nats://localhost:4222"`
	TraceRatio          float64       `env:"MG_JAEGER_TRACE_RATIO"         envDefault:"1.0"`
	SpicedbHost         string        `env:"MG_SPICEDB_HOST"               envDefault:"localhost"`
	SpicedbPort         string        `env:"MG_SPICEDB_PORT"               envDefault:"50051"`
	SpicedbPreSharedKey string        `env:"MG_SPICEDB_PRE_SHARED_KEY"     envDefault:"12345678"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new clients configuration
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
	tm, err := postgres.Migration()
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	db, err := pgclient.Setup(dbConfig, *tm)
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

	// Setup new redis cache client
	cacheclient, err := redisclient.Connect(cfg.CacheURL)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer cacheclient.Close()

	policyEvaluator, policyService, err := newSpiceDBPolicyServiceEvaluator(cfg, logger)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy evaluator and Policy manager are successfully connected to SpiceDB gRPC server")

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
	logger.Info("AuthZ  successfully connected to auth gRPC server " + authnClient.Secure())

	chgrpccfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&chgrpccfg, env.Options{Prefix: envPrefixChannels}); err != nil {
		logger.Error(fmt.Sprintf("failed to load channels gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	channelsgRPC, channelsClient, err := grpcclient.SetupChannelsClient(ctx, chgrpccfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Channels gRPC client successfully connected to channels gRPC server " + channelsClient.Secure())
	defer channelsClient.Close()

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

	svc, psvc, err := newService(ctx, db, dbConfig, authz, policyEvaluator, policyService, cacheclient, cfg.CacheKeyDuration, cfg.ESURL, channelsgRPC, groupsClient, tracer, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create services: %s", err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	mux := chi.NewRouter()
	httpSvc := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, authn, mux, logger, cfg.InstanceID), logger)

	grpcServerConfig := server.Config{Port: defSvcAuthGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	registerClientsServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcClientsV1.RegisterClientsServiceServer(srv, grpcapi.NewServer(psvc))
	}
	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerClientsServer, logger)

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

func newService(ctx context.Context, db *sqlx.DB, dbConfig pgclient.Config, authz mgauthz.Authorization, pe policies.Evaluator, ps policies.Service, cacheClient *redis.Client, keyDuration time.Duration, esURL string, channels grpcChannelsV1.ChannelsServiceClient, groups grpcGroupsV1.GroupsServiceClient, tracer trace.Tracer, logger *slog.Logger) (clients.Service, pClients.Service, error) {
	database := pg.NewDatabase(db, dbConfig, tracer)
	repo := postgres.NewRepository(database)

	idp := uuid.New()
	sidp, err := sid.New()
	if err != nil {
		return nil, nil, err
	}

	// Clients service
	cache := cache.NewCache(cacheClient, keyDuration)

	csvc, err := clients.NewService(repo, ps, cache, channels, groups, idp, sidp)
	if err != nil {
		return nil, nil, err
	}

	csvc, err = events.NewEventStoreMiddleware(ctx, csvc, esURL)
	if err != nil {
		return nil, nil, err
	}

	csvc = tracing.New(csvc, tracer)

	counter, latency := prometheus.MakeMetrics(svcName, "api")
	csvc = middleware.MetricsMiddleware(csvc, counter, latency)
	csvc = middleware.MetricsMiddleware(csvc, counter, latency)

	csvc, err = middleware.AuthorizationMiddleware(policies.ClientType, csvc, authz, repo, clients.NewOperationPermissionMap(), clients.NewRolesOperationPermissionMap(), clients.NewExternalOperationPermissionMap())
	if err != nil {
		return nil, nil, err
	}
	csvc = middleware.LoggingMiddleware(csvc, logger)

	isvc := pClients.New(repo, cache, pe, ps)

	return csvc, isvc, err
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
	pe := spicedb.NewPolicyEvaluator(client, logger)
	ps := spicedb.NewPolicyService(client, logger)

	return pe, ps, nil
}
