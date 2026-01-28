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
	"github.com/absmach/supermq"
	grpcDomainsV1 "github.com/absmach/supermq/api/grpc/domains/v1"
	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	grpcUsersV1 "github.com/absmach/supermq/api/grpc/users/v1"
	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/internal/email"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authsvcAuthn "github.com/absmach/supermq/pkg/authn/authsvc"
	jwksAuthn "github.com/absmach/supermq/pkg/authn/jwks"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	authsvcAuthz "github.com/absmach/supermq/pkg/authz/authsvc"
	domainsAuthz "github.com/absmach/supermq/pkg/domains/grpcclient"
	"github.com/absmach/supermq/pkg/grpcclient"
	jaegerclient "github.com/absmach/supermq/pkg/jaeger"
	"github.com/absmach/supermq/pkg/oauth2"
	googleoauth "github.com/absmach/supermq/pkg/oauth2/google"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/policies/spicedb"
	pg "github.com/absmach/supermq/pkg/postgres"
	pgclient "github.com/absmach/supermq/pkg/postgres"
	"github.com/absmach/supermq/pkg/prometheus"
	"github.com/absmach/supermq/pkg/server"
	grpcserver "github.com/absmach/supermq/pkg/server/grpc"
	httpserver "github.com/absmach/supermq/pkg/server/http"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/absmach/supermq/users"
	httpapi "github.com/absmach/supermq/users/api"
	grpcapi "github.com/absmach/supermq/users/api/grpc"
	"github.com/absmach/supermq/users/emailer"
	"github.com/absmach/supermq/users/events"
	"github.com/absmach/supermq/users/hasher"
	"github.com/absmach/supermq/users/middleware"
	"github.com/absmach/supermq/users/postgres"
	pusers "github.com/absmach/supermq/users/private"
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
	svcName          = "users"
	envPrefixDB      = "SMQ_USERS_DB_"
	envPrefixHTTP    = "SMQ_USERS_HTTP_"
	envPrefixGRPC    = "SMQ_USERS_GRPC_"
	envPrefixAuth    = "SMQ_AUTH_GRPC_"
	envPrefixDomains = "SMQ_DOMAINS_GRPC_"
	envPrefixGoogle  = "SMQ_GOOGLE_"
	defDB            = "users"
	defSvcHTTPPort   = "9002"
	defSvcGRPCPort   = "7002"
)

type config struct {
	LogLevel                   string        `env:"SMQ_USERS_LOG_LEVEL"                   envDefault:"info"`
	AdminEmail                 string        `env:"SMQ_USERS_ADMIN_EMAIL"                 envDefault:"admin@example.com"`
	AdminPassword              string        `env:"SMQ_USERS_ADMIN_PASSWORD"              envDefault:"12345678"`
	AdminUsername              string        `env:"SMQ_USERS_ADMIN_USERNAME"              envDefault:"admin"`
	AdminFirstName             string        `env:"SMQ_USERS_ADMIN_FIRST_NAME"            envDefault:"super"`
	AdminLastName              string        `env:"SMQ_USERS_ADMIN_LAST_NAME"             envDefault:"admin"`
	PassRegexText              string        `env:"SMQ_USERS_PASS_REGEX"                  envDefault:"^.{8,}$"`
	JaegerURL                  url.URL       `env:"SMQ_JAEGER_URL"                        envDefault:"http://localhost:4318/v1/traces"`
	SendTelemetry              bool          `env:"SMQ_SEND_TELEMETRY"                    envDefault:"true"`
	InstanceID                 string        `env:"SMQ_USERS_INSTANCE_ID"                 envDefault:""`
	ESURL                      string        `env:"SMQ_ES_URL"                            envDefault:"nats://localhost:4222"`
	TraceRatio                 float64       `env:"SMQ_JAEGER_TRACE_RATIO"                envDefault:"1.0"`
	SelfRegister               bool          `env:"SMQ_USERS_ALLOW_SELF_REGISTER"         envDefault:"false"`
	OAuthUIRedirectURL         string        `env:"SMQ_OAUTH_UI_REDIRECT_URL"             envDefault:"http://localhost:9095/domains"`
	OAuthUIErrorURL            string        `env:"SMQ_OAUTH_UI_ERROR_URL"                envDefault:"http://localhost:9095/error"`
	DeleteInterval             time.Duration `env:"SMQ_USERS_DELETE_INTERVAL"             envDefault:"24h"`
	DeleteAfter                time.Duration `env:"SMQ_USERS_DELETE_AFTER"                envDefault:"720h"`
	SpicedbHost                string        `env:"SMQ_SPICEDB_HOST"                      envDefault:"localhost"`
	SpicedbPort                string        `env:"SMQ_SPICEDB_PORT"                      envDefault:"50051"`
	SpicedbPreSharedKey        string        `env:"SMQ_SPICEDB_PRE_SHARED_KEY"            envDefault:"12345678"`
	PasswordResetURLPrefix     string        `env:"SMQ_PASSWORD_RESET_URL_PREFIX"         envDefault:"http://localhost/password/reset"`
	PasswordResetEmailTemplate string        `env:"SMQ_PASSWORD_RESET_EMAIL_TEMPLATE"     envDefault:"reset-password-email.tmpl"`
	VerificationURLPrefix      string        `env:"SMQ_VERIFICATION_URL_PREFIX"           envDefault:"http://localhost/verify-email"`
	VerificationEmailTemplate  string        `env:"SMQ_VERIFICATION_EMAIL_TEMPLATE"       envDefault:"verification-email.tmpl"`
	AuthKeyAlgorithm           string        `env:"SMQ_AUTH_KEYS_ALGORITHM"               envDefault:"RS256"`
	JWKSURL                    string        `env:"SMQ_AUTH_JWKS_URL"                     envDefault:"http://auth:9001/keys/.well-known/jwks.json"`
	PassRegex                  *regexp.Regexp
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

	resetPasswordEmailConfig := email.Config{}
	if err := env.Parse(&resetPasswordEmailConfig); err != nil {
		logger.Error(fmt.Sprintf("failed to load reset password email configuration : %s", err.Error()))
		exitCode = 1
		return
	}
	resetPasswordEmailConfig.Template = cfg.PasswordResetEmailTemplate

	verificationEmailConfig := email.Config{}
	if err := env.Parse(&verificationEmailConfig); err != nil {
		logger.Error(fmt.Sprintf("failed to load verification password email configuration : %s", err.Error()))
		exitCode = 1
		return
	}
	verificationEmailConfig.Template = cfg.VerificationEmailTemplate

	dbConfig := pgclient.Config{Name: defDB}
	if err := env.ParseWithOptions(&dbConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}

	migration := postgres.Migration()
	db, err := pgclient.Setup(dbConfig, *migration)
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

	database := pg.NewDatabase(db, dbConfig, tracer)
	repo := postgres.NewRepository(database)

	authClientConfig := grpcclient.Config{}
	if err := env.ParseWithOptions(&authClientConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	tokenClient, tokenHandler, err := grpcclient.SetupTokenClient(ctx, authClientConfig)
	if err != nil {
		logger.Error("failed to create token gRPC client " + err.Error())
		exitCode = 1
		return
	}
	defer tokenHandler.Close()
	logger.Info("Token service client successfully connected to auth gRPC server " + tokenHandler.Secure())

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
	domAuthz, domainsClient, domainsHandler, err := domainsAuthz.NewAuthorization(ctx, domsGrpcCfg)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer domainsHandler.Close()

	authz, authzHandler, err := authsvcAuthz.NewAuthorization(ctx, authClientConfig, domAuthz)
	if err != nil {
		logger.Error("failed to create authz " + err.Error())
		exitCode = 1
		return
	}
	defer authzHandler.Close()
	logger.Info("AuthZ successfully connected to auth gRPC server " + authzHandler.Secure())

	policyService, err := newPolicyService(cfg, logger)
	if err != nil {
		logger.Error("failed to create new policies service " + err.Error())
		exitCode = 1
		return
	}
	logger.Info("Policy client successfully connected to spicedb gRPC server")

	csvc, err := newService(ctx, authz, tokenClient, policyService, domainsClient, repo, tracer, cfg, resetPasswordEmailConfig, verificationEmailConfig, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup service: %s", err))
		exitCode = 1
		return
	}

	psvc := pusers.New(repo)

	grpcServerConfig := server.Config{Port: defSvcGRPCPort}
	if err := env.ParseWithOptions(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}

	registerUsersServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		grpcUsersV1.RegisterUsersServiceServer(srv, grpcapi.NewServer(psvc))
	}
	gs := grpcserver.NewServer(ctx, cancel, svcName, grpcServerConfig, registerUsersServer, logger)

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
	idp := uuid.New()
	httpSrv := httpserver.NewServer(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(csvc, authnMiddleware, tokenClient, cfg.SelfRegister, mux, logger, cfg.InstanceID, cfg.PassRegex, idp, oauthProvider), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, supermq.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return httpSrv.Start()
	})

	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, httpSrv, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("users service terminated: %s", err))
	}
}

func newService(ctx context.Context, authz smqauthz.Authorization, token grpcTokenV1.TokenServiceClient, policyService policies.Service, domainsClient grpcDomainsV1.DomainsServiceClient, repo users.Repository, tracer trace.Tracer, c config, resetPasswordEmailConfig, verificationEmailConfig email.Config, logger *slog.Logger) (users.Service, error) {
	idp := uuid.New()
	hsr := hasher.New()

	// Creating users service
	emailerClient, err := emailer.New(
		c.PasswordResetURLPrefix,
		c.VerificationURLPrefix,
		&resetPasswordEmailConfig,
		&verificationEmailConfig,
	)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
		return nil, err
	}

	svc := users.NewService(token, repo, policyService, emailerClient, hsr, idp)

	svc, err = events.NewEventStoreMiddleware(ctx, svc, c.ESURL)
	if err != nil {
		return nil, err
	}
	svc = middleware.NewAuthorization(svc, authz, c.SelfRegister)

	svc = middleware.NewTracing(svc, tracer)
	svc = middleware.NewLogging(svc, logger)
	counter, latency := prometheus.MakeMetrics(svcName, "api")
	svc = middleware.NewMetrics(svc, counter, latency)

	userID, err := createAdmin(ctx, c, repo, hsr, svc)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create admin client: %s", err))
	}
	if err := createAdminPolicy(ctx, userID, authz, policyService); err != nil {
		return nil, err
	}

	users.NewDeleteHandler(ctx, repo, policyService, domainsClient, c.DeleteInterval, c.DeleteAfter, logger)

	return svc, err
}

func createAdmin(ctx context.Context, c config, repo users.Repository, hsr users.Hasher, svc users.Service) (string, error) {
	id, err := uuid.New().ID()
	if err != nil {
		return "", err
	}
	hash, err := hsr.Hash(c.AdminPassword)
	if err != nil {
		return "", err
	}

	user := users.User{
		ID:        id,
		Email:     c.AdminEmail,
		FirstName: c.AdminFirstName,
		LastName:  c.AdminLastName,
		Credentials: users.Credentials{
			Username: "admin",
			Secret:   hash,
		},
		Metadata: users.Metadata{
			"role": "admin",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Role:      users.AdminRole,
		Status:    users.EnabledStatus,
	}

	if u, err := repo.RetrieveByEmail(ctx, user.Email); err == nil {
		return u.ID, nil
	}

	// Create an admin
	if _, err = repo.Save(ctx, user); err != nil {
		return "", err
	}
	if _, err = svc.IssueToken(ctx, c.AdminUsername, c.AdminPassword); err != nil {
		return "", err
	}
	return user.ID, nil
}

func createAdminPolicy(ctx context.Context, userID string, authz smqauthz.Authorization, policyService policies.Service) error {
	if err := authz.Authorize(ctx, smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  policies.AdministratorRelation,
		Object:      policies.SuperMQObject,
		ObjectType:  policies.PlatformType,
	}, nil); err != nil {
		err := policyService.AddPolicy(ctx, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			Object:      policies.SuperMQObject,
			ObjectType:  policies.PlatformType,
		})
		if err != nil {
			return err
		}
	}
	return nil
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
