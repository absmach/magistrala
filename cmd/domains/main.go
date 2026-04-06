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
	grpcDomainsV1 "github.com/absmach/magistrala/api/grpc/domains/v1"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/domains"
	domainsSvc "github.com/absmach/magistrala/domains"
	domainsgrpcapi "github.com/absmach/magistrala/domains/api/grpc"
	httpapi "github.com/absmach/magistrala/domains/api/http"
	cache "github.com/absmach/magistrala/domains/cache"
	"github.com/absmach/magistrala/domains/events"
	dmw "github.com/absmach/magistrala/domains/middleware"
	doperations "github.com/absmach/magistrala/domains/operations"
	dpostgres "github.com/absmach/magistrala/domains/postgres"
	"github.com/absmach/magistrala/domains/private"
	redisclient "github.com/absmach/magistrala/internal/clients/redis"
	mglog "github.com/absmach/magistrala/logger"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	authsvcAuthn "github.com/absmach/magistrala/pkg/authn/authsvc"
	jwksAuthn "github.com/absmach/magistrala/pkg/authn/jwks"
	"github.com/absmach/magistrala/pkg/authz"
	authsvcAuthz "github.com/absmach/magistrala/pkg/authz/authsvc"
	"github.com/absmach/magistrala/pkg/callout"
	domainsAuthz "github.com/absmach/magistrala/pkg/domains/psvc"
	"github.com/absmach/magistrala/pkg/grpcclient"
	"github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/policies/spicedb"
	"github.com/absmach/magistrala/pkg/postgres"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/server"
	grpcserver "github.com/absmach/magistrala/pkg/server/grpc"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/sid"
	spicedbdecoder "github.com/absmach/magistrala/pkg/spicedb"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	svcName                = "domains"
	envPrefixHTTP          = "MG_DOMAINS_HTTP_"
	envPrefixGrpc          = "MG_DOMAINS_GRPC_"
	envPrefixDB            = "MG_DOMAINS_DB_"
	envPrefixAuth          = "MG_AUTH_GRPC_"
	envPrefixDomainCallout = "MG_DOMAINS_CALLOUT_"
	defDB                  = "domains"
	defSvcHTTPPort         = "9004"
	defSvcGRPCPort         = "7004"
)

type config struct {
	LogLevel            string        `env:"MG_DOMAINS_LOG_LEVEL"            envDefault:"info"`
	JaegerURL           url.URL       `env:"MG_JAEGER_URL"                   envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool          `env:"MG_SEND_TELEMETRY"               envDefault:"true"`
	CacheURL            string        `env:"MG_DOMAINS_CACHE_URL"            envDefault:"redis://localhost:6379/0"`
	CacheKeyDuration    time.Duration `env:"MG_DOMAINS_CACHE_KEY_DURATION"   envDefault:"10m"`
	InstanceID          string        `env:"MG_DOMAINS_INSTANCE_ID"          envDefault:""`
	SpicedbHost         string        `env:"MG_SPICEDB_HOST"                 envDefault:"localhost"`
	SpicedbPort         string        `env:"MG_SPICEDB_PORT"                 envDefault:"50051"`
	SpicedbSchemaFile   string        `env:"MG_SPICEDB_SCHEMA_FILE"          envDefault:"schema.zed"`
	SpicedbPreSharedKey string        `env:"MG_SPICEDB_PRE_SHARED_KEY"       envDefault:"12345678"`
	TraceRatio          float64       `env:"MG_JAEGER_TRACE_RATIO"           envDefault:"1.0"`
	ESURL               string        `env:"MG_ES_URL"                       envDefault:"amqp://guest:guest@localhost:5682/"`
	AuthKeyAlgorithm    string        `env:"MG_AUTH_KEYS_ALGORITHM"          envDefault:"RS256"`
	JWKSURL             string        `env:"MG_AUTH_JWKS_URL"                envDefault:"http://auth:9001/keys/.well-known/jwks.json"`
	PermissionsFile     string        `env:"MG_PERMISSIONS_FILE"             envDefault:"permission.yaml"`
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

	dm, err := dpostgres.Migration()
	if err != nil {
		logger.Error(fmt.Sprintf("failed create migrations for domain: %s", err.Error()))
		exitCode = 1
		return
	}

	db, err := pgclient.Setup(dbConfig, *dm)
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

	time.Sleep(1 * time.Second)

	clientConfig := grpcclient.Config{}
	if err := env.ParseWithOptions(&clientConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC server configuration : %s", err))
		exitCode = 1
		return
	}

	isSymmetric, err := auth.IsSymmetricAlgorithm(cfg.AuthKeyAlgorithm)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse auth key algorithm : %s", err))
		exitCode = 1
		return
	}
	var authn smqauthn.Authentication
	var authnClient grpcclient.Handler
	switch {
	case !isSymmetric:
		authn, authnClient, err = jwksAuthn.NewAuthentication(ctx, cfg.JWKSURL, clientConfig)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authnClient.Close()
		logger.Info("AuthN successfully set up jwks authentication on " + cfg.JWKSURL)
	default:
		authn, authnClient, err = authsvcAuthn.NewAuthentication(ctx, clientConfig)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authnClient.Close()
		logger.Info("AuthN successfully connected to auth gRPC server " + authnClient.Secure())
	}
	authnMiddleware := smqauthn.NewAuthNMiddleware(authn)

	database := postgres.NewDatabase(db, dbConfig, tracer)
	domainsRepo := dpostgres.NewRepository(database)

	cacheclient, err := redisclient.Connect(cfg.CacheURL)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer cacheclient.Close()
	cache := cache.NewDomainsCache(cacheclient, cfg.CacheKeyDuration)

	psvc := private.New(domainsRepo, cache)

	domAuthz := domainsAuthz.NewAuthorization(psvc)

	authz, authzHandler, err := authsvcAuthz.NewAuthorization(ctx, clientConfig, domAuthz)
	if err != nil {
		logger.Error(fmt.Sprintf("authz failed to connect to auth gRPC server : %s", err.Error()))
		exitCode = 1
		return
	}
	defer authzHandler.Close()
	logger.Info("Authz successfully connected to auth gRPC server " + authzHandler.Secure())

	policyService, err := newPolicyService(cfg, logger)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy client successfully connected to spicedb gRPC server")

	callCfg := callout.Config{}
	if err := env.ParseWithOptions(&callCfg, env.Options{Prefix: envPrefixDomainCallout}); err != nil {
		logger.Error(fmt.Sprintf("failed to parse callout config : %s", err))
		exitCode = 1
		return
	}

	call, err := callout.New(callCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create new callout: %s", err))
		exitCode = 1
		return
	}

	svc, err := newDomainService(ctx, domainsRepo, cache, tracer, cfg, authz, policyService, logger, call)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s service: %s", svcName, err.Error()))
		exitCode = 1
		return
	}

	grpcServerConfig := server.Config{Port: defSvcGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGrpc}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	registerDomainsServiceServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcDomainsV1.RegisterDomainsServiceServer(srv, domainsgrpcapi.NewDomainsServer(psvc))
	}

	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerDomainsServiceServer, logger)

	g.Go(func() error {
		return gs.Start()
	})

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	mux := chi.NewMux()
	idp := uuid.New()
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, authnMiddleware, mux, logger, cfg.InstanceID, idp), logger)

	g.Go(func() error {
		return hs.Start()
	})

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("domains service terminated: %s", err))
	}
}

func newDomainService(ctx context.Context, domainsRepo domainsSvc.Repository, cache domainsSvc.Cache, tracer trace.Tracer, cfg config, authz authz.Authorization, policiessvc policies.Service, logger *slog.Logger, callout callout.Callout) (domains.Service, error) {
	idProvider := uuid.New()
	sidProvider, err := sid.New()
	if err != nil {
		return nil, fmt.Errorf("failed to init short id provider : %w", err)
	}

	availableActions, builtInRoles, err := availableActionsAndBuiltInRoles(cfg.SpicedbSchemaFile)
	if err != nil {
		return nil, err
	}

	permConfig, err := permissions.ParsePermissionsFile(cfg.PermissionsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse permissions file: %w", err)
	}

	svc, err := domainsSvc.New(domainsRepo, cache, policiessvc, idProvider, sidProvider, availableActions, builtInRoles)
	if err != nil {
		return nil, fmt.Errorf("failed to init domain service: %w", err)
	}
	svc, err = events.NewEventStoreMiddleware(ctx, svc, cfg.ESURL)
	if err != nil {
		return nil, fmt.Errorf("failed to init domain event store middleware: %w", err)
	}

	domainOps, domainRoleOps, err := permConfig.GetEntityPermissions("domains")
	if err != nil {
		return nil, fmt.Errorf("failed to get domain permissions: %w", err)
	}

	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{policies.DomainType: domainOps},
		permissions.EntitiesOperationDetails[permissions.Operation]{policies.DomainType: doperations.OperationDetails()},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create entities operations: %w", err)
	}

	roleOps, err := permissions.NewOperations(roles.Operations(), domainRoleOps)
	if err != nil {
		return nil, fmt.Errorf("failed to create role operations: %w", err)
	}

	svc, err = dmw.NewAuthorization(policies.DomainType, svc, authz, entitiesOps, roleOps)
	if err != nil {
		return nil, err
	}

	svc, err = dmw.NewCallout(svc, entitiesOps, roleOps, callout)
	if err != nil {
		return nil, err
	}

	counter, latency := prometheus.MakeMetrics("domains", "api")
	svc = dmw.NewMetrics(svc, counter, latency)

	svc = dmw.NewLogging(svc, logger)

	svc = dmw.NewTracing(svc, tracer)
	return svc, nil
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

func availableActionsAndBuiltInRoles(spicedbSchemaFile string) ([]roles.Action, map[roles.BuiltInRoleName][]roles.Action, error) {
	availableActions, err := spicedbdecoder.GetActionsFromSchema(spicedbSchemaFile, policies.DomainType)
	if err != nil {
		return []roles.Action{}, map[roles.BuiltInRoleName][]roles.Action{}, err
	}

	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		domains.BuiltInRoleAdmin: availableActions,
	}

	return availableActions, builtInRoles, err
}
