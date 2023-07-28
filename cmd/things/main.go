// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains things main function to start the things service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-zoo/bone"
	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/postgres"
	"github.com/mainflux/mainflux/internal/server"
	grpcserver "github.com/mainflux/mainflux/internal/server/grpc"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things/clients"
	capi "github.com/mainflux/mainflux/things/clients/api"
	cpostgres "github.com/mainflux/mainflux/things/clients/postgres"
	redisthcache "github.com/mainflux/mainflux/things/clients/redis"
	localusers "github.com/mainflux/mainflux/things/clients/standalone"
	ctracing "github.com/mainflux/mainflux/things/clients/tracing"
	"github.com/mainflux/mainflux/things/groups"
	gapi "github.com/mainflux/mainflux/things/groups/api"
	gpostgres "github.com/mainflux/mainflux/things/groups/postgres"
	redischcache "github.com/mainflux/mainflux/things/groups/redis"
	gtracing "github.com/mainflux/mainflux/things/groups/tracing"
	tpolicies "github.com/mainflux/mainflux/things/policies"
	papi "github.com/mainflux/mainflux/things/policies/api"
	grpcapi "github.com/mainflux/mainflux/things/policies/api/grpc"
	httpapi "github.com/mainflux/mainflux/things/policies/api/http"
	ppostgres "github.com/mainflux/mainflux/things/policies/postgres"
	redispcache "github.com/mainflux/mainflux/things/policies/redis"
	ppracing "github.com/mainflux/mainflux/things/policies/tracing"
	thingsPg "github.com/mainflux/mainflux/things/postgres"
	upolicies "github.com/mainflux/mainflux/users/policies"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	svcName            = "things"
	envPrefix          = "MF_THINGS_"
	envPrefixCache     = "MF_THINGS_CACHE_"
	envPrefixES        = "MF_THINGS_ES_"
	envPrefixHttp      = "MF_THINGS_HTTP_"
	envPrefixAuthGrpc  = "MF_THINGS_AUTH_GRPC_"
	defDB              = "things"
	defSvcHttpPort     = "9000"
	defSvcAuthGrpcPort = "7000"
)

type config struct {
	LogLevel         string `env:"MF_THINGS_LOG_LEVEL"           envDefault:"info"`
	StandaloneID     string `env:"MF_THINGS_STANDALONE_ID"       envDefault:""`
	StandaloneToken  string `env:"MF_THINGS_STANDALONE_TOKEN"    envDefault:""`
	JaegerURL        string `env:"MF_JAEGER_URL"                 envDefault:"http://jaeger:14268/api/traces"`
	CacheKeyDuration string `env:"MF_THINGS_CACHE_KEY_DURATION"  envDefault:"10m"`
	SendTelemetry    bool   `env:"MF_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID       string `env:"MF_THINGS_INSTANCE_ID"        envDefault:""`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new things configuration
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID, err = uuid.New().ID()
		if err != nil {
			log.Fatalf("Failed to generate instanceID: %s", err)
		}
	}

	// Create new database for things
	dbConfig := pgClient.Config{Name: defDB}
	db, err := pgClient.SetupWithConfig(envPrefix, *thingsPg.Migration(), dbConfig)
	if err != nil {
		logger.Fatal(err.Error())
	}
	var exitCode int
	defer mflog.ExitWithError(&exitCode)
	defer db.Close()

	tp, err := jaegerClient.NewProvider(svcName, cfg.JaegerURL, instanceID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	// Setup new redis cache client
	cacheClient, err := redisClient.Setup(envPrefixCache)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer cacheClient.Close()

	// Setup new redis event store client
	esClient, err := redisClient.Setup(envPrefixES)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer esClient.Close()

	var auth upolicies.AuthServiceClient
	switch cfg.StandaloneID != "" && cfg.StandaloneToken != "" {
	case true:
		auth = localusers.NewAuthService(cfg.StandaloneID, cfg.StandaloneToken)
		logger.Info("Using standalone auth service")
	default:
		authServiceClient, authHandler, err := authClient.Setup(envPrefix, svcName)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authHandler.Close()
		auth = authServiceClient
		logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())
	}

	csvc, gsvc, psvc := newService(db, auth, cacheClient, esClient, cfg.CacheKeyDuration, tracer, logger)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	mux := bone.New()
	hsp := httpserver.New(ctx, cancel, "things-policies", httpServerConfig, httpapi.MakeHandler(csvc, psvc, mux, logger), logger)
	hsc := httpserver.New(ctx, cancel, "things-clients", httpServerConfig, capi.MakeHandler(csvc, mux, logger, instanceID), logger)
	hsg := httpserver.New(ctx, cancel, "things-groups", httpServerConfig, gapi.MakeHandler(gsvc, mux, logger), logger)

	registerThingsServiceServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		tpolicies.RegisterAuthServiceServer(srv, grpcapi.NewServer(csvc, psvc))
	}
	grpcServerConfig := server.Config{Port: defSvcAuthGrpcPort}
	if err := env.Parse(&grpcServerConfig, env.Options{Prefix: envPrefixAuthGrpc, AltPrefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	gs := grpcserver.New(ctx, cancel, svcName, grpcServerConfig, registerThingsServiceServer, logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	// Start all servers
	g.Go(func() error {
		return hsp.Start()
	})
	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hsc, hsg, hsp, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}

func newService(db *sqlx.DB, auth upolicies.AuthServiceClient, cacheClient *redis.Client, esClient *redis.Client, keyDuration string, tracer trace.Tracer, logger mflog.Logger) (clients.Service, groups.Service, tpolicies.Service) {
	database := postgres.NewDatabase(db, tracer)
	cRepo := cpostgres.NewRepository(database)
	gRepo := gpostgres.NewRepository(database)
	pRepo := ppostgres.NewRepository(database)

	idp := uuid.New()

	kDuration, err := time.ParseDuration(keyDuration)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse cache key duration: %s", err.Error()))
	}

	policyCache := redispcache.NewCache(cacheClient, kDuration)
	thingCache := redisthcache.NewCache(cacheClient, kDuration)

	psvc := tpolicies.NewService(auth, pRepo, policyCache, idp)
	csvc := clients.NewService(auth, psvc, cRepo, gRepo, thingCache, idp)
	gsvc := groups.NewService(auth, psvc, gRepo, idp)

	csvc = redisthcache.NewEventStoreMiddleware(csvc, esClient)
	gsvc = redischcache.NewEventStoreMiddleware(gsvc, esClient)
	psvc = redispcache.NewEventStoreMiddleware(psvc, esClient)

	csvc = ctracing.New(csvc, tracer)
	csvc = capi.LoggingMiddleware(csvc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	csvc = capi.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = internal.MakeMetrics(fmt.Sprintf("%s_groups", svcName), "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)
	psvc = ppracing.New(psvc, tracer)
	psvc = papi.LoggingMiddleware(psvc, logger)
	counter, latency = internal.MakeMetrics(fmt.Sprintf("%s_policies", svcName), "api")
	psvc = papi.MetricsMiddleware(psvc, counter, latency)

	return csvc, gsvc, psvc
}
