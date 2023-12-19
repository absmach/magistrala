// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains users main function to start the users service.
package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/absmach/magistrala"
	authSvc "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal"
	jaegerclient "github.com/absmach/magistrala/internal/clients/jaeger"
	pgclient "github.com/absmach/magistrala/internal/clients/postgres"
	"github.com/absmach/magistrala/internal/email"
	mggroups "github.com/absmach/magistrala/internal/groups"
	gapi "github.com/absmach/magistrala/internal/groups/api"
	gevents "github.com/absmach/magistrala/internal/groups/events"
	gpostgres "github.com/absmach/magistrala/internal/groups/postgres"
	gtracing "github.com/absmach/magistrala/internal/groups/tracing"
	"github.com/absmach/magistrala/internal/postgres"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/users"
	capi "github.com/absmach/magistrala/users/api"
	"github.com/absmach/magistrala/users/emailer"
	uevents "github.com/absmach/magistrala/users/events"
	"github.com/absmach/magistrala/users/hasher"
	clientspg "github.com/absmach/magistrala/users/postgres"
	ctracing "github.com/absmach/magistrala/users/tracing"
	"github.com/caarlos0/env/v10"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "users"
	envPrefixDB    = "MG_USERS_DB_"
	envPrefixHTTP  = "MG_USERS_HTTP_"
	envPrefixAuth  = "MG_AUTH_GRPC_"
	defDB          = "users"
	defSvcHTTPPort = "9002"

	streamID = "magistrala.users"
)

type config struct {
	LogLevel      string  `env:"MG_USERS_LOG_LEVEL"              envDefault:"info"`
	AdminEmail    string  `env:"MG_USERS_ADMIN_EMAIL"            envDefault:"admin@example.com"`
	AdminPassword string  `env:"MG_USERS_ADMIN_PASSWORD"         envDefault:"12345678"`
	PassRegexText string  `env:"MG_USERS_PASS_REGEX"             envDefault:"^.{8,}$"`
	ResetURL      string  `env:"MG_TOKEN_RESET_ENDPOINT"         envDefault:"/reset-request"`
	JaegerURL     url.URL `env:"MG_JAEGER_URL"                   envDefault:"http://localhost:14268/api/traces"`
	SendTelemetry bool    `env:"MG_SEND_TELEMETRY"               envDefault:"true"`
	InstanceID    string  `env:"MG_USERS_INSTANCE_ID"            envDefault:""`
	ESURL         string  `env:"MG_ES_URL"                       envDefault:"nats://localhost:4222"`
	TraceRatio    float64 `env:"MG_JAEGER_TRACE_RATIO"           envDefault:"1.0"`
	SelfRegister  bool    `env:"MG_USERS_ALLOW_SELF_REGISTER"    envDefault:"false"`
	PassRegex     *regexp.Regexp
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
		logger.Fatal(fmt.Sprintf("failed to init logger: %s", err.Error()))
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

	authConfig := auth.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	authClient, authHandler, err := auth.Setup(authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	csvc, gsvc, err := newService(ctx, authClient, db, dbConfig, tracer, cfg, ec, logger)
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

	mux := chi.NewRouter()
	httpSrv := httpserver.New(ctx, cancel, svcName, httpServerConfig, capi.MakeHandler(csvc, gsvc, mux, logger, cfg.InstanceID), logger)

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

func newService(ctx context.Context, authClient magistrala.AuthServiceClient, db *sqlx.DB, dbConfig pgclient.Config, tracer trace.Tracer, c config, ec email.Config, logger mglog.Logger) (users.Service, groups.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	cRepo := clientspg.NewRepository(database)
	gRepo := gpostgres.New(database)

	idp := uuid.New()
	hsr := hasher.New()

	emailerClient, err := emailer.New(c.ResetURL, &ec)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
	}

	csvc := users.NewService(cRepo, authClient, emailerClient, hsr, idp, c.PassRegex, c.SelfRegister)
	gsvc := mggroups.NewService(gRepo, idp, authClient)

	csvc, err = uevents.NewEventStoreMiddleware(ctx, csvc, c.ESURL)
	if err != nil {
		return nil, nil, err
	}
	gsvc, err = gevents.NewEventStoreMiddleware(ctx, gsvc, c.ESURL, streamID)
	if err != nil {
		return nil, nil, err
	}

	csvc = ctracing.New(csvc, tracer)
	csvc = capi.LoggingMiddleware(csvc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	csvc = capi.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = internal.MakeMetrics("groups", "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)

	clientID, err := createAdmin(ctx, c, cRepo, hsr, csvc)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create admin client: %s", err))
	}
	if err := createAdminPolicy(ctx, clientID, authClient); err != nil {
		return nil, nil, err
	}
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

func createAdminPolicy(ctx context.Context, clientID string, authClient magistrala.AuthServiceClient) error {
	res, err := authClient.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: authSvc.UserType,
		Subject:     clientID,
		Permission:  authSvc.AdministratorRelation,
		Object:      authSvc.MagistralaObject,
		ObjectType:  authSvc.PlatformType,
	})
	if err != nil || !res.Authorized {
		addPolicyRes, err := authClient.AddPolicy(ctx, &magistrala.AddPolicyReq{
			SubjectType: authSvc.UserType,
			Subject:     clientID,
			Relation:    authSvc.AdministratorRelation,
			Object:      authSvc.MagistralaObject,
			ObjectType:  authSvc.PlatformType,
		})
		if err != nil {
			return err
		}
		if !addPolicyRes.Authorized {
			return errors.ErrAuthorization
		}
	}
	return nil
}
