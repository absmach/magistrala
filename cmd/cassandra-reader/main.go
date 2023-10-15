// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains cassandra-reader main function to start the cassandra-reader service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gocql/gocql"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	cassandraclient "github.com/mainflux/mainflux/internal/clients/cassandra"
	authclient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/cassandra"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "cassandra-reader"
	envPrefixDB    = "MF_CASSANDRA_"
	envPrefixHTTP  = "MF_CASSANDRA_READER_HTTP_"
	defSvcHTTPPort = "9003"
)

type config struct {
	LogLevel      string `env:"MF_CASSANDRA_READER_LOG_LEVEL"     envDefault:"info"`
	SendTelemetry bool   `env:"MF_SEND_TELEMETRY"                 envDefault:"true"`
	InstanceID    string `env:"MF_CASSANDRA_READER_INSTANCE_ID"   envDefault:""`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create cassandra reader service configurations
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s service configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
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

	// Create new auth grpc client
	auth, authHandler, err := authclient.Setup(svcName)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()

	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	// Create new cassandra client
	csdSession, err := cassandraclient.Setup(envPrefixDB)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer csdSession.Close()

	// Create new service
	repo := newService(csdSession, logger)

	// Create new http server
	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	hs := httpserver.New(ctx, cancel, svcName, httpServerConfig, api.MakeHandler(repo, auth, svcName, cfg.InstanceID), logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	// Start servers
	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Cassandra reader service terminated: %s", err))
	}
}

func newService(csdSession *gocql.Session, logger mflog.Logger) readers.MessageRepository {
	repo := cassandra.New(csdSession)
	repo = api.LoggingMiddleware(repo, logger)
	counter, latency := internal.MakeMetrics("cassandra", "message_reader")
	repo = api.MetricsMiddleware(repo, counter, latency)
	return repo
}
