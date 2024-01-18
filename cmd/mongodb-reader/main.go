// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains mongodb-reader main function to start the mongodb-reader service.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	chclient "github.com/absmach/callhome/pkg/client"
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal"
	mongoclient "github.com/absmach/magistrala/internal/clients/mongo"
	"github.com/absmach/magistrala/internal/server"
	httpserver "github.com/absmach/magistrala/internal/server/http"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/readers"
	"github.com/absmach/magistrala/readers/api"
	"github.com/absmach/magistrala/readers/mongodb"
	"github.com/caarlos0/env/v10"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "mongodb-reader"
	envPrefixDB    = "MG_MONGO_"
	envPrefixHTTP  = "MG_MONGO_READER_HTTP_"
	envPrefixAuth  = "MG_AUTH_GRPC_"
	envPrefixAuthz = "MG_THINGS_AUTH_GRPC_"
	defSvcHTTPPort = "9007"
)

type config struct {
	LogLevel      string `env:"MG_MONGO_READER_LOG_LEVEL"     envDefault:"info"`
	SendTelemetry bool   `env:"MG_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID    string `env:"MG_MONGO_READER_INSTANCE_ID"   envDefault:""`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

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

	db, err := mongoclient.Setup(envPrefixDB)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to setup mongo database : %s", err))
		exitCode = 1
		return
	}

	repo := newService(db, logger)

	authConfig := auth.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuth}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	ac, acHandler, err := auth.Setup(authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer acHandler.Close()

	logger.Info("Successfully connected to auth grpc server " + acHandler.Secure())

	authConfig = auth.Config{}
	if err := env.ParseWithOptions(&authConfig, env.Options{Prefix: envPrefixAuthz}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s auth configuration : %s", svcName, err))
		exitCode = 1
		return
	}

	tc, tcHandler, err := auth.SetupAuthz(authConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer tcHandler.Close()

	logger.Info("Successfully connected to things grpc server " + tcHandler.Secure())

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.ParseWithOptions(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(repo, ac, tc, svcName, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, magistrala.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("MongoDB reader service terminated: %s", err))
	}
}

func newService(db *mongo.Database, logger *slog.Logger) readers.MessageRepository {
	repo := mongodb.New(db)
	repo = api.LoggingMiddleware(repo, logger)
	counter, latency := internal.MakeMetrics("mongodb", "message_reader")
	repo = api.MetricsMiddleware(repo, counter, latency)

	return repo
}
