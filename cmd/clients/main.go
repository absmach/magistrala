// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains clients main function to start the clients service.
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
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcGroupsV1 "github.com/absmach/supermq/api/grpc/groups/v1"
	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/clients"
	grpcapi "github.com/absmach/supermq/clients/api/grpc"
	httpapi "github.com/absmach/supermq/clients/api/http"
	"github.com/absmach/supermq/clients/cache"
	"github.com/absmach/supermq/clients/events"
	"github.com/absmach/supermq/clients/middleware"
	clientsOps "github.com/absmach/supermq/clients/operations"
	"github.com/absmach/supermq/clients/postgres"
	pClients "github.com/absmach/supermq/clients/private"
	doperations "github.com/absmach/supermq/domains/operations"
	dpostgres "github.com/absmach/supermq/domains/postgres"
	goperations "github.com/absmach/supermq/groups/operations"
	gpostgres "github.com/absmach/supermq/groups/postgres"
	redisclient "github.com/absmach/supermq/internal/clients/redis"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authsvcAuthn "github.com/absmach/supermq/pkg/authn/authsvc"
	jwksAuthn "github.com/absmach/supermq/pkg/authn/jwks"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	authsvcAuthz "github.com/absmach/supermq/pkg/authz/authsvc"
	"github.com/absmach/supermq/pkg/callout"
	dconsumer "github.com/absmach/supermq/pkg/domains/events/consumer"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	gconsumer "github.com/absmach/supermq/pkg/groups/events/consumer"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/policies/spicedb"
	pg "github.com/absmach/supermq/pkg/postgres"
	pgclient "github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/server"
	grpcserver "github.com/absmach/supermq/pkg/server/grpc"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/sid"
	spicedbdecoder "github.com/absmach/supermq/pkg/spicedb"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	svcName                = "clients"
	envPrefixDB            = "SMQ_CLIENTS_DB_"
	envPrefixHTTP          = "SMQ_CLIENTS_HTTP_"
	envPrefixGRPC          = "SMQ_CLIENTS_GRPC_"
	envPrefixAuth          = "SMQ_AUTH_GRPC_"
	envPrefixChannels      = "SMQ_CHANNELS_GRPC_"
	envPrefixGroups        = "SMQ_GROUPS_GRPC_"
	envPrefixDomains       = "SMQ_DOMAINS_GRPC_"
	envPrefixClientCallout = "SMQ_CLIENTS_CALLOUT_"
	defDB                  = "clients"
	defSvcHTTPPort         = "9000"
	defSvcAuthGRPCPort     = "7000"
)

type config struct {
	InstanceID          string        `env:"SMQ_CLIENTS_INSTANCE_ID"        envDefault:""`
	LogLevel            string        `env:"SMQ_CLIENTS_LOG_LEVEL"          envDefault:"info"`
	StandaloneID        string        `env:"SMQ_CLIENTS_STANDALONE_ID"      envDefault:""`
	StandaloneToken     string        `env:"SMQ_CLIENTS_STANDALONE_TOKEN"   envDefault:""`
	CacheURL            string        `env:"SMQ_CLIENTS_CACHE_URL"          envDefault:"redis://localhost:6379/0"`
	CacheKeyDuration    time.Duration `env:"SMQ_CLIENTS_CACHE_KEY_DURATION" envDefault:"10m"`
	JaegerURL           url.URL       `env:"SMQ_JAEGER_URL"                 envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool          `env:"SMQ_SEND_TELEMETRY"             envDefault:"true"`
	ESURL               string        `env:"SMQ_ES_URL"                     envDefault:"nats://localhost:4222"`
	ESConsumerName      string        `env:"SMQ_CLIENTS_EVENT_CONSUMER"     envDefault:"clients"`
	TraceRatio          float64       `env:"SMQ_JAEGER_TRACE_RATIO"         envDefault:"1.0"`
	SpicedbHost         string        `env:"SMQ_SPICEDB_HOST"               envDefault:"localhost"`
	SpicedbPort         string        `env:"SMQ_SPICEDB_PORT"               envDefault:"50051"`
	SpicedbPreSharedKey string        `env:"SMQ_SPICEDB_PRE_SHARED_KEY"     envDefault:"12345678"`
	SpicedbSchemaFile   string        `env:"SMQ_SPICEDB_SCHEMA_FILE"        envDefault:"schema.zed"`
	AuthKeyAlgorithm    string        `env:"SMQ_AUTH_KEYS_ALGORITHM"        envDefault:"RS256"`
	JWKSURL             string        `env:"SMQ_AUTH_JWKS_URL"              envDefault:"http://auth:9001/keys/.well-known/jwks.json"`
	PermissionsFile     string        `env:"SMQ_PERMISSIONS_FILE"           envDefault:"permission.yaml"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new clients configuration
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

	// Create new database for clients
	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	tm, err := postgres.Migration()
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	db, err := pgclient.Setup(dbConfig, *tm)
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

	// Setup new redis cache client
	cacheclient, err := redisclient.Connect(cfg.CacheURL)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer cacheclient.Close()

	policyEvaluator, policyService, err := newSpiceDBPolicyServiceEvaluator(cfg, logger)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy evaluator and Policy manager are successfully connected to SpiceDB gRPC server")

	grpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&grpcCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC client configuration : %s", err))
		exitCode = 1
		return
	}

	alg, err := auth.IsSymmetricAlgorithm(cfg.AuthKeyAlgorithm)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse auth key algorithm : %s", err))
		exitCode = 1
		return
	}
	var authn smqauthn.Authentication
	var authnClient grpcclient.Handler
	switch {
	case !alg:
		authn, authnClient, err = jwksAuthn.NewAuthentication(ctx, cfg.JWKSURL, grpcCfg)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authnClient.Close()
		logger.Info("AuthN successfully set up jwks authentication on " + cfg.JWKSURL)
	default:
		authn, authnClient, err = authsvcAuthn.NewAuthentication(ctx, grpcCfg)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authnClient.Close()
		logger.Info("AuthN successfully connected to auth gRPC server " + authnClient.Secure())
	}
	authnMiddleware := smqauthn.NewAuthNMiddleware(authn)

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

	callCfg := callout.Config{}
	if err := env.ParseWithOptions(&callCfg, env.Options{Prefix: envPrefixClientCallout}); err != nil {
		logger.Error(fmt.Sprintf("failed to parse callout config : %s", err))
		exitCode = 1
		return
	}

	authz, authzClient, err := authsvcAuthz.NewAuthorization(ctx, grpcCfg, domAuthz)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authzClient.Close()
	logger.Info("AuthZ  successfully connected to auth gRPC server " + authzClient.Secure())

	chgrpccfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&chgrpccfg, env.Options{Prefix: envPrefixChannels}); err != nil {
		logger.Error(fmt.Sprintf("failed to load channels gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	channelsgRPC, channelsClient, err := grpcclient.SetupChannelsClient(ctx, chgrpccfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Channels gRPC client successfully connected to channels gRPC server " + channelsClient.Secure())
	defer channelsClient.Close()

	groupsgRPCCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&groupsgRPCCfg, env.Options{Prefix: envPrefixGroups}); err != nil {
		logger.Error(fmt.Sprintf("failed to load groups gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	groupsClient, groupsHandler, err := grpcclient.SetupGroupsClient(ctx, groupsgRPCCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to groups gRPC server: %s", err))
		exitCode = 1
		return
	}
	defer groupsHandler.Close()
	logger.Info("Groups gRPC client successfully connected to groups gRPC server " + groupsHandler.Secure())

	callout, err := callout.New(callCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create new callout: %s", err))
		exitCode = 1
		return
	}

	permConfig, err := permissions.ParsePermissionsFile(cfg.PermissionsFile)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse permissions file: %s", err))
		exitCode = 1
		return
	}

	svc, psvc, err := newService(ctx, db, dbConfig, authz, policyEvaluator, policyService, cacheclient,
		cfg, channelsgRPC, groupsClient, tracer, logger, callout, permConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create services: %s", err))
		exitCode = 1
		return
	}

	ddatabase := pg.NewDatabase(db, dbConfig, tracer)
	drepo := dpostgres.NewRepository(ddatabase)

	if err := dconsumer.DomainsEventsSubscribe(ctx, drepo, svc, cfg.ESURL, cfg.ESConsumerName, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to create domains event store : %s", err))
		exitCode = 1
		return
	}

	gdatabase := pg.NewDatabase(db, dbConfig, tracer)
	grepo := gpostgres.New(gdatabase)

	if err := gconsumer.GroupsEventsSubscribe(ctx, grepo, cfg.ESURL, cfg.ESConsumerName, logger); err != nil {
		logger.Error(fmt.Sprintf("failed to create groups event store : %s", err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	mux := chi.NewRouter()
	idp := uuid.New()
	httpSvc := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, authnMiddleware, mux, logger, cfg.InstanceID, idp), logger)

	grpcServerConfig := server.Config{Port: defSvcAuthGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	registerClientsServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcClientsV1.RegisterClientsServiceServer(srv, grpcapi.NewServer(psvc))
	}
	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerClientsServer, logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	// Start all servers
	g.Go(func() error {
		return httpSvc.Start()
	})

	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSvc)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}

func newService(ctx context.Context, db *sqlx.DB, dbConfig pgclient.Config, authz smqauthz.Authorization, pe policies.Evaluator, ps policies.Service, cacheClient *redis.Client, cfg config, channels grpcChannelsV1.ChannelsServiceClient, groups grpcGroupsV1.GroupsServiceClient, tracer trace.Tracer, logger *slog.Logger, callout callout.Callout, permConfig *permissions.PermissionConfig) (clients.Service, pClients.Service, error) {
	database := pg.NewDatabase(db, dbConfig, tracer)
	repo := postgres.NewRepository(database)

	idp := uuid.New()
	sidp, err := sid.New()
	if err != nil {
		return nil, nil, err
	}

	// Clients service
	cache := cache.NewCache(cacheClient, cfg.CacheKeyDuration)

	availableActions, builtInRoles, err := availableActionsAndBuiltInRoles(cfg.SpicedbSchemaFile)
	if err != nil {
		return nil, nil, err
	}

	csvc, err := clients.NewService(repo, ps, cache, channels, groups, idp, sidp, availableActions, builtInRoles)
	if err != nil {
		return nil, nil, err
	}

	csvc, err = events.NewEventStoreMiddleware(ctx, csvc, cfg.ESURL)
	if err != nil {
		return nil, nil, err
	}

	csvc = middleware.NewTracing(csvc, tracer)

	counter, latency := prometheus.MakeMetrics(svcName, "api")
	csvc = middleware.NewMetrics(csvc, counter, latency)

	clientOps, clientRoleOps, err := permConfig.GetEntityPermissions("clients")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get client permissions: %w", err)
	}

	domainOps, _, err := permConfig.GetEntityPermissions("domains")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get domain permissions: %w", err)
	}

	groupOps, _, err := permConfig.GetEntityPermissions("groups")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get group permissions: %w", err)
	}

	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{
			policies.ClientType: clientOps,
			policies.DomainType: domainOps,
			policies.GroupType:  groupOps,
		},
		permissions.EntitiesOperationDetails[permissions.Operation]{
			policies.ClientType: clientsOps.OperationDetails(),
			policies.DomainType: doperations.OperationDetails(),
			policies.GroupType:  goperations.OperationDetails(),
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create entities operations: %w", err)
	}

	roleOps, err := permissions.NewOperations(roles.Operations(), clientRoleOps)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create role operations: %w", err)
	}

	csvc, err = middleware.NewAuthorization(policies.ClientType, csvc, authz, repo, entitiesOps, roleOps)
	if err != nil {
		return nil, nil, err
	}

	csvc, err = middleware.NewCallout(csvc, repo, entitiesOps, roleOps, callout)
	if err != nil {
		return nil, nil, err
	}

	csvc = middleware.NewLogging(csvc, logger)

	isvc := pClients.New(repo, cache, pe, ps)

	return csvc, isvc, err
}

func newSpiceDBPolicyServiceEvaluator(cfg config, logger *slog.Logger) (policies.Evaluator, policies.Service, error) {
	client, err := authzed.NewClientWithExperimentalAPIs(
		fmt.Sprintf("%s:%s", cfg.SpicedbHost, cfg.SpicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(cfg.SpicedbPreSharedKey),
	)
	if err != nil {
		return nil, nil, err
	}
	pe := spicedb.NewPolicyEvaluator(client, logger)
	ps := spicedb.NewPolicyService(client, logger)

	return pe, ps, nil
}

func availableActionsAndBuiltInRoles(spicedbSchemaFile string) ([]roles.Action, map[roles.BuiltInRoleName][]roles.Action, error) {
	availableActions, err := spicedbdecoder.GetActionsFromSchema(spicedbSchemaFile, policies.ClientType)
	if err != nil {
		return []roles.Action{}, map[roles.BuiltInRoleName][]roles.Action{}, err
	}

	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		clients.BuiltInRoleAdmin: availableActions,
	}

	return availableActions, builtInRoles, err
}
