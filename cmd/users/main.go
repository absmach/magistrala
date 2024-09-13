// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains users main function to start the users service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"regexp"
	"time"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	authSvc "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/email"
	mggroups "github.com/absmach/magistrala/internal/groups"
	gapi "github.com/absmach/magistrala/internal/groups/api"
	gevents "github.com/absmach/magistrala/internal/groups/events"
	gpostgres "github.com/absmach/magistrala/internal/groups/postgres"
	gtracing "github.com/absmach/magistrala/internal/groups/tracing"
	mgpolicy "github.com/absmach/magistrala/internal/policies"
	mglog "github.com/absmach/magistrala/logger"
	authclient "github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/grpcclient"
	jaegerclient "github.com/absmach/magistrala/pkg/jaeger"
	"github.com/absmach/magistrala/pkg/oauth2"
	googleoauth "github.com/absmach/magistrala/pkg/oauth2/google"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/postgres"
	pgclient "github.com/absmach/magistrala/pkg/postgres"
	"github.com/absmach/magistrala/pkg/prometheus"
	"github.com/absmach/magistrala/pkg/server"
	httpserver "github.com/absmach/magistrala/pkg/server/http"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/users"
	capi "github.com/absmach/magistrala/users/api"
	"github.com/absmach/magistrala/users/emailer"
	uevents "github.com/absmach/magistrala/users/events"
	"github.com/absmach/magistrala/users/hasher"
	cmiddleware "github.com/absmach/magistrala/users/middleware"
	clientspg "github.com/absmach/magistrala/users/postgres"
	ctracing "github.com/absmach/magistrala/users/tracing"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/caarlos0/env/v11"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	svcName         = "users"
	envPrefixDB     = "MG_USERS_DB_"
	envPrefixHTTP   = "MG_USERS_HTTP_"
	envPrefixAuth   = "MG_AUTH_GRPC_"
	envPrefixGoogle = "MG_GOOGLE_"
	defDB           = "users"
	defSvcHTTPPort  = "9002"

	streamID = "magistrala.users"
)

type config struct {
	LogLevel            string        `env:"MG_USERS_LOG_LEVEL"           envDefault:"info"`
	AdminEmail          string        `env:"MG_USERS_ADMIN_EMAIL"         envDefault:"admin@example.com"`
	AdminPassword       string        `env:"MG_USERS_ADMIN_PASSWORD"      envDefault:"12345678"`
	PassRegexText       string        `env:"MG_USERS_PASS_REGEX"          envDefault:"^.{8,}$"`
	ResetURL            string        `env:"MG_TOKEN_RESET_ENDPOINT"      envDefault:"/reset-request"`
	JaegerURL           url.URL       `env:"MG_JAEGER_URL"                envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry       bool          `env:"MG_SEND_TELEMETRY"            envDefault:"true"`
	InstanceID          string        `env:"MG_USERS_INSTANCE_ID"         envDefault:""`
	ESURL               string        `env:"MG_ES_URL"                    envDefault:"nats://localhost:4222"`
	TraceRatio          float64       `env:"MG_JAEGER_TRACE_RATIO"        envDefault:"1.0"`
	SelfRegister        bool          `env:"MG_USERS_ALLOW_SELF_REGISTER" envDefault:"false"`
	OAuthUIRedirectURL  string        `env:"MG_OAUTH_UI_REDIRECT_URL"     envDefault:"http://localhost:9095/domains"`
	OAuthUIErrorURL     string        `env:"MG_OAUTH_UI_ERROR_URL"        envDefault:"http://localhost:9095/error"`
	DeleteInterval      time.Duration `env:"MG_USERS_DELETE_INTERVAL"     envDefault:"24h"`
	DeleteAfter         time.Duration `env:"MG_USERS_DELETE_AFTER"        envDefault:"720h"`
	SpicedbHost         string        `env:"MG_SPICEDB_HOST"              envDefault:"localhost"`
	SpicedbPort         string        `env:"MG_SPICEDB_PORT"              envDefault:"50051"`
	SpicedbSchemaFile   string        `env:"MG_SPICEDB_SCHEMA_FILE"       envDefault:"./docker/spicedb/schema.zed"`
	SpicedbPreSharedKey string        `env:"MG_SPICEDB_PRE_SHARED_KEY"    envDefault:"12345678"`
	PassRegex           *regexp.Regexp
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}
	passRegex, err := regexp.Compile(cfg.PassRegexText)
	if err != nil {
		log.Fatalf("invalid password validation rules %s\n", cfg.PassRegexText)
	}
	cfg.PassRegex = passRegex

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

	ec := email.Config{}
	if err := env.Parse(&ec); err != nil {
		logger.Error(fmt.Sprintf("failed to load email configuration : %s", err.Error()))
		exitCode = 1
		return
	}

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	cm := clientspg.Migration()
	gm := gpostgres.Migration()
	cm.Migrations = append(cm.Migrations, gm.Migrations...)
	db, err := pgclient.Setup(dbConfig, *cm)
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

	clientConfig := grpcclient.Config{}
	if err := env.ParseWithOptions(&clientConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authClient, authHandler, err := grpcclient.SetupAuthClient(ctx, clientConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()
	logger.Info("AuthService gRPC client successfully connected to auth gRPC server " + authHandler.Secure())

	authPolicyClient, authPolicyHandler, err := grpcclient.SetupPolicyClient(ctx, clientConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authPolicyHandler.Close()
	logger.Info("PolicyService gRPC client successfully connected to auth gRPC server " + authPolicyHandler.Secure())

	policyClient, err := newPolicyClient(cfg, logger)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy client successfully connected to spicedb gRPC server")

	csvc, gsvc, err := newService(ctx, authClient, authPolicyClient, policyClient, db, dbConfig, tracer, cfg, ec, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup service: %s", err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}

	oauthConfig := oauth2.Config{}
	if err := env.ParseWithOptions(&oauthConfig, env.Options{Prefix: envPrefixGoogle}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s Google configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	oauthProvider := googleoauth.NewProvider(oauthConfig, cfg.OAuthUIRedirectURL, cfg.OAuthUIErrorURL)

	mux := chi.NewRouter()
	httpSrv := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, capi.MakeHandler(csvc, authClient, cfg.SelfRegister, gsvc, mux, logger, cfg.InstanceID, cfg.PassRegex, oauthProvider), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return httpSrv.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSrv)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("users service terminated: %s", err))
	}
}

func newService(ctx context.Context, authClient authclient.AuthClient, authPolicyClient magistrala.PolicyServiceClient, policyClient policies.PolicyClient, db *sqlx.DB, dbConfig pgclient.Config, tracer trace.Tracer, c config, ec email.Config, logger *slog.Logger) (users.Service, groups.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	cRepo := clientspg.NewRepository(database)
	gRepo := gpostgres.New(database)

	idp := uuid.New()
	hsr := hasher.New()

	emailerClient, err := emailer.New(c.ResetURL, &ec)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
	}

	csvc := users.NewService(cRepo, policyClient, emailerClient, hsr, idp)
	gsvc := mggroups.NewService(gRepo, idp, policyClient)

	csvc, err = uevents.NewEventStoreMiddleware(ctx, csvc, c.ESURL)
	if err != nil {
		return nil, nil, err
	}
	gsvc, err = gevents.NewEventStoreMiddleware(ctx, gsvc, c.ESURL, streamID)
	if err != nil {
		return nil, nil, err
	}

	csvc = ctracing.New(csvc, tracer)
	csvc = cmiddleware.LoggingMiddleware(csvc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	csvc = cmiddleware.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = prometheus.MakeMetrics("groups", "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)

	clientID, err := createAdmin(ctx, c, cRepo, hsr, csvc)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create admin client: %s", err))
	}
	if err := createAdminPolicy(ctx, clientID, authClient, policyClient); err != nil {
		return nil, nil, err
	}

	users.NewDeleteHandler(ctx, cRepo, authPolicyClient, c.DeleteInterval, c.DeleteAfter, logger)

	return csvc, gsvc, err
}

func createAdmin(ctx context.Context, c config, crepo clientspg.Repository, hsr users.Hasher, svc users.Service) (string, error) {
	id, err := uuid.New().ID()
	if err != nil {
		return "", err
	}
	hash, err := hsr.Hash(c.AdminPassword)
	if err != nil {
		return "", err
	}

	client := mgclients.Client{
		ID:   id,
		Name: "admin",
		Credentials: mgclients.Credentials{
			Identity: c.AdminEmail,
			Secret:   hash,
		},
		Metadata: mgclients.Metadata{
			"role": "admin",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Role:      mgclients.AdminRole,
		Status:    mgclients.EnabledStatus,
	}

	if c, err := crepo.RetrieveByIdentity(ctx, client.Credentials.Identity); err == nil {
		return c.ID, nil
	}

	// Create an admin
	if _, err = crepo.Save(ctx, client); err != nil {
		return "", err
	}
	if _, err = svc.IssueToken(ctx, c.AdminEmail, c.AdminPassword, ""); err != nil {
		return "", err
	}
	return client.ID, nil
}

func createAdminPolicy(ctx context.Context, clientID string, authClient authclient.AuthClient, policyService policies.PolicyClient) error {
	res, err := authClient.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: authSvc.UserType,
		Subject:     clientID,
		Permission:  authSvc.AdministratorRelation,
		Object:      authSvc.MagistralaObject,
		ObjectType:  authSvc.PlatformType,
	})
	if err != nil || !res.Authorized {
		err := policyService.AddPolicy(ctx, policies.PolicyReq{
			SubjectType: authSvc.UserType,
			Subject:     clientID,
			Relation:    authSvc.AdministratorRelation,
			Object:      authSvc.MagistralaObject,
			ObjectType:  authSvc.PlatformType,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func newPolicyClient(cfg config, logger *slog.Logger) (policies.PolicyClient, error) {
	client, err := authzed.NewClientWithExperimentalAPIs(
		fmt.Sprintf("%s:%s", cfg.SpicedbHost, cfg.SpicedbPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpcutil.WithInsecureBearerToken(cfg.SpicedbPreSharedKey),
	)
	if err != nil {
		return nil, err
	}
	policyClient := mgpolicy.NewPolicyClient(client, logger)

	return policyClient, nil
}
