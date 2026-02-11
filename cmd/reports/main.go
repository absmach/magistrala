// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains reports main function to start the service.
package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"time"

	chclient "github.com/absmach/callhome/pkg/client"
	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/internal/email"
	"github.com/absmach/magistrala/pkg/emailer"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/ticker"
	grpcClient "github.com/absmach/magistrala/readers/api/grpc"
	"github.com/absmach/magistrala/reports"
	httpapi "github.com/absmach/magistrala/reports/api"
	"github.com/absmach/magistrala/reports/middleware"
	"github.com/absmach/magistrala/reports/operations"
	repg "github.com/absmach/magistrala/reports/postgres"
	"github.com/absmach/supermq"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnsvc "github.com/absmach/supermq/pkg/authn/authsvc"
	mgauthz "github.com/absmach/supermq/pkg/authz"
	authzsvc "github.com/absmach/supermq/pkg/authz/authsvc"
	"github.com/absmach/supermq/pkg/callout"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/policies/spicedb"
	pgclient "github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/server"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	spicedbdecoder "github.com/absmach/supermq/pkg/spicedb"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	svcName          = "reports"
	envPrefixDB      = "MG_REPORTS_DB_"
	envPrefixHTTP    = "MG_REPORTS_HTTP_"
	envPrefixCallout = "MG_REPORTS_CALLOUT_"
	envPrefixAuth    = "SMQ_AUTH_GRPC_"
	defDB            = "repo"
	defSvcHTTPPort   = "9017"
	envPrefixGrpc    = "MG_TIMESCALE_READER_GRPC_"
	envPrefixDomains = "SMQ_DOMAINS_GRPC_"
	templatePath     = "template/reports_default_template.html"
	reportEntity     = "report"
)

// We use a buffered channel to prevent blocking, as logging is an expensive operation.
const channBuffer = 256

//go:embed template/reports_default_template.html
var templateFS embed.FS

type config struct {
	LogLevel            string  `env:"MG_REPORTS_LOG_LEVEL"           envDefault:"info"`
	InstanceID          string  `env:"MG_REPORTS_INSTANCE_ID"         envDefault:""`
	JaegerURL           url.URL `env:"SMQ_JAEGER_URL"                 envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool    `env:"SMQ_SEND_TELEMETRY"             envDefault:"true"`
	ESURL               string  `env:"SMQ_ES_URL"                     envDefault:"nats://localhost:4222"`
	TraceRatio          float64 `env:"SMQ_JAEGER_TRACE_RATIO"         envDefault:"1.0"`
	BrokerURL           string  `env:"SMQ_MESSAGE_BROKER_URL"         envDefault:"nats://localhost:4222"`
	DefaultTemplatePath string  `env:"MG_REPORTS_DEFAULT_TEMPLATE"    envDefault:""`
	ConverterURL        string  `env:"MG_PDF_CONVERTER_URL"           envDefault:"http://localhost:4000/pdf"`
	SpicedbHost         string  `env:"SMQ_SPICEDB_HOST"               envDefault:"localhost"`
	SpicedbPort         string  `env:"SMQ_SPICEDB_PORT"               envDefault:"50051"`
	SpicedbPreSharedKey string  `env:"SMQ_SPICEDB_PRE_SHARED_KEY"     envDefault:"12345678"`
	SpicedbSchemaFile   string  `env:"SMQ_SPICEDB_SCHEMA_FILE"        envDefault:"schema.zed"`
	PermissionsFile     string  `env:"SMQ_PERMISSIONS_FILE"           envDefault:"permission.yaml"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

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

	var templateData []byte

	switch cfg.DefaultTemplatePath {
	case "":
		templateData, err = templateFS.ReadFile(templatePath)
	default:
		templateData, err = os.ReadFile(templatePath)
	}

	if err != nil {
		logger.Error(fmt.Sprintf("failed to read report template: %s", err))
		exitCode = 1
		return
	}

	template := reports.ReportTemplate(string(templateData))

	if err := template.Validate(); err != nil {
		logger.Error(fmt.Sprintf("failed to validate report template: %s", err))
		exitCode = 1
		return
	}
	logger.Info("Report template validated successfully")

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

	grpcCfg := grpcclient.Config{}
	if err := env.ParseWithOptions(&grpcCfg, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load auth gRPC client configuration : %s", err))
		exitCode = 1

		return
	}
	authn, authnClient, err := authnsvc.NewAuthentication(ctx, grpcCfg)
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

	authz, authzClient, err := authzsvc.NewAuthorization(ctx, grpcCfg, domAuthz)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authzClient.Close()
	logger.Info("AuthZ  successfully connected to auth gRPC server " + authnClient.Secure())

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

	runInfo := make(chan pkglog.RunInfo, channBuffer)

	svc, err := newService(cfg, database, runInfo, authz, ec, logger, readersClient, template, callout)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create services: %s", err))
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
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
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

func newService(cfg config, db pgclient.Database, runInfo chan pkglog.RunInfo, authz mgauthz.Authorization, ec email.Config, logger *slog.Logger, readersClient grpcReadersV1.ReadersServiceClient, template reports.ReportTemplate, callout callout.Callout) (reports.Service, error) {
	repo := repg.NewRepository(db)
	idp := uuid.New()

	emailClient, err := emailer.New(&ec)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
	}

	policyService, err := newSpiceDBPolicyServiceEvaluator(cfg, logger)
	if err != nil {
		return nil, err
	}
	logger.Info("Policy service successfully connected to SpiceDB gRPC server")

	availableActions, builtInRoles, err := availableActionsAndBuiltInRoles(cfg.SpicedbSchemaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get available actions and built-in roles: %w", err)
	}

	csvc, err := reports.NewService(repo, runInfo, policyService, idp, ticker.NewTicker(time.Second*30), emailClient, readersClient, template, cfg.ConverterURL, availableActions, builtInRoles)
	if err != nil {
		return nil, fmt.Errorf("failed to create reports service: %w", err)
	}

	permConfig, err := permissions.ParsePermissionsFile(cfg.PermissionsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse permissions file: %w", err)
	}

	reportOps, reportRoleOps, err := permConfig.GetEntityPermissions(reportEntity)
	if err != nil {
		return nil, fmt.Errorf("failed to get report permissions: %w", err)
	}

	entitiesOps, err := permissions.NewEntitiesOperations(
		permissions.EntitiesPermission{
			mgPolicies.ReportType: reportOps,
		},
		permissions.EntitiesOperationDetails[permissions.Operation]{
			mgPolicies.ReportType: operations.OperationDetails(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create entities operations: %w", err)
	}

	roleOps, err := permissions.NewOperations(roles.Operations(), reportRoleOps)
	if err != nil {
		return nil, fmt.Errorf("failed to create role operations: %w", err)
	}

	csvc, err = middleware.AuthorizationMiddleware(csvc, authz, entitiesOps, roleOps)
	if err != nil {
		return nil, err
	}
	csvc, err = middleware.NewCallout(csvc, callout, entitiesOps, roleOps)
	if err != nil {
		return nil, err
	}
	csvc = middleware.LoggingMiddleware(csvc, logger)

	return csvc, nil
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
	availableActions, err := spicedbdecoder.GetActionsFromSchema(spicedbSchemaFile, reportEntity)
	if err != nil {
		return []roles.Action{}, map[roles.BuiltInRoleName][]roles.Action{}, err
	}

	builtInRoles := map[roles.BuiltInRoleName][]roles.Action{
		reports.BuiltInRoleAdmin: availableActions,
	}

	return availableActions, builtInRoles, err
}
