package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	api "github.com/mainflux/mainflux/auth/api"
	grpcapi "github.com/mainflux/mainflux/auth/api/grpc"
	httpapi "github.com/mainflux/mainflux/auth/api/http"
	"github.com/mainflux/mainflux/auth/jwt"
	apostgres "github.com/mainflux/mainflux/auth/postgres"
	"github.com/mainflux/mainflux/auth/spicedb"
	"github.com/mainflux/mainflux/auth/tracing"
	"github.com/mainflux/mainflux/internal"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgclient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/internal/postgres"

	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	grpcserver "github.com/mainflux/mainflux/internal/server/grpc"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	svcName           = "auth"
	envPrefixHTTP     = "MF_AUTH_HTTP_"
	envPrefixGrpc     = "MF_AUTH_GRPC_"
	envPrefixDB       = "MF_AUTH_DB_"
	defDB             = "auth"
	defSvcHTTPPort    = "8180"
	defSvcGRPCPort    = "8181"
	SpicePreSharedKey = "12345678"
)

type config struct {
	LogLevel          string `env:"MF_AUTH_LOG_LEVEL"               envDefault:"info"`
	SecretKey         string `env:"MF_AUTH_SECRET_KEY"              envDefault:"secret"`
	JaegerURL         string `env:"MF_JAEGER_URL"                   envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry     bool   `env:"MF_SEND_TELEMETRY"               envDefault:"true"`
	InstanceID        string `env:"MF_AUTH_ADAPTER_INSTANCE_ID"     envDefault:""`
	AccessDuration    string `env:"MF_AUTH_ACCESS_TOKEN_DURATION"  envDefault:"30m"`
	RefreshDuration   string `env:"MF_AUTH_REFRESH_TOKEN_DURATION" envDefault:"24h"`
	SpicedbHost       string `env:"MF_SPICEDB_HOST"                 envDefault:"localhost"`
	SpicedbPort       string `env:"MF_SPICEDB_PORT"                 envDefault:"50051"`
	SpicedbSchemaFile string `env:"MF_SPICEDB_SCHEMA_FILE"          envDefault:"./docker/spicedb/schema.zed"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to init logger: %s", err.Error()))
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

	dbConfig := pgclient.Config{Name: defDB}
	if err := dbConfig.LoadEnv(envPrefixDB); err != nil {
		logger.Fatal(err.Error())
	}
	db, err := pgclient.SetupWithConfig(envPrefixDB, *apostgres.Migration(), dbConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaegerclient.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID)
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

	spicedbclient, err := initSpiceDB(cfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to init spicedb grpc client : %s\n", err.Error()))
		exitCode = 1
		return
	}

	svc := newService(db, tracer, cfg, dbConfig, logger, spicedbclient)

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, logger, cfg.InstanceID), logger)

	grpcServerConfig := server.Config{Port: defSvcGRPCPort}
	if err := env.Parse(&grpcServerConfig, env.Options{Prefix: envPrefixGrpc}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	registerAuthServiceServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		mainflux.RegisterAuthServiceServer(srv, grpcapi.NewServer(svc))
	}

	gs := grpcserver.New(ctx, cancel, svcName, grpcServerConfig, registerAuthServiceServer, logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})
	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("users service terminated: %s", err))
	}
}

func initSpiceDB(cfg config) (*authzed.Client, error) {
	client, err := authzed.NewClient(
		fmt.Sprintf("%s:%s", cfg.SpicedbHost, cfg.SpicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(SpicePreSharedKey),
	)
	if err != nil {
		return client, err
	}

	if err := initSchema(client, cfg.SpicedbSchemaFile); err != nil {
		return client, err
	}

	return client, nil
}

func initSchema(client *authzed.Client, schemaFilePath string) error {
	schemaContent, err := os.ReadFile(schemaFilePath)
	if err != nil {
		return fmt.Errorf("failed to read spice db schema file : %w", err)
	}

	if _, err = client.SchemaServiceClient.WriteSchema(context.Background(), &v1.WriteSchemaRequest{Schema: string(schemaContent)}); err != nil {
		return fmt.Errorf("failed to create schema in spicedb : %w", err)
	}

	return nil
}

func newService(db *sqlx.DB, tracer trace.Tracer, cfg config, dbConfig pgclient.Config, logger mflog.Logger, spicedbClient *authzed.Client) auth.Service {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	keysRepo := apostgres.New(database)

	pa := spicedb.NewPolicyAgent(spicedbClient, logger)
	idProvider := uuid.New()
	t := jwt.New([]byte(cfg.SecretKey))

	aDuration, err := time.ParseDuration(cfg.AccessDuration)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse access token duration: %s", err.Error()))
	}
	rDuration, err := time.ParseDuration(cfg.RefreshDuration)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse refresh token duration: %s", err.Error()))
	}

	svc := auth.New(keysRepo, idProvider, t, pa, aDuration, rDuration)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("groups", "api")
	svc = api.MetricsMiddleware(svc, counter, latency)
	svc = tracing.New(svc, tracer)

	return svc
}
