// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux/internal"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	thingsClient "github.com/mainflux/mainflux/internal/clients/grpc/things"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/postgres"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "postgres-reader"
	envPrefix      = "MF_POSTGRES_READER_"
	envPrefixHttp  = "MF_POSTGRES_READER_HTTP_"
	defDB          = "messages"
	defSvcHttpPort = "8180"
)

type config struct {
	LogLevel  string `env:"MF_POSTGRES_READER_LOG_LEVEL"     envDefault:"info"`
	JaegerURL string `env:"MF_JAEGER_URL"                    envDefault:"localhost:6831"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err.Error())
	}

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	tc, tcHandler, err := thingsClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer tcHandler.Close()
	logger.Info("Successfully connected to things grpc server " + tcHandler.Secure())

	auth, authHandler, err := authClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	dbConfig := pgClient.Config{Name: defDB}
	if err := dbConfig.LoadEnv(envPrefix); err != nil {
		log.Fatal(err.Error())
	}
	db, err := pgClient.Connect(dbConfig)
	if err != nil {
		log.Fatalf("failed to setup postgres database : %s", err.Error())
	}
	defer db.Close()

	repo := newService(db, logger)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		log.Fatalf("failed to load %s HTTP server configuration : %s", svcName, err.Error())
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(repo, tc, auth, svcName, logger), logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Postgres reader service terminated: %s", err))
	}
}

func newService(db *sqlx.DB, logger logger.Logger) readers.MessageRepository {
	svc := postgres.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("postgres", "message_reader")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
