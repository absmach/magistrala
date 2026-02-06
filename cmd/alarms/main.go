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

	"github.com/absmach/magistrala/alarms"
	httpAPI "github.com/absmach/magistrala/alarms/api"
	"github.com/absmach/magistrala/alarms/brokers"
	"github.com/absmach/magistrala/alarms/consumer"
	"github.com/absmach/magistrala/alarms/middleware"
	"github.com/absmach/magistrala/alarms/operations"
	alarmsRepo "github.com/absmach/magistrala/alarms/postgres"
	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/prometheus"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authn/authsvc"
	authsvcAuthz "github.com/absmach/supermq/pkg/authz/authsvc"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	"github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/messaging"
	brokerstracing "github.com/absmach/supermq/pkg/messaging/brokers/tracing"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/policies/spicedb"
	"github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	spicedbdecoder "github.com/absmach/supermq/pkg/spicedb"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	svcName          = "alarms"
	envPrefixDB      = "MG_ALARMS_DB_"
	envPrefixHTTP    = "MG_ALARMS_HTTP_"
	envPrefixAuth    = "SMQ_AUTH_GRPC_"
	defDB            = "alarms"
	defSvcHTTPPort   = "8050"
	envPrefixDomains = "SMQ_DOMAINS_GRPC_"
	alarmEntity      = "alarm"
)

type config struct {
	LogLevel            string  `env:"MG_ALARMS_LOG_LEVEL"    envDefault:"info"`
	BrokerURL           string  `env:"SMQ_MESSAGE_BROKER_URL" envDefault:"nats://localhost:4222"`
	InstanceID          string  `env:"MG_ALARMS_INSTANCE_ID"  envDefault:""`
	JaegerURL           url.URL `env:"SMQ_JAEGER_URL"         envDefault:"http://localhost:4318/v1/traces"`
	TraceRatio          float64 `env:"SMQ_JAEGER_TRACE_RATIO" envDefault:"1.0"`
	SpicedbHost         string  `env:"SMQ_SPICEDB_HOST"                 envDefault:"localhost"`
	SpicedbPort         string  `env:"SMQ_SPICEDB_PORT"                 envDefault:"50051"`
	SpicedbPreSharedKey string  `env:"SMQ_SPICEDB_PRE_SHARED_KEY"       envDefault:"12345678"`
	SpicedbSchemaFile   string  `env:"SMQ_SPICEDB_SCHEMA_FILE"          envDefault:"schema.zed"`
	PermissionsFile     string  `env:"SMQ_PERMISSIONS_FILE"             envDefault:"permission.yaml"`
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

	dbConfig := postgres.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
	}

	migrations, err := alarmsRepo.Migration()
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}

	db, err := postgres.Setup(dbConfig, *migrations)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	repo := alarmsRepo.NewAlarmsRepo(db)

	authConfig := grpcclient.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	authn, authnClient, err := authsvc.NewAuthentication(ctx, authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	am := smqauthn.NewAuthNMiddleware(authn)
	defer authnClient.Close()
	logger.Info("AuthN  successfully connected to auth gRPC server " + authnClient.Secure())

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

	authz, authzHandler, err := authsvcAuthz.NewAuthorization(ctx, authConfig, domAuthz)
	if err != nil {
		logger.Error("failed to create authz " + err.Error())
		exitCode = 1
		return
	}
	defer authzHandler.Close()

	logger.Info("AuthZ successfully connected to auth gRPC server " + authzHandler.Secure())

	idp := uuid.New()

	policyService, err := newSpiceDBPolicyServiceEvaluator(cfg, logger)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy service are successfully connected to SpiceDB gRPC server")

	availableActions, buildInRoles, err := availableActionsAndBuiltInRoles(cfg.SpicedbSchemaFile)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get available actions and built-in roles: %s", err))
		exitCode = 1
		return
	}

	svc, err := alarms.NewService(policyService, idp, repo, availableActions, buildInRoles)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s service: %s", svcName, err))
		exitCode = 1
		return
	}

	permConfig, err := permissions.ParsePermissionsFile(cfg.PermissionsFile)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse permissions file: %s", err))
		exitCode = 1
		return
	}

	alarmOps, alarmRoleOps, err := permConfig.GetEntityPermissions(alarmEntity)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get alarm permissions: %s", err))
		exitCode = 1
		return
	}

	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{
			mgPolicies.AlarmsType: alarmOps,
		},
		permissions.EntitiesOperationDetails[permissions.Operation]{
			mgPolicies.AlarmsType: operations.OperationDetails(),
		},
	)

	roleOps, err := permissions.NewOperations(roles.Operations(), alarmRoleOps)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create role operations: %s", err))
		exitCode = 1
		return
	}

	svc, err = middleware.NewAuthorizationMiddleware(svc, authz, entitiesOps, roleOps)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create authorization middleware: %s", err))
		exitCode = 1
		return
	}

	svc = middleware.NewLoggingMiddleware(logger, svc)
	counter, latency := prometheus.MakeMetrics("alarms", "api")
	svc = middleware.NewMetricsMiddleware(counter, latency, svc)
	svc = middleware.NewTracingMiddleware(tracer, svc)

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpAPI.MakeHandler(svc, logger, idp, cfg.InstanceID, am), logger)

	pubSub, err := brokers.NewPubSub(ctx, cfg.BrokerURL, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to message broker: %s", err))
		exitCode = 1
		return
	}
	defer pubSub.Close()
	pubSub = brokerstracing.NewPubSub(httpServerConfig, tracer, pubSub)

	consumer := consumer.NewHandler(svc, logger)

	subCfg := messaging.SubscriberConfig{
		ID:             svcName,
		Topic:          brokers.AllTopic,
		DeliveryPolicy: messaging.DeliverAllPolicy,
		Handler:        consumer,
	}
	if err := pubSub.Subscribe(ctx, subCfg); err != nil {
		logger.Error(fmt.Sprintf("failed to subscribe to message broker: %s", err))
		exitCode = 1

		return
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("billing service terminated: %s", err))
	}
}

func newSpiceDBPolicyServiceEvaluator(cfg config, logger *slog.Logger) (policies.Service, error) {
	client, err := authzed.NewClientWithExperimentalAPIs(
		fmt.Sprintf("%s:%s", cfg.SpicedbHost, cfg.SpicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(cfg.SpicedbPreSharedKey),
	)
	if err != nil {
		return nil, err
	}
	ps := spicedb.NewPolicyService(client, logger)

	return ps, nil
}

func availableActionsAndBuiltInRoles(spicedbSchemaFile string) ([]roles.Action, map[roles.BuiltInRoleName][]roles.Action, error) {
	availableActions, err := spicedbdecoder.GetActionsFromSchema(spicedbSchemaFile, alarmEntity)
	if err != nil {
		return []roles.Action{}, map[roles.BuiltInRoleName][]roles.Action{}, err
	}

	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		alarms.BuiltInRoleAdmin: availableActions,
	}

	return availableActions, builtInRoles, err
}
