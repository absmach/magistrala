// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains things main function to start the things service.
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
	redisclient "github.com/absmach/magistrala/internal/clients/redis"
	mggroups "github.com/absmach/magistrala/internal/groups"
	gapi "github.com/absmach/magistrala/internal/groups/api"
	gevents "github.com/absmach/magistrala/internal/groups/events"
	gpostgres "github.com/absmach/magistrala/internal/groups/postgres"
	gtracing "github.com/absmach/magistrala/internal/groups/tracing"
	mgpolicy "github.com/absmach/magistrala/internal/policy"
	mglog "github.com/absmach/magistrala/logger"
	authclient "github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/grpcclient"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/policy"
	"github.com/absmach/magistrala/pkg/postgres"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	grpcserver "github.com/absmach/magistrala/pkg/server/grpc"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/things"
	"github.com/absmach/magistrala/things/api"
	grpcapi "github.com/absmach/magistrala/things/api/grpc"
	httpapi "github.com/absmach/magistrala/things/api/http"
	thcache "github.com/absmach/magistrala/things/cache"
	thevents "github.com/absmach/magistrala/things/events"
	thingspg "github.com/absmach/magistrala/things/postgres"
	localusers "github.com/absmach/magistrala/things/standalone"
	ctracing "github.com/absmach/magistrala/things/tracing"
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
	svcName            = "things"
	envPrefixDB        = "MG_THINGS_DB_"
	envPrefixHTTP      = "MG_THINGS_HTTP_"
	envPrefixGRPC      = "MG_THINGS_AUTH_GRPC_"
	envPrefixAuth      = "MG_AUTH_GRPC_"
	defDB              = "things"
	defSvcHTTPPort     = "9000"
	defSvcAuthGRPCPort = "7000"

	streamID = "magistrala.things"
)

type config struct {
	LogLevel            string        `env:"MG_THINGS_LOG_LEVEL"           envDefault:"info"`
	StandaloneID        string        `env:"MG_THINGS_STANDALONE_ID"       envDefault:""`
	StandaloneToken     string        `env:"MG_THINGS_STANDALONE_TOKEN"    envDefault:""`
	JaegerURL           url.URL       `env:"MG_JAEGER_URL"                 envDefault:"http://localhost:4318/v1/traces"`
	CacheKeyDuration    time.Duration `env:"MG_THINGS_CACHE_KEY_DURATION"  envDefault:"10m"`
	SendTelemetry       bool          `env:"MG_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID          string        `env:"MG_THINGS_INSTANCE_ID"         envDefault:""`
	ESURL               string        `env:"MG_ES_URL"                     envDefault:"nats://localhost:4222"`
	CacheURL            string        `env:"MG_THINGS_CACHE_URL"           envDefault:"redis://localhost:6379/0"`
	TraceRatio          float64       `env:"MG_JAEGER_TRACE_RATIO"         envDefault:"1.0"`
	SpicedbHost         string        `env:"MG_SPICEDB_HOST"               envDefault:"localhost"`
	SpicedbPort         string        `env:"MG_SPICEDB_PORT"               envDefault:"50051"`
	SpicedbSchemaFile   string        `env:"MG_SPICEDB_SCHEMA_FILE"        envDefault:"./docker/spicedb/schema.zed"`
	SpicedbPreSharedKey string        `env:"MG_SPICEDB_PRE_SHARED_KEY"     envDefault:"12345678"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new things configuration
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

	// Create new database for things
	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	tm := thingspg.Migration()
	gm := gpostgres.Migration()
	tm.Migrations = append(tm.Migrations, gm.Migrations...)
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

	var (
		authClient   authclient.AuthClient
		policyClient policy.PolicyClient
	)
	switch cfg.StandaloneID != "" && cfg.StandaloneToken != "" {
	case true:
		authClient = localusers.NewAuthClient(cfg.StandaloneID, cfg.StandaloneToken)
		policyClient = localusers.NewPolicyClient(cfg.StandaloneID, cfg.StandaloneToken)
		logger.Info("Using standalone auth service")
	default:
		clientConfig := grpcclient.Config{}
		if err := env.ParseWithOptions(&clientConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
			logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
			exitCode = 1
			return
		}

		authServiceClient, authHandler, err := grpcclient.SetupAuthClient(ctx, clientConfig)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authHandler.Close()
		authClient = authServiceClient
		logger.Info("AuthService gRPC client successfully connected to auth gRPC server " + authHandler.Secure())

		policyClient, err = newPolicyClient(cfg, logger)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		logger.Info("Policy client successfully connected to spicedb gRPC server")
	}

	csvc, gsvc, err := newService(ctx, db, dbConfig, authClient, policyClient, cacheclient, cfg.CacheKeyDuration, cfg.ESURL, tracer, logger)
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
	httpSvc := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(csvc, gsvc, mux, logger, cfg.InstanceID), logger)

	grpcServerConfig := server.Config{Port: defSvcAuthGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	registerThingsServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		magistrala.RegisterAuthzServiceServer(srv, grpcapi.NewServer(csvc))
	}
	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerThingsServer, logger)

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

func newService(ctx context.Context, db *sqlx.DB, dbConfig pgclient.Config, authClient authclient.AuthClient, policyClient policy.PolicyClient, cacheClient *redis.Client, keyDuration time.Duration, esURL string, tracer trace.Tracer, logger *slog.Logger) (things.Service, groups.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	cRepo := thingspg.NewRepository(database)
	gRepo := gpostgres.New(database)

	idp := uuid.New()

	thingCache := thcache.NewCache(cacheClient, keyDuration)

	csvc := things.NewService(authClient, policyClient, cRepo, gRepo, thingCache, idp)
	gsvc := mggroups.NewService(gRepo, idp, authClient, policyClient)

	csvc, err := thevents.NewEventStoreMiddleware(ctx, csvc, esURL)
	if err != nil {
		return nil, nil, err
	}

	gsvc, err = gevents.NewEventStoreMiddleware(ctx, gsvc, esURL, streamID)
	if err != nil {
		return nil, nil, err
	}

	csvc = ctracing.New(csvc, tracer)
	csvc = api.LoggingMiddleware(csvc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	csvc = api.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = prometheus.MakeMetrics(fmt.Sprintf("%s_groups", svcName), "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)

	return csvc, gsvc, err
}

func newPolicyClient(cfg config, logger *slog.Logger) (policy.PolicyClient, error) {
	client, err := authzed.NewClientWithExperimentalAPIs(
		fmt.Sprintf("%s:%s", cfg.SpicedbHost, cfg.SpicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(cfg.SpicedbPreSharedKey),
	)
	if err != nil {
		return nil, err
	}
	policyClient := mgpolicy.NewPolicyClient(client, logger)

	return policyClient, nil
}
