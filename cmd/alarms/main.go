// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/absmach/magistrala/alarms"
	httpAPI "github.com/absmach/magistrala/alarms/api"
	"github.com/absmach/magistrala/alarms/consumer"
	"github.com/absmach/magistrala/alarms/consumer/brokers"
	"github.com/absmach/magistrala/alarms/middleware"
	alarmsRepo "github.com/absmach/magistrala/alarms/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	smqlog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/pkg/authn/authsvc"
	authsvcAuthz "github.com/absmach/supermq/pkg/authz/authsvc"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	"github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	brokerstracing "github.com/absmach/supermq/pkg/messaging/brokers/tracing"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
)

const (
	svcName          = "alarms"
	envPrefixDB      = "MG_ALARMS_DB_"
	envPrefixHTTP    = "MG_ALARMS_HTTP_"
	envPrefixAuth    = "SMQ_AUTH_GRPC_"
	defDB            = "alarms"
	defSvcHTTPPort   = "8050"
	envPrefixDomains = "SMQ_DOMAINS_GRPC_"
)

type config struct {
	LogLevel   string  `env:"MG_ALARMS_LOG_LEVEL"    envDefault:"info"`
	BrokerURL  string  `env:"SMQ_MESSAGE_BROKER_URL" envDefault:"nats://localhost:4222"`
	InstanceID string  `env:"MG_ALARMS_INSTANCE_ID"  envDefault:""`
	JaegerURL  url.URL `env:"SMQ_JAEGER_URL"         envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio float64 `env:"SMQ_JAEGER_TRACE_RATIO" envDefault:"1.0"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	logger, err := smqlog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer smqlog.ExitWithError(&exitCode)

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

	dbConfig := postgres.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
	}

	db, err := postgres.Setup(dbConfig, *alarmsRepo.Migration())
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	repo := alarmsRepo.NewAlarmsRepo(db)

	authConfig := grpcclient.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	authn, authnClient, err := authsvc.NewAuthentication(ctx, authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authnClient.Close()
	logger.Info("AuthN  successfully connected to auth gRPC server " + authnClient.Secure())

	domsGrpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&domsGrpcCfg, env.Options{Prefix: envPrefixDomains}); err != nil {
		logger.Error(fmt.Sprintf("failed to load domains gRPC client configuration : %s", err))
		exitCode = 1
		return
	}

	domAuthz, _, domainsHandler, err := domainsAuthz.NewAuthorization(ctx, domsGrpcCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer domainsHandler.Close()

	authz, authzHandler, err := authsvcAuthz.NewAuthorization(ctx, authConfig, domAuthz)
	if err != nil {
		logger.Error("failed to create authz " + err.Error())
		exitCode = 1
		return
	}
	defer authzHandler.Close()

	logger.Info("AuthZ successfully connected to auth gRPC server " + authzHandler.Secure())

	idp := uuid.New()

	svc := alarms.NewService(idp, repo)

	svc = middleware.NewAuthorizationMiddleware(svc, authz)
	svc = middleware.NewLoggingMiddleware(logger, svc)
	counter, latency := prometheus.MakeMetrics("alarms", "api")
	svc = middleware.NewMetricsMiddleware(counter, latency, svc)
	svc = middleware.NewTracingMiddleware(tracer, svc)

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpAPI.MakeHandler(svc, logger, idp, cfg.InstanceID, authn), logger)

	pubSub, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pubSub.Close()
	pubSub = brokerstracing.NewPubSub(httpServerConfig, tracer, pubSub)

	consumer := consumer.Newhandler(svc, logger)

	subCfg := messaging.SubscriberConfig{
		ID:             svcName,
		Topic:          brokers.AllTopic,
		DeliveryPolicy: messaging.DeliverAllPolicy,
		Handler:        consumer,
	}
	if err := pubSub.Subscribe(ctx, subCfg); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to message broker: %s", err))
		exitCode = 1

		return
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("billing service terminated: %s", err))
	}
}
