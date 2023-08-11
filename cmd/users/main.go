// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains users main function to start the users service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-zoo/bone"
	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgclient "github.com/mainflux/mainflux/internal/clients/postgres"
	redisclient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/email"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/postgres"
	"github.com/mainflux/mainflux/internal/server"
	grpcserver "github.com/mainflux/mainflux/internal/server/grpc"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	gpostgres "github.com/mainflux/mainflux/pkg/groups/postgres"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users/clients"
	capi "github.com/mainflux/mainflux/users/clients/api"
	"github.com/mainflux/mainflux/users/clients/emailer"
	uclients "github.com/mainflux/mainflux/users/clients/postgres"
	ucache "github.com/mainflux/mainflux/users/clients/redis"
	ctracing "github.com/mainflux/mainflux/users/clients/tracing"
	"github.com/mainflux/mainflux/users/groups"
	gapi "github.com/mainflux/mainflux/users/groups/api"
	gcache "github.com/mainflux/mainflux/users/groups/redis"
	gtracing "github.com/mainflux/mainflux/users/groups/tracing"
	"github.com/mainflux/mainflux/users/hasher"
	"github.com/mainflux/mainflux/users/jwt"
	"github.com/mainflux/mainflux/users/policies"
	papi "github.com/mainflux/mainflux/users/policies/api"
	grpcapi "github.com/mainflux/mainflux/users/policies/api/grpc"
	httpapi "github.com/mainflux/mainflux/users/policies/api/http"
	ppostgres "github.com/mainflux/mainflux/users/policies/postgres"
	pcache "github.com/mainflux/mainflux/users/policies/redis"
	ptracing "github.com/mainflux/mainflux/users/policies/tracing"
	clientspg "github.com/mainflux/mainflux/users/postgres"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	svcName        = "users"
	envPrefixDB    = "MF_USERS_DB_"
	envPrefixES    = "MF_USERS_ES_"
	envPrefixHTTP  = "MF_USERS_HTTP_"
	envPrefixGrpc  = "MF_USERS_GRPC_"
	defDB          = "users"
	defSvcHTTPPort = "9002"
	defSvcGRPCPort = "9192"
)

type config struct {
	LogLevel        string `env:"MF_USERS_LOG_LEVEL"              envDefault:"info"`
	SecretKey       string `env:"MF_USERS_SECRET_KEY"             envDefault:"secret"`
	AdminEmail      string `env:"MF_USERS_ADMIN_EMAIL"            envDefault:""`
	AdminPassword   string `env:"MF_USERS_ADMIN_PASSWORD"         envDefault:""`
	PassRegexText   string `env:"MF_USERS_PASS_REGEX"             envDefault:"^.{8,}$"`
	AccessDuration  string `env:"MF_USERS_ACCESS_TOKEN_DURATION"  envDefault:"15m"`
	RefreshDuration string `env:"MF_USERS_REFRESH_TOKEN_DURATION" envDefault:"24h"`
	ResetURL        string `env:"MF_TOKEN_RESET_ENDPOINT"         envDefault:"/reset-request"`
	JaegerURL       string `env:"MF_JAEGER_URL"                   envDefault:"http://jaeger:14268/api/traces"`
	SendTelemetry   bool   `env:"MF_SEND_TELEMETRY"               envDefault:"true"`
	InstanceID      string `env:"MF_USERS_INSTANCE_ID"            envDefault:""`
	PassRegex       *regexp.Regexp
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

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to init logger: %s", err.Error()))
	}

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

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
	if err := dbConfig.LoadEnv(envPrefixDB); err != nil {
		logger.Fatal(err.Error())
	}
	db, err := pgclient.SetupWithConfig(envPrefixDB, *clientspg.Migration(), dbConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaegerclient.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID)
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

	// Setup new redis event store client
	esClient, err := redisclient.Setup(envPrefixES)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer esClient.Close()

	csvc, gsvc, psvc := newService(ctx, db, dbConfig, esClient, tracer, cfg, ec, logger)

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	mux := bone.New()
	hsc := httpserver.New(ctx, cancel, svcName, httpServerConfig, capi.MakeHandler(csvc, mux, logger, cfg.InstanceID), logger)
	hsg := httpserver.New(ctx, cancel, svcName, httpServerConfig, gapi.MakeHandler(gsvc, mux, logger), logger)
	hsp := httpserver.New(ctx, cancel, svcName, httpServerConfig, httpapi.MakeHandler(psvc, mux, logger), logger)

	grpcServerConfig := server.Config{Port: defSvcGRPCPort}
	if err := env.Parse(&grpcServerConfig, env.Options{Prefix: envPrefixGrpc}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err.Error()))
		exitCode = 1
		return
	}
	registerAuthServiceServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		policies.RegisterAuthServiceServer(srv, grpcapi.NewServer(csvc, psvc))
	}
	gs := grpcserver.New(ctx, cancel, svcName, grpcServerConfig, registerAuthServiceServer, logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hsp.Start()
	})
	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hsc, hsg, hsp, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("users service terminated: %s", err))
	}
}

func newService(ctx context.Context, db *sqlx.DB, dbConfig pgclient.Config, esClient *redis.Client, tracer trace.Tracer, c config, ec email.Config, logger mflog.Logger) (clients.Service, groups.Service, policies.Service) {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	cRepo := uclients.NewRepository(database)
	gRepo := gpostgres.New(database)
	pRepo := ppostgres.NewRepository(database)

	idp := uuid.New()
	hsr := hasher.New()

	aDuration, err := time.ParseDuration(c.AccessDuration)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse access token duration: %s", err.Error()))
	}
	rDuration, err := time.ParseDuration(c.RefreshDuration)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse refresh token duration: %s", err.Error()))
	}
	tokenizer := jwt.NewRepository([]byte(c.SecretKey), aDuration, rDuration)

	emailer, err := emailer.New(c.ResetURL, &ec)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to configure e-mailing util: %s", err.Error()))
	}
	csvc := clients.NewService(cRepo, pRepo, tokenizer, emailer, hsr, idp, c.PassRegex)
	gsvc := groups.NewService(gRepo, pRepo, tokenizer, idp)
	psvc := policies.NewService(pRepo, tokenizer, idp)

	csvc = ucache.NewEventStoreMiddleware(ctx, csvc, esClient)
	gsvc = gcache.NewEventStoreMiddleware(ctx, gsvc, esClient)
	psvc = pcache.NewEventStoreMiddleware(ctx, psvc, esClient)

	csvc = ctracing.New(csvc, tracer)
	csvc = capi.LoggingMiddleware(csvc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	csvc = capi.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = internal.MakeMetrics("groups", "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)

	psvc = ptracing.New(psvc, tracer)
	psvc = papi.LoggingMiddleware(psvc, logger)
	counter, latency = internal.MakeMetrics("policies", "api")
	psvc = papi.MetricsMiddleware(psvc, counter, latency)

	if err := createAdmin(ctx, c, cRepo, hsr, csvc); err != nil {
		logger.Error(fmt.Sprintf("failed to create admin client: %s", err))
	}
	return csvc, gsvc, psvc
}

func createAdmin(ctx context.Context, c config, crepo uclients.Repository, hsr clients.Hasher, svc clients.Service) error {
	id, err := uuid.New().ID()
	if err != nil {
		return err
	}
	hash, err := hsr.Hash(c.AdminPassword)
	if err != nil {
		return err
	}

	client := mfclients.Client{
		ID:   id,
		Name: "admin",
		Credentials: mfclients.Credentials{
			Identity: c.AdminEmail,
			Secret:   hash,
		},
		Metadata: mfclients.Metadata{
			"role": "admin",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Role:      mfclients.AdminRole,
		Status:    mfclients.EnabledStatus,
	}

	if _, err := crepo.RetrieveByIdentity(ctx, client.Credentials.Identity); err == nil {
		return nil
	}

	// Create an admin
	if _, err = crepo.Save(ctx, client); err != nil {
		return err
	}
	if _, err = svc.IssueToken(ctx, c.AdminEmail, c.AdminPassword); err != nil {
		return err
	}

	return nil
}
