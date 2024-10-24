// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/absmach/magistrala/auth"
	api "github.com/absmach/magistrala/auth/api"
	authgrpcapi "github.com/absmach/magistrala/auth/api/grpc/auth"
	tokengrpcapi "github.com/absmach/magistrala/auth/api/grpc/token"
	httpapi "github.com/absmach/magistrala/auth/api/http"
	"github.com/absmach/magistrala/auth/jwt"
	apostgres "github.com/absmach/magistrala/auth/postgres"
	"github.com/absmach/magistrala/auth/tracing"
	grpcAuthV1 "github.com/absmach/magistrala/internal/grpc/auth/v1"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/policies/spicedb"
	"github.com/absmach/magistrala/pkg/postgres"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	grpcserver "github.com/absmach/magistrala/pkg/server/grpc"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	svcName        = "auth"
	envPrefixHTTP  = "MG_AUTH_HTTP_"
	envPrefixGrpc  = "MG_AUTH_GRPC_"
	envPrefixDB    = "MG_AUTH_DB_"
	defDB          = "auth"
	defSvcHTTPPort = "8189"
	defSvcGRPCPort = "8181"
)

type config struct {
	LogLevel            string        `env:"MG_AUTH_LOG_LEVEL"               envDefault:"info"`
	SecretKey           string        `env:"MG_AUTH_SECRET_KEY"              envDefault:"secret"`
	JaegerURL           url.URL       `env:"MG_JAEGER_URL"                   envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool          `env:"MG_SEND_TELEMETRY"               envDefault:"true"`
	InstanceID          string        `env:"MG_AUTH_ADAPTER_INSTANCE_ID"     envDefault:""`
	AccessDuration      time.Duration `env:"MG_AUTH_ACCESS_TOKEN_DURATION"   envDefault:"1h"`
	RefreshDuration     time.Duration `env:"MG_AUTH_REFRESH_TOKEN_DURATION"  envDefault:"24h"`
	InvitationDuration  time.Duration `env:"MG_AUTH_INVITATION_DURATION"     envDefault:"168h"`
	SpicedbHost         string        `env:"MG_SPICEDB_HOST"                 envDefault:"localhost"`
	SpicedbPort         string        `env:"MG_SPICEDB_PORT"                 envDefault:"50051"`
	SpicedbSchemaFile   string        `env:"MG_SPICEDB_SCHEMA_FILE"          envDefault:"./docker/spicedb/schema.zed"`
	SpicedbPreSharedKey string        `env:"MG_SPICEDB_PRE_SHARED_KEY"       envDefault:"12345678"`
	TraceRatio          float64       `env:"MG_JAEGER_TRACE_RATIO"           envDefault:"1.0"`
	ESURL               string        `env:"MG_ES_URL"                       envDefault:"nats://localhost:4222"`
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

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
	}

	am := apostgres.Migration()
	db, err := pgclient.Setup(dbConfig, *am)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

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

	spicedbclient, err := initSpiceDB(ctx, cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init spicedb grpc client : %s\n", err.Error()))
		exitCode = 1
		return
	}
	svc := newService(ctx, db, tracer, cfg, dbConfig, logger, spicedbclient)

	grpcServerConfig := server.Config{Port: defSvcGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGrpc}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	registerAuthServiceServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcTokenV1.RegisterTokenServiceServer(srv, tokengrpcapi.NewTokenServer(svc))
		grpcAuthV1.RegisterAuthServiceServer(srv, authgrpcapi.NewAuthServer(svc))
	}

	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerAuthServiceServer, logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}
	g.Go(func() error {
		return gs.Start()
	})

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, logger, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("users service terminated: %s", err))
	}
}

func initSpiceDB(ctx context.Context, cfg config) (*authzed.ClientWithExperimental, error) {
	client, err := authzed.NewClientWithExperimentalAPIs(
		fmt.Sprintf("%s:%s", cfg.SpicedbHost, cfg.SpicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(cfg.SpicedbPreSharedKey),
	)
	if err != nil {
		return client, err
	}

	if err := initSchema(ctx, client, cfg.SpicedbSchemaFile); err != nil {
		return client, err
	}

	return client, nil
}

func initSchema(ctx context.Context, client *authzed.ClientWithExperimental, schemaFilePath string) error {
	schemaContent, err := os.ReadFile(schemaFilePath)
	if err != nil {
		return fmt.Errorf("failed to read spice db schema file : %w", err)
	}

	if _, err = client.SchemaServiceClient.WriteSchema(ctx, &v1.WriteSchemaRequest{Schema: string(schemaContent)}); err != nil {
		return fmt.Errorf("failed to create schema in spicedb : %w", err)
	}

	return nil
}

func newService(_ context.Context, db *sqlx.DB, tracer trace.Tracer, cfg config, dbConfig pgclient.Config, logger *slog.Logger, spicedbClient *authzed.ClientWithExperimental) auth.Service {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	keysRepo := apostgres.New(database)
	idProvider := uuid.New()

	pEvaluator := spicedb.NewPolicyEvaluator(spicedbClient, logger)
	pService := spicedb.NewPolicyService(spicedbClient, logger)

	t := jwt.New([]byte(cfg.SecretKey))

	svc := auth.New(keysRepo, idProvider, t, pEvaluator, pService, cfg.AccessDuration, cfg.RefreshDuration, cfg.InvitationDuration)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics("auth", "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc
}
