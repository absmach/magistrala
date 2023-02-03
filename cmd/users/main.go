// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/mainflux/mainflux/internal"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/internal/email"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/bcrypt"
	"github.com/mainflux/mainflux/users/emailer"
	"github.com/mainflux/mainflux/users/tracing"
	"golang.org/x/sync/errgroup"

	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	jaegerClient "github.com/mainflux/mainflux/internal/clients/jaeger"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users/api"
	usersPg "github.com/mainflux/mainflux/users/postgres"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	svcName        = "users"
	envPrefix      = "MF_USERS_"
	envPrefixHttp  = "MF_USERS_HTTP_"
	defDB          = "users"
	defSvcHttpPort = "8180"
)

type config struct {
	LogLevel      string `env:"MF_USERS_LOG_LEVEL"               envDefault:"info"`
	AdminEmail    string `env:"MF_USERS_ADMIN_EMAIL"             envDefault:""`
	AdminPassword string `env:"MF_USERS_ADMIN_PASSWORD"          envDefault:""`
	PassRegexText string `env:"MF_USERS_PASS_REGEX"              envDefault:"^.{8,}$"`
	SelfRegister  bool   `env:"MF_USERS_ALLOW_SELF_REGISTER"     envDefault:"true"`
	ResetURL      string `env:"MF_TOKEN_RESET_ENDPOINT"          envDefault:"email.tmpl"`
	JaegerURL     string `env:"MF_JAEGER_URL"                    envDefault:"localhost:6831"`
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
		log.Fatalf("Invalid password validation rules %s\n", cfg.PassRegexText)
	}
	cfg.PassRegex = passRegex

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	ec := email.Config{}
	if err := env.Parse(&ec); err != nil {
		log.Fatalf("failed to load email configuration : %s", err.Error())
	}

	dbConfig := pgClient.Config{Name: defDB}
	db, err := pgClient.SetupWithConfig(envPrefix, *usersPg.Migration(), dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	auth, authHandler, err := authClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		log.Fatal(err)
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	dbTracer, dbCloser, err := jaegerClient.NewTracer("auth_db", cfg.JaegerURL)
	if err != nil {
		log.Fatalf("failed to init Jaeger: %s", err.Error())
	}
	defer dbCloser.Close()

	svc := newService(db, dbTracer, auth, cfg, ec, logger)

	tracer, closer, err := jaegerClient.NewTracer("users", cfg.JaegerURL)
	if err != nil {
		log.Fatalf("failed to init Jaeger: %s", err.Error())
	}
	defer closer.Close()

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(svc, tracer, logger), logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Users service terminated: %s", err))
	}
}

func newService(db *sqlx.DB, tracer opentracing.Tracer, auth mainflux.AuthServiceClient, c config, ec email.Config, logger logger.Logger) users.Service {
	database := usersPg.NewDatabase(db)
	hasher := bcrypt.New()
	userRepo := tracing.UserRepositoryMiddleware(usersPg.NewUserRepo(database), tracer)

	emailer, err := emailer.New(c.ResetURL, &ec)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to configure e-mailing util: %s", err.Error()))
	}

	idProvider := uuid.New()

	svc := users.New(userRepo, hasher, auth, emailer, idProvider, c.PassRegex)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	svc = api.MetricsMiddleware(svc, counter, latency)

	if err := createAdmin(svc, userRepo, c, auth); err != nil {
		log.Fatalf("failed to create admin user: " + err.Error())
	}

	switch c.SelfRegister {
	case true:
		// If MF_USERS_ALLOW_SELF_REGISTER environment variable is "true",
		// everybody can create a new user. Here, check the existence of that
		// policy. If the policy does not exist, create it; otherwise, there is
		// no need to do anything further.
		_, err := auth.Authorize(context.Background(), &mainflux.AuthorizeReq{Obj: "user", Act: "create", Sub: "*"})
		if err != nil {
			// Add a policy that allows anybody to create a user
			apr, err := auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Obj: "user", Act: "create", Sub: "*"})
			if err != nil {
				log.Fatalf("failed to add the policy related to MF_USERS_ALLOW_SELF_REGISTER: " + err.Error())
			}
			if !apr.GetAuthorized() {
				log.Fatalf("failed to authorized the policy result related to MF_USERS_ALLOW_SELF_REGISTER: " + errors.ErrAuthorization.Error())
			}
		}
	default:
		// If MF_USERS_ALLOW_SELF_REGISTER environment variable is "false",
		// everybody cannot create a new user. Therefore, delete a policy that
		// allows everybody to create a new user.
		dpr, err := auth.DeletePolicy(context.Background(), &mainflux.DeletePolicyReq{Obj: "user", Act: "create", Sub: "*"})
		if err != nil {
			log.Fatalf("failed to delete a policy: " + err.Error())
		}
		if !dpr.GetDeleted() {
			log.Fatalf("deleting a policy expected to succeed.")
		}
	}

	return svc
}

func createAdmin(svc users.Service, userRepo users.UserRepository, c config, auth mainflux.AuthServiceClient) error {
	user := users.User{
		Email:    c.AdminEmail,
		Password: c.AdminPassword,
	}

	if admin, err := userRepo.RetrieveByEmail(context.Background(), user.Email); err == nil {
		// The admin is already created. Check existence of the admin policy.
		_, err := auth.Authorize(context.Background(), &mainflux.AuthorizeReq{Obj: "authorities", Act: "member", Sub: admin.ID})
		if err != nil {
			apr, err := auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Obj: "authorities", Act: "member", Sub: admin.ID})
			if err != nil {
				return err
			}
			if !apr.GetAuthorized() {
				return errors.ErrAuthorization
			}
		}
		return nil
	}

	// Add a policy that allows anybody to create a user
	apr, err := auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Obj: "user", Act: "create", Sub: "*"})
	if err != nil {
		return err
	}
	if !apr.GetAuthorized() {
		return errors.ErrAuthorization
	}

	// Create an admin
	uid, err := svc.Register(context.Background(), "", user)
	if err != nil {
		return err
	}

	apr, err = auth.AddPolicy(context.Background(), &mainflux.AddPolicyReq{Obj: "authorities", Act: "member", Sub: uid})
	if err != nil {
		return err
	}
	if !apr.GetAuthorized() {
		return errors.ErrAuthorization
	}

	return nil
}
