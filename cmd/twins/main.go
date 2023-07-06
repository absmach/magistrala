// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains twins main function to start the twins service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	mongoClient "github.com/mainflux/mainflux/internal/clients/mongo"
	redisClient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	pstracing "github.com/mainflux/mainflux/pkg/messaging/tracing"
	"github.com/mainflux/mainflux/pkg/uuid"
	localusers "github.com/mainflux/mainflux/things/clients/standalone"
	"github.com/mainflux/mainflux/twins"
	"github.com/mainflux/mainflux/twins/api"
	twapi "github.com/mainflux/mainflux/twins/api/http"
	twmongodb "github.com/mainflux/mainflux/twins/mongodb"
	rediscache "github.com/mainflux/mainflux/twins/redis"
	"github.com/mainflux/mainflux/twins/tracing"
	"github.com/mainflux/mainflux/users/policies"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "twins"
	queue          = "twins"
	envPrefix      = "MF_TWINS_"
	envPrefixHttp  = "MF_TWINS_HTTP_"
	envPrefixCache = "MF_TWINS_CACHE_"
	defSvcHttpPort = "9018"
)

type config struct {
	LogLevel        string `env:"MF_TWINS_LOG_LEVEL"          envDefault:"info"`
	StandaloneID    string `env:"MF_TWINS_STANDALONE_ID"      envDefault:""`
	StandaloneToken string `env:"MF_TWINS_STANDALONE_TOKEN"   envDefault:""`
	ChannelID       string `env:"MF_TWINS_CHANNEL_ID"         envDefault:""`
	BrokerURL       string `env:"MF_BROKER_URL"               envDefault:"nats://localhost:4222"`
	JaegerURL       string `env:"MF_JAEGER_URL"               envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry   bool   `env:"MF_SEND_TELEMETRY"           envDefault:"true"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	cacheClient, err := redisClient.Setup(envPrefixCache)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer cacheClient.Close()

	db, err := mongoClient.Setup(envPrefix)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to setup postgres database : %s", err))
	}

	tp, err := jaegerClient.NewProvider("twins_db", cfg.JaegerURL)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to init Jaeger: %s", err))
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	var auth policies.AuthServiceClient
	switch cfg.StandaloneID != "" && cfg.StandaloneToken != "" {
	case true:
		auth = localusers.NewAuthService(cfg.StandaloneID, cfg.StandaloneToken)
	default:
		authServiceClient, authHandler, err := authClient.Setup(envPrefix, svcName, cfg.JaegerURL)
		if err != nil {
			logger.Fatal(err.Error())
		}
		defer authHandler.Close()
		auth = authServiceClient
		logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())
	}

	pubSub, err := brokers.NewPubSub(cfg.BrokerURL, queue, logger)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to connect to message broker: %s", err))
	}
	pubSub = pstracing.NewPubSub(tracer, pubSub)
	defer pubSub.Close()

	svc := newService(ctx, svcName, pubSub, cfg.ChannelID, auth, tracer, db, cacheClient, logger)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Fatal(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, twapi.MakeHandler(svc, logger), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Twins service terminated: %s", err))
	}
}

func newService(ctx context.Context, id string, ps messaging.PubSub, chanID string, users policies.AuthServiceClient, tracer trace.Tracer, db *mongo.Database, cacheClient *redis.Client, logger mflog.Logger) twins.Service {
	twinRepo := twmongodb.NewTwinRepository(db)
	twinRepo = tracing.TwinRepositoryMiddleware(tracer, twinRepo)

	stateRepo := twmongodb.NewStateRepository(db)
	stateRepo = tracing.StateRepositoryMiddleware(tracer, stateRepo)

	idProvider := uuid.New()
	twinCache := rediscache.NewTwinCache(cacheClient)
	twinCache = tracing.TwinCacheMiddleware(tracer, twinCache)

	svc := twins.New(ps, users, twinRepo, twinCache, stateRepo, idProvider, chanID, logger)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	err := ps.Subscribe(ctx, id, brokers.SubjectAllChannels, handle(ctx, logger, chanID, svc))
	if err != nil {
		logger.Fatal(err.Error())
	}
	return svc
}

func handle(ctx context.Context, logger mflog.Logger, chanID string, svc twins.Service) handlerFunc {
	return func(msg *messaging.Message) error {
		if msg.Channel == chanID {
			return nil
		}

		if err := svc.SaveStates(ctx, msg); err != nil {
			logger.Error(fmt.Sprintf("State save failed: %s", err))
			return err
		}

		return nil
	}
}

type handlerFunc func(msg *messaging.Message) error

func (h handlerFunc) Handle(msg *messaging.Message) error {
	return h(msg)
}

func (h handlerFunc) Cancel() error {
	return nil
}
