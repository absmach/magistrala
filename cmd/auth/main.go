package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	api "github.com/mainflux/mainflux/auth/api"
	grpcapi "github.com/mainflux/mainflux/auth/api/grpc"
	httpapi "github.com/mainflux/mainflux/auth/api/http"
	"github.com/mainflux/mainflux/auth/jwt"
	"github.com/mainflux/mainflux/auth/keto"
	authPg "github.com/mainflux/mainflux/auth/postgres"
	"github.com/mainflux/mainflux/auth/tracing"
	"github.com/mainflux/mainflux/internal"
	grpcClient "github.com/mainflux/mainflux/internal/clients/grpc"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	grpcserver "github.com/mainflux/mainflux/internal/server/grpc"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go"
	acl "github.com/ory/keto/proto/ory/keto/acl/v1alpha1"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const (
	svcName        = "auth"
	envPrefix      = "MF_AUTH_"
	envPrefixHttp  = "MF_AUTH_HTTP_"
	envPrefixGrpc  = "MF_AUTH_GRPC_"
	defDB          = "auth"
	defSvcHttpPort = "8180"
	defSvcGrpcPort = "8181"
)

type config struct {
	LogLevel      string        `env:"MF_AUTH_LOG_LEVEL"             envDefault:"info"`
	Secret        string        `env:"MF_AUTH_SECRET"                envDefault:"auth"`
	KetoReadHost  string        `env:"MF_KETO_READ_REMOTE_HOST"      envDefault:"mainflux-keto"`
	KetoReadPort  string        `env:"MF_KETO_READ_REMOTE_PORT"      envDefault:"4466"`
	KetoWriteHost string        `env:"MF_KETO_WRITE_REMOTE_HOST"     envDefault:"mainflux-keto"`
	KetoWritePort string        `env:"MF_KETO_WRITE_REMOTE_PORT"     envDefault:"4467"`
	LoginDuration time.Duration `env:"MF_AUTH_LOGIN_TOKEN_DURATION"  envDefault:"10h"`
	JaegerURL     string        `env:"MF_JAEGER_URL"                 envDefault:"localhost:6831"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create auth service configurations
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Create new postgres client
	dbConfig := pgClient.Config{Name: defDB}
	db, err := pgClient.SetupWithConfig(envPrefix, *authPg.Migration(), dbConfig)
	if err != nil {
		log.Fatalf("failed to setup postgres database : %s", err.Error())
	}
	defer db.Close()

	// Create new tracer for database
	dbTracer, dbCloser, err := jaegerClient.NewTracer("auth_db", cfg.JaegerURL)
	if err != nil {
		log.Fatalf("failed to init Jaeger: %s", err.Error())
	}
	defer dbCloser.Close()

	// Create new keto reader grpc client
	readerConn, _, err := grpcClient.Connect(grpcClient.Config{ClientTLS: false, URL: fmt.Sprintf("%s:%s", cfg.KetoReadHost, cfg.KetoReadPort)})
	if err != nil {
		log.Fatalf("failed to connect to keto gRPC: %s", err.Error())
	}

	// Create new keto writer grpc client
	writerConn, _, err := grpcClient.Connect(grpcClient.Config{ClientTLS: false, URL: fmt.Sprintf("%s:%s", cfg.KetoWriteHost, cfg.KetoWritePort)})
	if err != nil {
		log.Fatalf("failed to connect to keto gRPC: %s", err.Error())
	}

	svc := newService(db, dbTracer, cfg.Secret, logger, readerConn, writerConn, cfg.LoginDuration)

	// Create new HTTP Server
	tracer, closer, err := jaegerClient.NewTracer("auth", cfg.JaegerURL)
	if err != nil {
		log.Fatalf("failed to init Jaeger: %s", err.Error())
	}
	defer closer.Close()

	httpServerConfig := server.Config{Port: defSvcHttpPort}

	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}

	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, tracer, logger), logger)

	// Create new grpc server
	grpcServerConfig := server.Config{Port: defSvcGrpcPort}

	if err := env.Parse(&grpcServerConfig, env.Options{Prefix: envPrefixGrpc, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s gRPC server configuration : %s", svcName, err.Error())
	}
	registerAuthServiceServer := func(srv *grpc.Server) {
		mainflux.RegisterAuthServiceServer(srv, grpcapi.NewServer(tracer, svc))
	}

	gs := grpcserver.New(ctx, cancel, svcName, grpcServerConfig, registerAuthServiceServer, logger)

	// Start servers
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
		logger.Error(fmt.Sprintf("Authentication service terminated: %s", err))
	}
}

func newService(db *sqlx.DB, tracer opentracing.Tracer, secret string, logger logger.Logger, readerConn, writerConn *grpc.ClientConn, duration time.Duration) auth.Service {
	database := authPg.NewDatabase(db)
	keysRepo := tracing.New(authPg.New(database), tracer)

	groupsRepo := authPg.NewGroupRepo(database)
	groupsRepo = tracing.GroupRepositoryMiddleware(tracer, groupsRepo)

	pa := keto.NewPolicyAgent(acl.NewCheckServiceClient(readerConn), acl.NewWriteServiceClient(writerConn), acl.NewReadServiceClient(readerConn))

	idProvider := uuid.New()
	t := jwt.New(secret)

	svc := auth.New(keysRepo, groupsRepo, idProvider, t, pa, duration)
	svc = api.LoggingMiddleware(svc, logger)

	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
