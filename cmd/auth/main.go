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
	"github.com/absmach/supermq"
	grpcAuthV1 "github.com/absmach/supermq/api/grpc/auth/v1"
	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/auth"
	authgrpcapi "github.com/absmach/supermq/auth/api/grpc/auth"
	tokengrpcapi "github.com/absmach/supermq/auth/api/grpc/token"
	httpapi "github.com/absmach/supermq/auth/api/http"
	"github.com/absmach/supermq/auth/cache"
	"github.com/absmach/supermq/auth/hasher"
	"github.com/absmach/supermq/auth/middleware"
	apostgres "github.com/absmach/supermq/auth/postgres"
	"github.com/absmach/supermq/auth/tokenizer/asymmetric"
	"github.com/absmach/supermq/auth/tokenizer/symmetric"
	redisclient "github.com/absmach/supermq/internal/clients/redis"
	smqlog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/policies/spicedb"
	pgclient "github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	grpcserver "github.com/absmach/supermq/pkg/server/grpc"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	svcName        = "auth"
	envPrefixHTTP  = "SMQ_AUTH_HTTP_"
	envPrefixGrpc  = "SMQ_AUTH_GRPC_"
	envPrefixDB    = "SMQ_AUTH_DB_"
	defDB          = "auth"
	defSvcHTTPPort = "8189"
	defSvcGRPCPort = "8181"
)

type config struct {
	LogLevel                      string        `env:"SMQ_AUTH_LOG_LEVEL"                         envDefault:"info"`
	JaegerURL                     url.URL       `env:"SMQ_JAEGER_URL"                             envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry                 bool          `env:"SMQ_SEND_TELEMETRY"                         envDefault:"true"`
	InstanceID                    string        `env:"SMQ_AUTH_ADAPTER_INSTANCE_ID"               envDefault:""`
	AccessDuration                time.Duration `env:"SMQ_AUTH_ACCESS_TOKEN_DURATION"             envDefault:"1h"`
	RefreshDuration               time.Duration `env:"SMQ_AUTH_REFRESH_TOKEN_DURATION"            envDefault:"24h"`
	KeyAlgorithm                  string        `env:"SMQ_AUTH_KEYS_ALGORITHM"                    envDefault:"EdDSA"`
	ActiveKeyPath                 string        `env:"SMQ_AUTH_KEYS_ACTIVE_KEY_PATH"              envDefault:"./keys/active.key"`
	RetiringKeyPath               string        `env:"SMQ_AUTH_KEYS_RETIRING_KEY_PATH"            envDefault:""`
	ActiveSecretKey               string        `env:"SMQ_AUTH_ACTIVE_SECRET_KEY"                 envDefault:"secret"`
	RetiringSecretKey             string        `env:"SMQ_AUTH_RETIRING_SECRET_KEY"               envDefault:"secret"`
	InvitationDuration            time.Duration `env:"SMQ_AUTH_INVITATION_DURATION"               envDefault:"168h"`
	SpicedbHost                   string        `env:"SMQ_SPICEDB_HOST"                           envDefault:"localhost"`
	SpicedbPort                   string        `env:"SMQ_SPICEDB_PORT"                           envDefault:"50051"`
	SpicedbSchemaFile             string        `env:"SMQ_SPICEDB_SCHEMA_FILE"                    envDefault:"./docker/spicedb/schema.zed"`
	SpicedbPreSharedKey           string        `env:"SMQ_SPICEDB_PRE_SHARED_KEY"                 envDefault:"12345678"`
	TraceRatio                    float64       `env:"SMQ_JAEGER_TRACE_RATIO"                     envDefault:"1.0"`
	ESURL                         string        `env:"SMQ_ES_URL"                                 envDefault:"nats://localhost:4222"`
	CacheURL                      string        `env:"SMQ_AUTH_CACHE_URL"                         envDefault:"redis://localhost:6379/0"`
	CacheKeyDuration              time.Duration `env:"SMQ_AUTH_CACHE_KEY_DURATION"                envDefault:"10m"`
	JWKSCacheMaxAge               int           `env:"SMQ_AUTH_JWKS_CACHE_MAX_AGE"                envDefault:"900"`
	JWKSCacheStaleWhileRevalidate int           `env:"SMQ_AUTH_JWKS_CACHE_STALE_WHILE_REVALIDATE" envDefault:"60"`
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

	cacheclient, err := redisclient.Connect(cfg.CacheURL)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer cacheclient.Close()

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

	isSymmetric, err := auth.IsSymmetricAlgorithm(cfg.KeyAlgorithm)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to determine key algorithm type: %s", err.Error()))
		exitCode = 1
		return
	}

	idProvider := uuid.New()

	if err := validateKeyConfig(isSymmetric, cfg, logger); err != nil {
		logger.Error(fmt.Sprintf("invalid key configuration: %s", err.Error()))
		exitCode = 1
		return
	}

	var tokenizer auth.Tokenizer
	switch {
	case isSymmetric:
		tokenizer, err = symmetric.NewTokenizer(cfg.KeyAlgorithm, []byte(cfg.ActiveSecretKey), []byte(cfg.RetiringSecretKey))
		if err != nil {
			logger.Error(fmt.Sprintf("failed to create symmetric key manager: %s", err.Error()))
			exitCode = 1
			return
		}
	default:
		tokenizer, err = asymmetric.NewTokenizer(cfg.ActiveKeyPath, cfg.RetiringKeyPath, idProvider, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("failed to create asymmetric key manager: %s", err.Error()))
			exitCode = 1
			return
		}
	}

	svc, err := newService(db, tracer, cfg, dbConfig, logger, spicedbclient, cacheclient, cfg.CacheKeyDuration, tokenizer, idProvider)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create service : %s\n", err.Error()))
		exitCode = 1
		return
	}

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
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
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
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, logger, cfg.InstanceID, cfg.JWKSCacheMaxAge, cfg.JWKSCacheStaleWhileRevalidate), logger)

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

func validateKeyConfig(isSymmetric bool, cfg config, l *slog.Logger) error {
	if isSymmetric {
		if cfg.ActiveSecretKey == "secret" {
			return fmt.Errorf("default secret key is insecure - please set SMQ_AUTH_ACTIVE_SECRET_KEY environment variable")
		}
		return nil
	}

	// Validate active key path
	_, err := os.Stat(cfg.ActiveKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("active key file not found: %s - please set SMQ_AUTH_KEYS_ACTIVE_KEY_PATH", cfg.ActiveKeyPath)
		}
		return fmt.Errorf("failed to access active key file: %w", err)
	}

	// Retiring key is optional - only validate if path is provided
	if cfg.RetiringKeyPath != "" {
		if _, err := os.Stat(cfg.RetiringKeyPath); err != nil {
			l.Warn("retiring key path provided but file not accessible", slog.Any("error", err))
		}
	}

	return nil
}

func newService(db *sqlx.DB, tracer trace.Tracer, cfg config, dbConfig pgclient.Config, logger *slog.Logger, spicedbClient *authzed.ClientWithExperimental, cacheClient *redis.Client, keyDuration time.Duration, tokenizer auth.Tokenizer, idProvider supermq.IDProvider) (auth.Service, error) {
	cache := cache.NewPatsCache(cacheClient, keyDuration)

	database := pgclient.NewDatabase(db, dbConfig, tracer)
	keysRepo := apostgres.New(database)
	patsRepo := apostgres.NewPatRepo(database, cache)
	hasher := hasher.New()

	pEvaluator := spicedb.NewPolicyEvaluator(spicedbClient, logger)
	pService := spicedb.NewPolicyService(spicedbClient, logger)

	svc := auth.New(keysRepo, patsRepo, nil, hasher, idProvider, tokenizer, pEvaluator, pService, cfg.AccessDuration, cfg.RefreshDuration, cfg.InvitationDuration)
	svc = middleware.NewLogging(svc, logger)
	counter, latency := prometheus.MakeMetrics("auth", "api")
	svc = middleware.NewMetrics(svc, counter, latency)
	svc = middleware.NewTracing(svc, tracer)

	return svc, nil
}
