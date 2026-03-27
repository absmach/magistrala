// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains groups main function to start the groups service.
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
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcGroupsV1 "github.com/absmach/supermq/api/grpc/groups/v1"
	"github.com/absmach/supermq/auth"
	doperations "github.com/absmach/supermq/domains/operations"
	dpostgres "github.com/absmach/supermq/domains/postgres"
	"github.com/absmach/supermq/groups"
	gpsvc "github.com/absmach/supermq/groups"
	grpcapi "github.com/absmach/supermq/groups/api/grpc"
	httpapi "github.com/absmach/supermq/groups/api/http"
	"github.com/absmach/supermq/groups/events"
	"github.com/absmach/supermq/groups/middleware"
	goperations "github.com/absmach/supermq/groups/operations"
	"github.com/absmach/supermq/groups/postgres"
	pgroups "github.com/absmach/supermq/groups/private"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authsvcAuthn "github.com/absmach/supermq/pkg/authn/authsvc"
	jwksAuthn "github.com/absmach/supermq/pkg/authn/jwks"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	authsvcAuthz "github.com/absmach/supermq/pkg/authz/authsvc"
	"github.com/absmach/supermq/pkg/callout"
	dconsumer "github.com/absmach/supermq/pkg/domains/events/consumer"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
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
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	svcName               = "groups"
	envPrefixDB           = "SMQ_GROUPS_DB_"
	envPrefixHTTP         = "SMQ_GROUPS_HTTP_"
	envPrefixgRPC         = "SMQ_GROUPS_GRPC_"
	envPrefixAuth         = "SMQ_AUTH_GRPC_"
	envPrefixDomains      = "SMQ_DOMAINS_GRPC_"
	envPrefixChannels     = "SMQ_CHANNELS_GRPC_"
	envPrefixClients      = "SMQ_CLIENTS_GRPC_"
	envPrefixGroupCallout = "SMQ_GROUPS_CALLOUT_"
	defDB                 = "groups"
	defSvcHTTPPort        = "9004"
	defSvcgRPCPort        = "7004"
)

type config struct {
	LogLevel            string  `env:"SMQ_GROUPS_LOG_LEVEL"          envDefault:"info"`
	InstanceID          string  `env:"SMQ_GROUPS_INSTANCE_ID"        envDefault:""`
	JaegerURL           url.URL `env:"SMQ_JAEGER_URL"                envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool    `env:"SMQ_SEND_TELEMETRY"            envDefault:"true"`
	ESURL               string  `env:"SMQ_ES_URL"                    envDefault:"nats://localhost:4222"`
	ESConsumerName      string  `env:"SMQ_GROUPS_EVENT_CONSUMER"     envDefault:"groups"`
	TraceRatio          float64 `env:"SMQ_JAEGER_TRACE_RATIO"        envDefault:"1.0"`
	SpicedbHost         string  `env:"SMQ_SPICEDB_HOST"              envDefault:"localhost"`
	SpicedbPort         string  `env:"SMQ_SPICEDB_PORT"              envDefault:"50051"`
	SpicedbSchemaFile   string  `env:"SMQ_SPICEDB_SCHEMA_FILE"       envDefault:"schema.zed"`
	SpicedbPreSharedKey string  `env:"SMQ_SPICEDB_PRE_SHARED_KEY"    envDefault:"12345678"`
	AuthKeyAlgorithm    string  `env:"SMQ_AUTH_KEYS_ALGORITHM"       envDefault:"RS256"`
	JWKSURL             string  `env:"SMQ_AUTH_JWKS_URL"             envDefault:"http://auth:9001/keys/.well-known/jwks.json"`
	PermissionsFile     string  `env:"SMQ_PERMISSIONS_FILE"          envDefault:"permission.yaml"`
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
		exitCode = 1
		return
	}
	gm, err := postgres.Migration()
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	db, err := pgclient.Setup(dbConfig, *gm)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaegerclient.NewProvider(ctx, svcName, cfg.JaegerURL, cfg.InstanceID, cfg.TraceRatio)
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

	authClientConfig := grpcclient.Config{}
	if err := env.ParseWithOptions(&authClientConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
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
		authn, authnClient, err = jwksAuthn.NewAuthentication(ctx, cfg.JWKSURL, authClientConfig)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authnClient.Close()
		logger.Info("AuthN successfully set up jwks authentication on " + cfg.JWKSURL)
	default:
		authn, authnClient, err = authsvcAuthn.NewAuthentication(ctx, authClientConfig)
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
	if err := env.ParseWithOptions(&callCfg, env.Options{Prefix: envPrefixGroupCallout}); err != nil {
		logger.Error(fmt.Sprintf("failed to parse callout config : %s", err))
		exitCode = 1
		return
	}

	authz, authzHandler, err := authsvcAuthz.NewAuthorization(ctx, authClientConfig, domAuthz)
	if err != nil {
		logger.Error("failed to create authz " + err.Error())
		exitCode = 1
		return
	}
	defer authzHandler.Close()
	logger.Info("Authz successfully connected to auth gRPC server " + authzHandler.Secure())

	policyService, err := newPolicyService(cfg, logger)
	if err != nil {
		logger.Error("failed to create new policies service " + err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy client successfully connected to spicedb gRPC server")

	chgrpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&chgrpcCfg, env.Options{Prefix: envPrefixChannels}); err != nil {
		logger.Error(fmt.Sprintf("failed to load channels gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	channelsClient, channelsHandler, err := grpcclient.SetupChannelsClient(ctx, chgrpcCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to channels gRPC server: %s", err))
		exitCode = 1
		return
	}
	defer channelsHandler.Close()
	logger.Info("Groups gRPC client successfully connected to channels gRPC server " + channelsHandler.Secure())

	thgrpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&thgrpcCfg, env.Options{Prefix: envPrefixClients}); err != nil {
		logger.Error(fmt.Sprintf("failed to load clients gRPC client configuration : %s", err))
		exitCode = 1
		return
	}
	clientsClient, clientsHandler, err := grpcclient.SetupClientsClient(ctx, thgrpcCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to clients gRPC server: %s", err))
		exitCode = 1
		return
	}
	defer clientsHandler.Close()
	logger.Info("Clients gRPC client successfully connected to clients gRPC server " + clientsHandler.Secure())

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

	svc, psvc, err := newService(ctx, authz, policyService, db, dbConfig, channelsClient, clientsClient, tracer, logger, cfg, callout, permConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup service: %s", err))
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

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}

	mux := chi.NewRouter()
	idp := uuid.New()
	httpSrv := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(svc, authnMiddleware, mux, logger, cfg.InstanceID, idp), logger)

	grpcServerConfig := server.Config{}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixgRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	registerGroupsServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcGroupsV1.RegisterGroupsServiceServer(srv, grpcapi.NewServer(psvc))
	}
	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerGroupsServer, logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return httpSrv.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSrv)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("groups service terminated: %s", err))
	}
}

func newService(ctx context.Context, authz smqauthz.Authorization, policy policies.Service, db *sqlx.DB,
	dbConfig pgclient.Config, channels grpcChannelsV1.ChannelsServiceClient,
	clients grpcClientsV1.ClientsServiceClient, tracer trace.Tracer, logger *slog.Logger, c config, callout callout.Callout, permConfig *permissions.PermissionConfig,
) (groups.Service, pgroups.Service, error) {
	database := pg.NewDatabase(db, dbConfig, tracer)
	idp := uuid.New()
	sid, err := sid.New()
	if err != nil {
		return nil, nil, err
	}

	availableActions, builtInRoles, err := availableActionsAndBuiltInRoles(c.SpicedbSchemaFile)
	if err != nil {
		return nil, nil, err
	}

	// Creating groups service
	repo := postgres.New(database)
	svc, err := gpsvc.NewService(repo, policy, idp, channels, clients, sid, availableActions, builtInRoles)
	if err != nil {
		return nil, nil, err
	}
	svc, err = events.New(ctx, svc, c.ESURL)
	if err != nil {
		return nil, nil, err
	}

	groupOps, groupRoleOps, err := permConfig.GetEntityPermissions("groups")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get group permissions: %w", err)
	}

	domainOps, _, err := permConfig.GetEntityPermissions("domains")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get domain permissions: %w", err)
	}

	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{
			policies.GroupType:  groupOps,
			policies.DomainType: domainOps,
		},
		permissions.EntitiesOperationDetails[permissions.Operation]{
			policies.GroupType:  goperations.OperationDetails(),
			policies.DomainType: doperations.OperationDetails(),
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create entities operations: %w", err)
	}

	roleOps, err := permissions.NewOperations(roles.Operations(), groupRoleOps)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create role operations: %w", err)
	}

	svc, err = middleware.NewAuthorization(policies.GroupType, svc, authz, repo, entitiesOps, roleOps)
	if err != nil {
		return nil, nil, err
	}

	svc, err = middleware.NewCallout(svc, repo, entitiesOps, roleOps, callout)
	if err != nil {
		return nil, nil, err
	}

	svc = middleware.NewTracing(svc, tracer)
	svc = middleware.NewLogging(svc, logger)
	counter, latency := prometheus.MakeMetrics("groups", "api")
	svc = middleware.NewMetrics(svc, counter, latency)

	psvc := pgroups.New(repo)
	return svc, psvc, err
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
	availableActions, err := spicedbdecoder.GetActionsFromSchema(spicedbSchemaFile, policies.GroupType)
	if err != nil {
		return []roles.Action{}, map[roles.BuiltInRoleName][]roles.Action{}, err
	}

	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		groups.BuiltInRoleAdmin: availableActions,
	}

	return availableActions, builtInRoles, err
}
