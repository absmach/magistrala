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
	"github.com/absmach/magistrala/alarms/brokers"
	"github.com/absmach/magistrala/alarms/consumer"
	"github.com/absmach/magistrala/alarms/middleware"
	"github.com/absmach/magistrala/alarms/operations"
	alarmsRepo "github.com/absmach/magistrala/alarms/postgres"
	dpostgres "github.com/absmach/magistrala/domains/postgres"
	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authn/authsvc"
	authsvcAuthz "github.com/absmach/magistrala/pkg/authz/authsvc"
	dconsumer "github.com/absmach/magistrala/pkg/domains/events/consumer"
	domainsAuthz "github.com/absmach/magistrala/pkg/domains/grpcclient"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/messaging"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	rconsumer "github.com/absmach/magistrala/pkg/re/events/consumer"
	"github.com/absmach/magistrala/pkg/server"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	rpostgres "github.com/absmach/magistrala/re/postgres"
	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
)

const (
	svcName          = "alarms"
	envPrefixDB      = "MG_ALARMS_DB_"
	envPrefixHTTP    = "MG_ALARMS_HTTP_"
	envPrefixAuth    = "MG_AUTH_GRPC_"
	defDB            = "alarms"
	defSvcHTTPPort   = "8050"
	envPrefixDomains = "MG_DOMAINS_GRPC_"
	alarmEntity      = "alarm"
)

type config struct {
	LogLevel        string  `env:"MG_ALARMS_LOG_LEVEL"    envDefault:"info"`
	BrokerURL       string  `env:"MG_MESSAGE_BROKER_URL" envDefault:"nats://localhost:4222"`
	InstanceID      string  `env:"MG_ALARMS_INSTANCE_ID"  envDefault:""`
	JaegerURL       url.URL `env:"MG_JAEGER_URL"         envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio      float64 `env:"MG_JAEGER_TRACE_RATIO" envDefault:"1.0"`
	ESURL           string  `env:"MG_ES_URL"             envDefault:"nats://localhost:4222"`
	ESConsumerName  string  `env:"MG_ALARMS_EVENT_CONSUMER" envDefault:"alarms"`
	PermissionsFile string  `env:"MG_PERMISSIONS_FILE"             envDefault:"permission.yaml"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	logger, err := mglog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer mglog.ExitWithError(&exitCode)

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

	migrations, err := alarmsRepo.Migration()
	if err != nil {
		logger.Error(fmt.Sprintf("failed to load migrations: %s", err))
		exitCode = 1
		return
	}

	db, err := postgres.Setup(dbConfig, *migrations)
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
	am := smqauthn.NewAuthNMiddleware(authn)
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

	ddatabase := postgres.NewDatabase(db, dbConfig, tracer)
	drepo := dpostgres.NewRepository(ddatabase)

	if err := dconsumer.DomainsEventsSubscribe(ctx, drepo, nil, cfg.ESURL, cfg.ESConsumerName, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to create domains event store : %s", err))
		exitCode = 1
		return
	}

	rdatabase := postgres.NewDatabase(db, dbConfig, tracer)
	rrepo := rpostgres.NewRepository(rdatabase)

	if err := rconsumer.RulesEventsSubscribe(ctx, rrepo, cfg.ESURL, cfg.ESConsumerName, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to rules events: %s", err))
		exitCode = 1
		return
	}

	idp := uuid.New()

	svc := alarms.NewService(idp, repo)

	permConfig, err := permissions.ParsePermissionsFile(cfg.PermissionsFile)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse permissions file: %s", err))
		exitCode = 1
		return
	}

	alarmOps, _, err := permConfig.GetEntityPermissions(alarmEntity)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get alarm permissions: %s", err))
		exitCode = 1
		return
	}

	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{
			operations.EntityType: alarmOps,
		},
		permissions.EntitiesOperationDetails[permissions.Operation]{
			operations.EntityType: operations.OperationDetails(),
		},
	)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create entity operations: %s", err))
		exitCode = 1
		return
	}

	svc, err = middleware.NewAuthorizationMiddleware(svc, authz, entitiesOps)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create authorization middleware: %s", err))
		exitCode = 1
		return
	}

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
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpAPI.MakeHandler(svc, logger, idp, cfg.InstanceID, am), logger)

	pubSub, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pubSub.Close()
	pubSub = brokerstracing.NewPubSub(httpServerConfig, tracer, pubSub)

	consumer := consumer.NewHandler(svc, logger)

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
