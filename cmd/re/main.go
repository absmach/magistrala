// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains rule engine main function to start the service.
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
	abrokers "github.com/absmach/magistrala/alarms/brokers"
	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/consumers/writers/brokers"
	"github.com/absmach/magistrala/internal/email"
	"github.com/absmach/magistrala/pkg/emailer"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/magistrala/pkg/ticker"
	"github.com/absmach/magistrala/re"
	httpapi "github.com/absmach/magistrala/re/api"
	"github.com/absmach/magistrala/re/middleware"
	repg "github.com/absmach/magistrala/re/postgres"
	grpcClient "github.com/absmach/magistrala/readers/api/grpc"
	"github.com/absmach/supermq"
	smqlog "github.com/absmach/supermq/logger"
	authnsvc "github.com/absmach/supermq/pkg/authn/authsvc"
	mgauthz "github.com/absmach/supermq/pkg/authz"
	authzsvc "github.com/absmach/supermq/pkg/authz/authsvc"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	smqbrokers "github.com/absmach/supermq/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/supermq/pkg/messaging/brokers/tracing"
	pgclient "github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"
)

const (
	svcName          = "rules_engine"
	envPrefixDB      = "MG_RE_DB_"
	envPrefixHTTP    = "MG_RE_HTTP_"
	envPrefixAuth    = "SMQ_AUTH_GRPC_"
	defDB            = "r"
	defSvcHTTPPort   = "9008"
	envPrefixGrpc    = "MG_TIMESCALE_READER_GRPC_"
	envPrefixDomains = "SMQ_DOMAINS_GRPC_"
)

// We use a buffered channel to prevent blocking, as logging is an expensive operation.
// A larger buffer size would also work, but we’d likely need another instance of RE in that case.
// A smaller size would probably work too, but there's no need to be that frugal with resources.
const channBuffer = 256

type config struct {
	LogLevel         string        `env:"MG_RE_LOG_LEVEL"           envDefault:"info"`
	InstanceID       string        `env:"MG_RE_INSTANCE_ID"         envDefault:""`
	JaegerURL        url.URL       `env:"SMQ_JAEGER_URL"             envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry    bool          `env:"SMQ_SEND_TELEMETRY"         envDefault:"true"`
	ESURL            string        `env:"SMQ_ES_URL"                 envDefault:"nats://localhost:4222"`
	CacheURL         string        `env:"MG_RE_CACHE_URL"           envDefault:"redis://localhost:6379/0"`
	CacheKeyDuration time.Duration `env:"MG_RE_CACHE_KEY_DURATION"  envDefault:"10m"`
	TraceRatio       float64       `env:"SMQ_JAEGER_TRACE_RATIO"     envDefault:"1.0"`
	BrokerURL        string        `env:"SMQ_MESSAGE_BROKER_URL"     envDefault:"nats://localhost:4222"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	var logger *slog.Logger
	logger, err := smqlog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err.Error())
	}

	var exitCode int
	defer smqlog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	ec := email.Config{}
	if err := env.Parse(&ec); err != nil {
		logger.Error(fmt.Sprintf("failed to load email configuration : %s", err))
		exitCode = 1

		return
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1

		return
	}
	db, err := pgclient.Setup(dbConfig, *repg.Migration())
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

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1

		return
	}

	msgSub, err := smqbrokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker for mg pubSub: %s", err))
		exitCode = 1

		return
	}
	defer msgSub.Close()
	msgSub = brokerstracing.NewPubSub(httpServerConfig, tracer, msgSub)

	writersPub, err := brokers.NewPublisher(ctx, cfg.BrokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker for writers publisher: %s", err))
		exitCode = 1

		return
	}
	defer writersPub.Close()
	writersPub = brokerstracing.NewPublisher(httpServerConfig, tracer, writersPub)

	alarmsPub, err := abrokers.NewPublisher(ctx, cfg.BrokerURL)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker for alarms publisher: %s", err))
		exitCode = 1

		return
	}
	defer alarmsPub.Close()
	alarmsPub = brokerstracing.NewPublisher(httpServerConfig, tracer, alarmsPub)

	grpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&grpcCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC client configuration : %s", err))
		exitCode = 1

		return
	}
	authn, authnClient, err := authnsvc.NewAuthentication(ctx, grpcCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1

		return
	}
	defer authnClient.Close()
	logger.Info("AuthN  successfully connected to auth gRPC server " + authnClient.Secure())
	runInfo := make(chan pkglog.RunInfo, channBuffer)

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

	authz, authzClient, err := authzsvc.NewAuthorization(ctx, grpcCfg, domAuthz)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authzClient.Close()
	logger.Info("AuthZ  successfully connected to auth gRPC server " + authnClient.Secure())

	database := pgclient.NewDatabase(db, dbConfig, tracer)
	regrpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&regrpcCfg, env.Options{Prefix: envPrefixGrpc}); err != nil {
		logger.Error(fmt.Sprintf("failed to load clients gRPC client configuration : %s", err))
		exitCode = 1
		return
	}

	client, err := grpcclient.NewHandler(regrpcCfg)
	if err != nil {
		exitCode = 1
		return
	}
	defer client.Close()

	readersClient := grpcClient.NewReadersClient(client.Connection(), regrpcCfg.Timeout)
	logger.Info("Readers gRPC client successfully connected to readers gRPC server " + client.Secure())

	svc, err := newService(database, runInfo, msgSub, writersPub, alarmsPub, authz, ec, logger, readersClient)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create services: %s", err))
		exitCode = 1

		return
	}
	subCfg := messaging.SubscriberConfig{
		ID:             svcName,
		Topic:          smqbrokers.SubjectAllMessages,
		DeliveryPolicy: messaging.DeliverAllPolicy,
		Handler:        svc,
	}
	if err := msgSub.Subscribe(ctx, subCfg); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to internal message broker: %s", err))
		exitCode = 1

		return
	}

	go func() {
		for info := range runInfo {
			logger.LogAttrs(context.Background(), info.Level, info.Message, info.Details...)
		}
	}()

	mux := chi.NewRouter()

	httpSvc := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, authn, mux, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return svc.StartScheduler(ctx)
	})

	g.Go(func() error {
		return httpSvc.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSvc)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}

func newService(db pgclient.Database, runInfo chan pkglog.RunInfo, rePubSub messaging.PubSub, writersPub, alarmsPub messaging.Publisher, authz mgauthz.Authorization, ec email.Config, logger *slog.Logger, readersClient grpcReadersV1.ReadersServiceClient) (re.Service, error) {
	repo := repg.NewRepository(db)
	idp := uuid.New()

	emailerClient, err := emailer.New(&ec)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
	}

	csvc := re.NewService(repo, runInfo, idp, rePubSub, writersPub, alarmsPub, ticker.NewTicker(time.Second*30), emailerClient, readersClient)
	csvc, err = middleware.AuthorizationMiddleware(csvc, authz)
	if err != nil {
		return nil, err
	}
	csvc = middleware.LoggingMiddleware(csvc, logger)

	return csvc, nil
}
