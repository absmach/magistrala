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

	callhome "github.com/mainflux/callhome/pkg/client"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authclient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgclient "github.com/mainflux/mainflux/internal/clients/postgres"
	redisclient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	mfgroups "github.com/mainflux/mainflux/internal/groups"
	gapi "github.com/mainflux/mainflux/internal/groups/api"
	gpostgres "github.com/mainflux/mainflux/internal/groups/postgres"
	gtracing "github.com/mainflux/mainflux/internal/groups/tracing"
	"github.com/mainflux/mainflux/internal/postgres"
	"github.com/mainflux/mainflux/internal/server"
	grpcserver "github.com/mainflux/mainflux/internal/server/grpc"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/api"
	grpcapi "github.com/mainflux/mainflux/things/api/grpc"
	httpapi "github.com/mainflux/mainflux/things/api/http"
	thcache "github.com/mainflux/mainflux/things/cache"
	thevents "github.com/mainflux/mainflux/things/events"
	thingspg "github.com/mainflux/mainflux/things/postgres"

	localusers "github.com/mainflux/mainflux/things/standalone"
	ctracing "github.com/mainflux/mainflux/things/tracing"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	svcName            = "things"
	envPrefixDB        = "MF_THINGS_DB_"
	envPrefixCache     = "MF_THINGS_CACHE_"
	envPrefixES        = "MF_THINGS_ES_"
	envPrefixHTTP      = "MF_THINGS_HTTP_"
	envPrefixGRPC      = "MF_THINGS_AUTH_GRPC_"
	defDB              = "things"
	defSvcHTTPPort     = "9000"
	defSvcAuthGRPCPort = "7000"
)

type config struct {
	LogLevel         string `env:"MF_THINGS_LOG_LEVEL"           envDefault:"info"`
	StandaloneID     string `env:"MF_THINGS_STANDALONE_ID"       envDefault:""`
	StandaloneToken  string `env:"MF_THINGS_STANDALONE_TOKEN"    envDefault:""`
	JaegerURL        string `env:"MF_JAEGER_URL"                 envDefault:"http://jaeger:14268/api/traces"`
	CacheKeyDuration string `env:"MF_THINGS_CACHE_KEY_DURATION"  envDefault:"10m"`
	SendTelemetry    bool   `env:"MF_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID       string `env:"MF_THINGS_INSTANCE_ID"        envDefault:""`
	ESURL            string `env:"MF_THINGS_ES_URL"              envDefault:"redis://localhost:6379/0"`
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

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	// Create new database for things
	dbConfig := pgclient.Config{Name: defDB}
	if err := dbConfig.LoadEnv(envPrefixDB); err != nil {
		logger.Fatal(err.Error())
	}

	tm := thingspg.Migration()
	gm := gpostgres.Migration()
	tm.Migrations = append(tm.Migrations, gm.Migrations...)
	db, err := pgclient.SetupWithConfig(envPrefixDB, *tm, dbConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaegerclient.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID)
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
	cacheclient, err := redisclient.Setup(envPrefixCache)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer cacheclient.Close()

	// Setup new redis event store client
	esclient, err := redisclient.Setup(envPrefixES)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer esclient.Close()

	var auth mainflux.AuthServiceClient

	switch cfg.StandaloneID != "" && cfg.StandaloneToken != "" {
	case true:
		auth = localusers.NewAuthService(cfg.StandaloneID, cfg.StandaloneToken)
		logger.Info("Using standalone auth service")
	default:
		authServiceClient, authHandler, err := authclient.Setup(svcName)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authHandler.Close()
		auth = authServiceClient
		logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())
	}

	csvc, gsvc, err := newService(ctx, db, dbConfig, auth, cacheclient, esclient, cfg.CacheKeyDuration, cfg.ESURL, tracer, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create services: %s", err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	mux := chi.NewRouter()
	httpSvc := httpserver.New(ctx, cancel, "things-clients", httpServerConfig, httpapi.MakeHandler(csvc, gsvc, mux, logger, cfg.InstanceID), logger)

	grpcServerConfig := server.Config{Port: defSvcAuthGRPCPort}
	if err := env.Parse(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	regiterAuthzServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		mainflux.RegisterAuthzServiceServer(srv, grpcapi.NewServer(csvc))
	}
	gs := grpcserver.New(ctx, cancel, svcName, grpcServerConfig, regiterAuthzServer, logger)

	if cfg.SendTelemetry {
		chc := callhome.New(svcName, mainflux.Version, logger, cancel)
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

func newService(ctx context.Context, db *sqlx.DB, dbConfig pgclient.Config, auth mainflux.AuthServiceClient, cacheClient *redis.Client, esClient *redis.Client, keyDuration, esURL string, tracer trace.Tracer, logger mflog.Logger) (things.Service, groups.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	cRepo := thingspg.NewRepository(database)
	gRepo := gpostgres.New(database)

	idp := uuid.New()

	kDuration, err := time.ParseDuration(keyDuration)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse cache key duration: %s", err.Error()))
	}

	thingCache := thcache.NewCache(cacheClient, kDuration)

	csvc := things.NewService(auth, cRepo, gRepo, thingCache, idp)
	gsvc := mfgroups.NewService(gRepo, idp, auth)

	csvc, err = thevents.NewEventStoreMiddleware(ctx, csvc, esURL)
	if err != nil {
		return nil, nil, err
	}

	csvc = ctracing.New(csvc, tracer)
	csvc = api.LoggingMiddleware(csvc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	csvc = api.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = internal.MakeMetrics(fmt.Sprintf("%s_groups", svcName), "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)

	return csvc, gsvc, err
}
