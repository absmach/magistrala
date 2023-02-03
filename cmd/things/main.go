// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	grpcserver "github.com/mainflux/mainflux/internal/server/grpc"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/api"
	authgrpcapi "github.com/mainflux/mainflux/things/api/auth/grpc"
	authhttpapi "github.com/mainflux/mainflux/things/api/auth/http"
	thhttpapi "github.com/mainflux/mainflux/things/api/things/http"
	thingsPg "github.com/mainflux/mainflux/things/postgres"
	rediscache "github.com/mainflux/mainflux/things/redis"
	"github.com/mainflux/mainflux/things/tracing"
	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const (
	svcName            = "things"
	envPrefix          = "MF_THINGS_"
	envPrefixCache     = "MF_THINGS_CACHE_"
	envPrefixES        = "MF_THINGS_ES_"
	envPrefixHttp      = "MF_THINGS_HTTP_"
	envPrefixAuthHttp  = "MF_THINGS_AUTH_HTTP_"
	envPrefixAuthGrpc  = "MF_THINGS_AUTH_GRPC_"
	defDB              = "things"
	defSvcHttpPort     = "8182"
	defSvcAuthHttpPort = "8989"
	defSvcAuthGrpcPort = "8181"
)

type config struct {
	LogLevel        string `env:"MF_THINGS_LOG_LEVEL"          envDefault:"info"`
	StandaloneEmail string `env:"MF_THINGS_STANDALONE_EMAIL"   envDefault:""`
	StandaloneToken string `env:"MF_THINGS_STANDALONE_TOKEN"   envDefault:""`
	JaegerURL       string `env:"MF_JAEGER_URL"                envDefault:"localhost:6831"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new things configuration
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s service configuration : %s", svcName, err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Create new database for things
	dbConfig := pgClient.Config{Name: defDB}
	db, err := pgClient.SetupWithConfig(envPrefix, *thingsPg.Migration(), dbConfig)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer db.Close()

	// Setup new redis cache client
	cacheClient, err := redisClient.Setup(envPrefixCache)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer cacheClient.Close()

	// Setup new redis event store client
	esClient, err := redisClient.Setup(envPrefixES)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer esClient.Close()

	// Setup new auth grpc client
	auth, authHandler, err := authClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		log.Fatal(err)
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	// Create tracer for things database
	dbTracer, dbCloser, err := jaegerClient.NewTracer("things_db", cfg.JaegerURL)
	if err != nil {
		log.Fatalf("failed to init Jaeger: %s", err.Error())
	}
	defer dbCloser.Close()

	// Create tracer for things cache
	cacheTracer, cacheCloser, err := jaegerClient.NewTracer("things_cache", cfg.JaegerURL)
	if err != nil {
		log.Fatalf("failed to init Jaeger: %s", err.Error())
	}
	defer cacheCloser.Close()

	// Create new service
	svc := newService(auth, dbTracer, cacheTracer, db, cacheClient, esClient, logger)

	// Create tracer for HTTP handler things
	thingsTracer, thingsCloser, err := jaegerClient.NewTracer("things", cfg.JaegerURL)
	if err != nil {
		log.Fatalf("failed to init Jaeger: %s", err.Error())
	}
	defer thingsCloser.Close()

	// Create new HTTP server
	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s gRPC server configuration : %s", svcName, err.Error())
	}
	hs1 := httpserver.New(ctx, cancel, "thing-http", httpServerConfig, thhttpapi.MakeHandler(thingsTracer, svc, logger), logger)

	// Create new things auth http server
	authHttpServerConfig := server.Config{Port: defSvcAuthHttpPort}
	if err := env.Parse(&authHttpServerConfig, env.Options{Prefix: envPrefixAuthHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s gRPC server configuration : %s", svcName, err.Error())
	}
	hs2 := httpserver.New(ctx, cancel, "auth-http", authHttpServerConfig, authhttpapi.MakeHandler(thingsTracer, svc, logger), logger)

	// Create new grpc server
	registerThingsServiceServer := func(srv *grpc.Server) {
		mainflux.RegisterThingsServiceServer(srv, authgrpcapi.NewServer(thingsTracer, svc))

	}
	grpcServerConfig := server.Config{Port: defSvcAuthGrpcPort}
	if err := env.Parse(&grpcServerConfig, env.Options{Prefix: envPrefixAuthGrpc, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s gRPC server configuration : %s", svcName, err.Error())
	}
	gs := grpcserver.New(ctx, cancel, svcName, grpcServerConfig, registerThingsServiceServer, logger)

	//Start all servers
	g.Go(func() error {
		return hs1.Start()
	})
	g.Go(func() error {
		return hs2.Start()
	})
	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs1, hs2, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Things service terminated: %s", err))
	}
}

func newService(auth mainflux.AuthServiceClient, dbTracer opentracing.Tracer, cacheTracer opentracing.Tracer, db *sqlx.DB, cacheClient *redis.Client, esClient *redis.Client, logger logger.Logger) things.Service {
	database := thingsPg.NewDatabase(db)

	thingsRepo := thingsPg.NewThingRepository(database)
	thingsRepo = tracing.ThingRepositoryMiddleware(dbTracer, thingsRepo)

	channelsRepo := thingsPg.NewChannelRepository(database)
	channelsRepo = tracing.ChannelRepositoryMiddleware(dbTracer, channelsRepo)

	chanCache := rediscache.NewChannelCache(cacheClient)
	chanCache = tracing.ChannelCacheMiddleware(cacheTracer, chanCache)

	thingCache := rediscache.NewThingCache(cacheClient)
	thingCache = tracing.ThingCacheMiddleware(cacheTracer, thingCache)
	idProvider := uuid.New()

	svc := things.New(auth, thingsRepo, channelsRepo, chanCache, thingCache, idProvider)
	svc = rediscache.NewEventStoreMiddleware(svc, esClient)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
