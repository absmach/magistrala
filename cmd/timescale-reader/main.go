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
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/timescale"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "timescaledb-reader"
	envPrefix      = "MF_TIMESCALE_READER_"
	envPrefixHttp  = "MF_TIMESCALE_READER_HTTP_"
	defDB          = "messages"
	defSvcHttpPort = "8180"
)

type config struct {
	LogLevel  string `env:"MF_TIMESCALE_READER_LOG_LEVEL"   envDefault:"info"`
	JaegerURL string `env:"MF_JAEGER_URL"                   envDefault:"localhost:6831"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	dbConfig := pgClient.Config{Name: defDB}
	if err := dbConfig.LoadEnv(envPrefix); err != nil {
		logger.Fatal(err.Error())
	}
	db, err := pgClient.Connect(dbConfig)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer db.Close()

	repo := newService(db, logger)

	auth, authHandler, err := authClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	tc, tcHandler, err := thingsClient.Setup(envPrefix, cfg.JaegerURL)
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer tcHandler.Close()
	logger.Info("Successfully connected to things grpc server " + tcHandler.Secure())

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Fatal(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(repo, tc, auth, svcName, logger), logger)

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Timescale reader service terminated: %s", err))
	}
}

func newService(db *sqlx.DB, logger mflog.Logger) readers.MessageRepository {
	svc := timescale.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("timescale", "message_reader")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
