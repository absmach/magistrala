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
	"github.com/absmach/magistrala"
	abrokers "github.com/absmach/magistrala/alarms/brokers"
	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/consumers/writers/brokers"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/internal/email"
	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	atomauthn "github.com/absmach/magistrala/pkg/authn/atom"
	"github.com/absmach/magistrala/pkg/callout"
	"github.com/absmach/magistrala/pkg/emailer"
	"github.com/absmach/magistrala/pkg/grpcclient"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/magistrala/pkg/messaging"
	smqbrokers "github.com/absmach/magistrala/pkg/messaging/brokers"
	brokerstracing "github.com/absmach/magistrala/pkg/messaging/brokers/tracing"
	"github.com/absmach/magistrala/pkg/permissions"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/ticker"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/re"
	httpapi "github.com/absmach/magistrala/re/api"
	"github.com/absmach/magistrala/re/events"
	"github.com/absmach/magistrala/re/middleware"
	"github.com/absmach/magistrala/re/operations"
	repg "github.com/absmach/magistrala/re/postgres"
	grpcClient "github.com/absmach/magistrala/readers/api/grpc"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName          = "rules_engine"
	envPrefixDB      = "MG_RE_DB_"
	envPrefixHTTP    = "MG_RE_HTTP_"
	envPrefixCallout = "MG_RE_CALLOUT_"
	defDB            = "r"
	defSvcHTTPPort   = "9008"
	envPrefixGrpc    = "MG_TIMESCALE_READER_GRPC_"
)

// We use a buffered channel to prevent blocking, as logging is an expensive operation.
// A larger buffer size would also work, but we’d likely need another instance of RE in that case.
// A smaller size would probably work too, but there's no need to be that frugal with resources.
const channBuffer = 256

type config struct {
	LogLevel         string        `env:"MG_RE_LOG_LEVEL"             envDefault:"info"`
	InstanceID       string        `env:"MG_RE_INSTANCE_ID"           envDefault:""`
	JaegerURL        url.URL       `env:"MG_JAEGER_URL"              envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry    bool          `env:"MG_SEND_TELEMETRY"          envDefault:"true"`
	ESURL            string        `env:"MG_ES_URL"                  envDefault:"nats://localhost:4222"`
	ESConsumerName   string        `env:"MG_RE_EVENT_CONSUMER"        envDefault:"rules_engine"`
	CacheURL         string        `env:"MG_RE_CACHE_URL"             envDefault:"redis://localhost:6379/0"`
	CacheKeyDuration time.Duration `env:"MG_RE_CACHE_KEY_DURATION"    envDefault:"10m"`
	TraceRatio       float64       `env:"MG_JAEGER_TRACE_RATIO"      envDefault:"1.0"`
	BrokerURL        string        `env:"MG_MESSAGE_BROKER_URL"      envDefault:"nats://localhost:4222"`
	PermissionsFile  string        `env:"MG_PERMISSIONS_FILE"        envDefault:"permission.yaml"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

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

	ec := email.Config{}
	if err := env.Parse(&ec); err != nil {
		logger.Error(fmt.Sprintf("failed to load email configuration : %s", err))
		exitCode = 1

		return
	}

	callCfg := callout.Config{}
	if err := env.ParseWithOptions(&callCfg, env.Options{Prefix: envPrefixCallout}); err != nil {
		logger.Error(fmt.Sprintf("failed to parse callout config : %s", err))
		exitCode = 1
		return
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1

		return
	}
	migration, err := repg.Migration()
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1

		return
	}
	db, err := pgclient.Setup(dbConfig, *migration)
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

	callout, err := callout.New(callCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create new callout: %s", err))
		exitCode = 1
		return
	}

	msgSub, err := smqbrokers.NewPubSub(ctx, cfg.BrokerURL, logger, smqbrokers.ConnectionName("re-msg-pubsub"))
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

	atomCfg := atom.LoadConfig()
	if atomCfg.URL == "" {
		logger.Error("ATOM_URL is required")
		exitCode = 1
		return
	}
	authnSvc := atomauthn.NewAuthentication()
	logger.Info("AuthN configured to use Atom bearer tokens")
	am := smqauthn.NewAuthNMiddleware(authnSvc)
	runInfo := make(chan pkglog.RunInfo, channBuffer)

	logger.Info("AuthZ configured to use Atom PDP")

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

	svc, err := newService(ctx, cfg, database, runInfo, msgSub, writersPub, alarmsPub, ec, logger, readersClient, callout, tracer)
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

	httpSvc := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, am, mux, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
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

func newService(ctx context.Context, cfg config, db pgclient.Database, runInfo chan pkglog.RunInfo, rePubSub messaging.PubSub, writersPub, alarmsPub messaging.Publisher, ec email.Config, logger *slog.Logger, readersClient grpcReadersV1.ReadersServiceClient, callout callout.Callout, tracer trace.Tracer) (re.Service, error) {
	repo := repg.NewRepository(db)
	idp := uuid.New()

	emailerClient, err := emailer.New(&ec)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
	}

	atomCfg := atom.LoadConfig()

	var csvc re.Service
	csvc, err = re.NewService(repo, runInfo, idp, rePubSub, writersPub, alarmsPub, ticker.NewTicker(time.Second*30), emailerClient, readersClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create RE service: %w", err)
	}
	csvc = re.WithAtom(csvc, atom.NewClient(atomCfg))

	csvc, err = events.NewEventStoreMiddleware(ctx, csvc, cfg.ESURL)
	if err != nil {
		return nil, fmt.Errorf("failed to init re event store middleware: %w", err)
	}

	permConfig, err := permissions.ParsePermissionsFile(cfg.PermissionsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse permissions file: %w", err)
	}

	ruleOps, _, err := permConfig.GetEntityPermissions(operations.EntityType)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule permissions: %w", err)
	}

	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{
			operations.EntityType: ruleOps,
		},
		permissions.EntitiesOperationDetails[permissions.Operation]{
			operations.EntityType: operations.OperationDetails(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create entities operations: %w", err)
	}

	csvc, err = middleware.AtomAuthorizationMiddleware(csvc, atom.NewClient(atomCfg), entitiesOps)
	if err != nil {
		return nil, err
	}
	csvc, err = middleware.NewCallout(csvc, callout, entitiesOps)
	if err != nil {
		return nil, err
	}
	csvc = middleware.LoggingMiddleware(csvc, logger)
	counter, latency := prometheus.MakeMetrics("re", "api")
	csvc = middleware.NewMetricsMiddleware(counter, latency, csvc)
	csvc = middleware.NewTracingMiddleware(tracer, csvc)

	return csvc, nil
}
