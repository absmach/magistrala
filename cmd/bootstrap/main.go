// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains bootstrap main function to start the bootstrap service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/bootstrap"
	httpapi "github.com/absmach/supermq/bootstrap/api"
	"github.com/absmach/supermq/bootstrap/events/consumer"
	"github.com/absmach/supermq/bootstrap/events/producer"
	"github.com/absmach/supermq/bootstrap/middleware"
	bootstrappg "github.com/absmach/supermq/bootstrap/postgres"
	"github.com/absmach/supermq/bootstrap/tracing"
	smqlog "github.com/absmach/supermq/logger"
	authsvcAuthn "github.com/absmach/supermq/pkg/authn/authsvc"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	authsvcAuthz "github.com/absmach/supermq/pkg/authz/authsvc"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/grpcclient"
	"github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/policies/spicedb"
	pgclient "github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/prometheus"
	mgsdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	svcName          = "bootstrap"
	envPrefixDB      = "SMQ_BOOTSTRAP_DB_"
	envPrefixHTTP    = "SMQ_BOOTSTRAP_HTTP_"
	envPrefixAuth    = "SMQ_AUTH_GRPC_"
	envPrefixDomains = "SMQ_DOMAINS_GRPC_"
	defDB            = "bootstrap"
	defSvcHTTPPort   = "9013"

	stream   = "events.supermq.clients"
	streamID = "supermq.bootstrap"
)

type config struct {
	LogLevel            string  `env:"SMQ_BOOTSTRAP_LOG_LEVEL"        envDefault:"info"`
	EncKey              string  `env:"SMQ_BOOTSTRAP_ENCRYPT_KEY"      envDefault:"12345678910111213141516171819202"`
	ESConsumerName      string  `env:"SMQ_BOOTSTRAP_EVENT_CONSUMER"   envDefault:"bootstrap"`
	ClientsURL          string  `env:"SMQ_CLIENTS_URL"                envDefault:"http://localhost:9000"`
	JaegerURL           url.URL `env:"SMQ_JAEGER_URL"                 envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool    `env:"SMQ_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID          string  `env:"SMQ_BOOTSTRAP_INSTANCE_ID"      envDefault:""`
	ESURL               string  `env:"SMQ_ES_URL"                     envDefault:"nats://localhost:4222"`
	TraceRatio          float64 `env:"SMQ_JAEGER_TRACE_RATIO"         envDefault:"1.0"`
	SpicedbHost         string  `env:"SMQ_SPICEDB_HOST"               envDefault:"localhost"`
	SpicedbPort         string  `env:"SMQ_SPICEDB_PORT"               envDefault:"50051"`
	SpicedbPreSharedKey string  `env:"SMQ_SPICEDB_PRE_SHARED_KEY"     envDefault:"12345678"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

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

	// Create new postgres client
	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
	}
	db, err := pgclient.Setup(dbConfig, *bootstrappg.Migration())
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	policySvc, err := newPolicyService(cfg, logger)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy client successfully connected to spicedb gRPC server")

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
	logger.Info("AuthN successfully connected to auth gRPC server " + authnClient.Secure())
	defer authnClient.Close()

	domsGrpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&domsGrpcCfg, env.Options{Prefix: envPrefixDomains}); err != nil {
		logger.Error(fmt.Sprintf("failed to load domains gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	domainsAuthz, _, domainsHandler, err := domainsAuthz.NewAuthorization(ctx, domsGrpcCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer domainsHandler.Close()

	authz, authzClient, err := authsvcAuthz.NewAuthorization(ctx, grpcCfg, domainsAuthz)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authzClient.Close()
	logger.Info("AuthZ successfully connected to auth gRPC server " + authzClient.Secure())

	// Create new service
	svc, err := newService(ctx, authz, policySvc, db, tracer, logger, cfg, dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s service: %s", svcName, err))
		exitCode = 1
		return
	}

	if err = subscribeToClientsES(ctx, svc, cfg, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to clients event store: %s", err))
		exitCode = 1
		return
	}

	logger.Info("Subscribed to Event Store")

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, authn, bootstrap.NewConfigReader([]byte(cfg.EncKey)), logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	// Start servers
	g.Go(func() error {
		return hs.Start()
	})
	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Bootstrap service terminated: %s", err))
	}
}

func newService(ctx context.Context, authz smqauthz.Authorization, policySvc policies.Service, db *sqlx.DB, tracer trace.Tracer, logger *slog.Logger, cfg config, dbConfig pgclient.Config) (bootstrap.Service, error) {
	database := pgclient.NewDatabase(db, dbConfig, tracer)

	repoConfig := bootstrappg.NewConfigRepository(database, logger)

	config := mgsdk.Config{
		ClientsURL: cfg.ClientsURL,
	}

	sdk := mgsdk.NewSDK(config)
	idp := uuid.New()

	svc := bootstrap.New(policySvc, repoConfig, sdk, []byte(cfg.EncKey), idp)

	publisher, err := store.NewPublisher(ctx, cfg.ESURL, streamID)
	if err != nil {
		return nil, err
	}

	svc = middleware.AuthorizationMiddleware(svc, authz)
	svc = producer.NewEventStoreMiddleware(svc, publisher)
	svc = middleware.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc, nil
}

func subscribeToClientsES(ctx context.Context, svc bootstrap.Service, cfg config, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, cfg.ESURL, logger)
	if err != nil {
		return err
	}

	subConfig := events.SubscriberConfig{
		Stream:   stream,
		Consumer: cfg.ESConsumerName,
		Handler:  consumer.NewEventHandler(svc),
	}
	return subscriber.Subscribe(ctx, subConfig)
}

func newPolicyService(cfg config, logger *slog.Logger) (policies.Service, error) {
	client, err := authzed.NewClientWithExperimentalAPIs(
		fmt.Sprintf("%s:%s", cfg.SpicedbHost, cfg.SpicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(cfg.SpicedbPreSharedKey),
	)
	if err != nil {
		return nil, err
	}
	policySvc := spicedb.NewPolicyService(client, logger)

	return policySvc, nil
}
