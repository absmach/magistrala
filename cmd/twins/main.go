// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains twins main function to start the twins service.
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
	"github.com/absmach/magistrala/internal"
	jaegerclient "github.com/absmach/magistrala/internal/clients/jaeger"
	mongoclient "github.com/absmach/magistrala/internal/clients/mongo"
	redisclient "github.com/absmach/magistrala/internal/clients/redis"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/uuid"
	localusers "github.com/absmach/magistrala/things/standalone"
	"github.com/absmach/magistrala/twins"
	"github.com/absmach/magistrala/twins/api"
	twapi "github.com/absmach/magistrala/twins/api/http"
	"github.com/absmach/magistrala/twins/events"
	twmongodb "github.com/absmach/magistrala/twins/mongodb"
	"github.com/absmach/magistrala/twins/tracing"
	"github.com/caarlos0/env/v10"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "twins"
	envPrefixDB    = "MG_TWINS_DB_"
	envPrefixHTTP  = "MG_TWINS_HTTP_"
	envPrefixAuth  = "MG_AUTH_GRPC_"
	defSvcHTTPPort = "9018"
)

type config struct {
	LogLevel        string  `env:"MG_TWINS_LOG_LEVEL"          envDefault:"info"`
	StandaloneID    string  `env:"MG_TWINS_STANDALONE_ID"      envDefault:""`
	StandaloneToken string  `env:"MG_TWINS_STANDALONE_TOKEN"   envDefault:""`
	ChannelID       string  `env:"MG_TWINS_CHANNEL_ID"         envDefault:""`
	BrokerURL       string  `env:"MG_MESSAGE_BROKER_URL"       envDefault:"nats://localhost:4222"`
	JaegerURL       url.URL `env:"MG_JAEGER_URL"               envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry   bool    `env:"MG_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID      string  `env:"MG_TWINS_INSTANCE_ID"        envDefault:""`
	ESURL           string  `env:"MG_ES_URL"                   envDefault:"nats://localhost:4222"`
	CacheURL        string  `env:"MG_TWINS_CACHE_URL"          envDefault:"redis://localhost:6379/0"`
	TraceRatio      float64 `env:"MG_JAEGER_TRACE_RATIO"       envDefault:"1.0"`
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

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	cacheClient, err := redisclient.Connect(cfg.CacheURL)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer cacheClient.Close()

	db, err := mongoclient.Setup(envPrefixDB)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup postgres database : %s", err))
		exitCode = 1
		return
	}

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	var authClient magistrala.AuthServiceClient
	switch cfg.StandaloneID != "" && cfg.StandaloneToken != "" {
	case true:
		authClient = localusers.NewAuthService(cfg.StandaloneID, cfg.StandaloneToken)
	default:
		authConfig := auth.Config{}
		if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
			logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
			exitCode = 1
			return
		}

		authServiceClient, authHandler, err := auth.Setup(authConfig)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authHandler.Close()
		authClient = authServiceClient
		logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())
	}

	pubSub, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pubSub.Close()
	pubSub = brokerstracing.NewPubSub(httpServerConfig, tracer, pubSub)

	svc, err := newService(ctx, svcName, pubSub, cfg, authClient, tracer, db, cacheClient, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s service: %s", svcName, err))
		exitCode = 1
		return
	}

	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, twapi.MakeHandler(svc, logger, cfg.InstanceID), logger)

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
		logger.Error(fmt.Sprintf("Twins service terminated: %s", err))
	}
}

func newService(ctx context.Context, id string, ps messaging.PubSub, cfg config, users magistrala.AuthServiceClient, tracer trace.Tracer, db *mongo.Database, cacheclient *redis.Client, logger *slog.Logger) (twins.Service, error) {
	twinRepo := twmongodb.NewTwinRepository(db)
	twinRepo = tracing.TwinRepositoryMiddleware(tracer, twinRepo)

	stateRepo := twmongodb.NewStateRepository(db)
	stateRepo = tracing.StateRepositoryMiddleware(tracer, stateRepo)

	idProvider := uuid.New()
	twinCache := events.NewTwinCache(cacheclient)
	twinCache = tracing.TwinCacheMiddleware(tracer, twinCache)

	svc := twins.New(ps, users, twinRepo, twinCache, stateRepo, idProvider, cfg.ChannelID, logger)

	var err error
	svc, err = events.NewEventStoreMiddleware(ctx, svc, cfg.ESURL)
	if err != nil {
		return nil, err
	}

	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	subCfg := messaging.SubscriberConfig{
		ID:      id,
		Topic:   brokers.SubjectAllChannels,
		Handler: handle(ctx, logger, cfg.ChannelID, svc),
	}
	if err = ps.Subscribe(ctx, subCfg); err != nil {
		logger.Error(err.Error())
	}

	return svc, nil
}

func handle(ctx context.Context, logger *slog.Logger, chanID string, svc twins.Service) handlerFunc {
	return func(msg *messaging.Message) error {
		if msg.GetChannel() == chanID {
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
