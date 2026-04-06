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
	"strings"
	"time"

	grpcCertsV1 "github.com/absmach/magistrala/api/grpc/certs/v1"
	"github.com/absmach/magistrala/certs"
	certsgrpc "github.com/absmach/magistrala/certs/api/grpc"
	httpapi "github.com/absmach/magistrala/certs/api/http"
	"github.com/absmach/magistrala/certs/middleware"
	"github.com/absmach/magistrala/certs/pki"
	"github.com/absmach/magistrala/certs/postgres"
	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	authsvcAuthn "github.com/absmach/magistrala/pkg/authn/authsvc"
	smqauthz "github.com/absmach/magistrala/pkg/authz"
	authsvcAuthz "github.com/absmach/magistrala/pkg/authz/authsvc"
	domainsAuthz "github.com/absmach/magistrala/pkg/domains/grpcclient"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/absmach/magistrala/pkg/jaeger"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	mgserver "github.com/absmach/magistrala/pkg/server"
	grpcserver "github.com/absmach/magistrala/pkg/server/grpc"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/caarlos0/env/v10"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	svcName          = "certs"
	envPrefixHTTP    = "MG_CERTS_HTTP_"
	envPrefixDB      = "MG_CERTS_DB_"
	envPrefixGRPC    = "MG_CERTS_GRPC_"
	envPrefixAuth    = "MG_AUTH_GRPC_"
	envPrefixDomains = "MG_DOMAINS_GRPC_"
	defSvcHTTPPort   = "9010"
	defSvcGRPCPort   = "7012"
	defDB            = "certs"
	serviceTokenKey  = "SERVICE_TOKEN="
)

type config struct {
	LogLevel   string  `env:"MG_CERTS_LOG_LEVEL"            envDefault:"info"`
	JaegerURL  url.URL `env:"MG_JAEGER_URL"                 envDefault:"http://jaeger:4318"`
	InstanceID string  `env:"MG_CERTS_INSTANCE_ID"          envDefault:""`
	TraceRatio float64 `env:"MG_JAEGER_TRACE_RATIO"         envDefault:"1.0"`
	Secret     string  `env:"MG_CERTS_SECRET"               envDefault:""`

	// OpenBao PKI settings
	OpenBaoHost          string `env:"MG_CERTS_OPENBAO_HOST"            envDefault:"http://localhost:8200"`
	OpenBaoAppRole       string `env:"MG_CERTS_OPENBAO_APP_ROLE"        envDefault:""`
	OpenBaoAppSecret     string `env:"MG_CERTS_OPENBAO_APP_SECRET"      envDefault:""`
	OpenBaoNamespace     string `env:"MG_CERTS_OPENBAO_NAMESPACE"       envDefault:""`
	OpenBaoPKIPath       string `env:"MG_CERTS_OPENBAO_PKI_PATH"        envDefault:"pki"`
	OpenBaoRole          string `env:"MG_CERTS_OPENBAO_ROLE"            envDefault:"certs"`
	OpenBaoServiceToken  string `env:"MG_CERTS_SERVICE_TOKEN"           envDefault:""`
	ServiceTokenPath     string `env:"MG_CERTS_SERVICE_TOKEN_PATH"      envDefault:""`
	SecretIDPath         string `env:"MG_CERTS_SECRET_ID_PATH"          envDefault:""`
	SecretRenewThreshold string `env:"MG_CERTS_SECRET_RENEW_THRESHOLD"  envDefault:"24h"`
	SecretIDTTL          string `env:"MG_CERTS_OPENBAO_SECRET_ID_TTL"   envDefault:"72h"`
	SecretCheckInterval  string `env:"MG_CERTS_SECRET_CHECK_INTERVAL"   envDefault:"30s"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := initLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	var exitCode int
	defer mglog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		cfg.InstanceID, err = uuid.New().ID()
		if err != nil {
			logger.Error(fmt.Sprintf("failed to generate instance ID: %v", err))
			exitCode = 1
			return
		}
	}

	if cfg.OpenBaoHost == "" {
		logger.Error("No host specified for OpenBao PKI engine")
		exitCode = 1
		return
	}

	if cfg.OpenBaoAppRole == "" {
		logger.Error("OpenBao AppRole not specified")
		exitCode = 1
		return
	}

	secretID := cfg.OpenBaoAppSecret
	if secretID == "" && cfg.SecretIDPath != "" {
		secretData, err := os.ReadFile(cfg.SecretIDPath)
		if err != nil {
			logger.Error("Failed to read secret ID from file", "path", cfg.SecretIDPath, "error", err)
			exitCode = 1
			return
		}
		secretID = strings.TrimSpace(string(secretData))
	}

	if secretID == "" {
		logger.Error("OpenBao secret ID not specified (provide via MG_CERTS_OPENBAO_APP_SECRET or MG_CERTS_SECRET_ID_PATH)")
		exitCode = 1
		return
	}

	serviceToken := cfg.OpenBaoServiceToken
	if serviceToken == "" && cfg.ServiceTokenPath != "" {
		tokenData, err := os.ReadFile(cfg.ServiceTokenPath)
		if err != nil {
			logger.Warn("Failed to read service token from file, secret renewal will be disabled", "path", cfg.ServiceTokenPath, "error", err)
		} else {
			tokenLine := string(tokenData)
			if strings.HasPrefix(tokenLine, serviceTokenKey) {
				serviceToken = strings.TrimSpace(strings.TrimPrefix(tokenLine, serviceTokenKey))
			}
		}
	}

	pkiAgent, err := pki.NewAgent(cfg.OpenBaoAppRole, secretID, cfg.OpenBaoHost, cfg.OpenBaoNamespace, cfg.OpenBaoPKIPath, cfg.OpenBaoRole, serviceToken, cfg.SecretRenewThreshold, cfg.SecretIDTTL, cfg.SecretCheckInterval, logger)
	if err != nil {
		logger.Error("failed to configure client for OpenBao PKI engine")
		exitCode = 1
		return
	}

	if err := pkiAgent.StartSecretRenewal(ctx); err != nil {
		logger.Warn("Failed to start secret renewal, service may lose access when secret expires", "error", err)
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	migrations := postgres.Migration()
	db, err := pgclient.Setup(dbConfig, *migrations)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaeger.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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

	authClientConfig := grpcclient.Config{}
	if err := env.ParseWithOptions(&authClientConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authn, authnHandler, err := authsvcAuthn.NewAuthentication(ctx, authClientConfig)
	if err != nil {
		logger.Error("failed to create authn " + err.Error())
		exitCode = 1
		return
	}
	defer authnHandler.Close()
	logger.Info("Authn successfully connected to auth gRPC server " + authnHandler.Secure())
	authnMiddleware := smqauthn.NewAuthNMiddleware(authn)
	authz, authzHandler, err := authsvcAuthz.NewAuthorization(ctx, authClientConfig, domAuthz)
	if err != nil {
		logger.Error("failed to create authz " + err.Error())
		exitCode = 1
		return
	}
	defer authzHandler.Close()
	logger.Info("Authz successfully connected to auth gRPC server " + authzHandler.Secure())
	httpServerConfig := mgserver.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	svc := newService(ctx, db, dbConfig, tracer, logger, pkiAgent, authz)

	grpcServerConfig := mgserver.Config{Port: defSvcGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		log.Printf("failed to load %s gRPC server configuration : %s", svcName, err.Error())
		exitCode = 1
		return
	}

	registerCertsServiceServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcCertsV1.RegisterCertsServiceServer(srv, certsgrpc.NewServer(svc))
	}
	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerCertsServiceServer, logger)

	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, authnMiddleware, logger, cfg.InstanceID, cfg.Secret), logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return mgserver.StopSignalHandler(ctx, cancel, logger, svcName, hs, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}

func newService(ctx context.Context, db *sqlx.DB, dbConfig pgclient.Config, tracer trace.Tracer, logger *slog.Logger, pkiAgent certs.Agent, authz smqauthz.Authorization) certs.Service {
	database := pgclient.NewDatabase(db, dbConfig, tracer)
	repo := postgres.NewRepository(database)
	svc, err := certs.NewService(ctx, pkiAgent, repo)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create service: %s", err))
		return nil
	}
	svc = middleware.AuthorizationMiddleware(authz, svc)
	svc = middleware.LoggingMiddleware(svc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.MetricsMiddleware(svc, counter, latency)
	svc = middleware.New(svc, tracer)

	return svc
}

func initLogger(levelText string) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelText)); err != nil {
		return &slog.Logger{}, fmt.Errorf(`{"level":"error","message":"%s: %s","ts":"%s"}`, err, levelText, time.RFC3339Nano)
	}

	logHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(logHandler), nil
}
