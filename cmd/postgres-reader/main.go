// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains postgres-reader main function to start the postgres-reader service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authClient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	thingsClient "github.com/mainflux/mainflux/internal/clients/grpc/things"
	pgClient "github.com/mainflux/mainflux/internal/clients/postgres"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
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
	defSvcHttpPort = "9009"
)

type config struct {
	LogLevel      string `env:"MF_POSTGRES_READER_LOG_LEVEL"     envDefault:"info"`
	SendTelemetry bool   `env:"MF_SEND_TELEMETRY"                envDefault:"true"`
	InstanceID    string `env:"MF_POSTGRES_READER_INSTANCE_ID"   envDefault:""`
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

	tc, tcHandler, err := thingsClient.Setup(envPrefix)
	if err != nil {
		logger.Fatal(err.Error())
	}

	instanceID := cfg.InstanceID
	if instanceID == "" {
		instanceID, err = uuid.New().ID()
		if err != nil {
			log.Fatalf("Failed to generate instanceID: %s", err)
		}
	}

	var exitCode int
	defer mflog.ExitWithError(&exitCode)
	defer tcHandler.Close()
	logger.Info("Successfully connected to things grpc server " + tcHandler.Secure())

	auth, authHandler, err := authClient.Setup(envPrefix, svcName)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()
	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	dbConfig := pgClient.Config{Name: defDB}
	if err := dbConfig.LoadEnv(envPrefix); err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	db, err := pgClient.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup postgres database : %s", err))
		exitCode = 1
		return
	}
	defer db.Close()

	repo := newService(db, logger)

	httpServerConfig := server.Config{Port: defSvcHttpPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHttp, AltPrefix: envPrefix}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(repo, tc, auth, svcName, instanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

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

func newService(db *sqlx.DB, logger mflog.Logger) readers.MessageRepository {
	svc := postgres.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	counter, latency := internal.MakeMetrics("postgres", "message_reader")
	svc = api.MetricsMiddleware(svc, counter, latency)

	return svc
}
