// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains influxdb-reader main function to start the influxdb-reader service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authclient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	influxdbclient "github.com/mainflux/mainflux/internal/clients/influxdb"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/server"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/influxdb"
	"golang.org/x/sync/errgroup"
)

const (
	svcName        = "influxdb-reader"
	envPrefixHTTP  = "MF_INFLUX_READER_HTTP_"
	envPrefixDB    = "MF_INFLUXDB_"
	defSvcHTTPPort = "9005"
)

type config struct {
	LogLevel      string `env:"MF_INFLUX_READER_LOG_LEVEL"  envDefault:"info"`
	SendTelemetry bool   `env:"MF_SEND_TELEMETRY"           envDefault:"true"`
	InstanceID    string `env:"MF_INFLUX_READER_INSTANCE_ID"   envDefault:""`
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

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	auth, authHandler, err := authclient.Setup(svcName)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer authHandler.Close()

	logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())

	influxDBConfig := influxdbclient.Config{}
	if err := env.Parse(&influxDBConfig, env.Options{Prefix: envPrefixDB}); err != nil {
		logger.Error(fmt.Sprintf("failed to load InfluxDB client configuration from environment variable : %s", err))
		exitCode = 1
		return
	}
	influxDBConfig.DBUrl = fmt.Sprintf("%s://%s:%s", influxDBConfig.Protocol, influxDBConfig.Host, influxDBConfig.Port)

	repocfg := influxdb.RepoConfig{
		Bucket: influxDBConfig.Bucket,
		Org:    influxDBConfig.Org,
	}

	client, err := influxdbclient.Connect(ctx, influxDBConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to connect to InfluxDB : %s", err))
		exitCode = 1
		return
	}
	defer client.Close()

	repo := newService(client, repocfg, logger)

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

	g.Go(func() error {
		return hs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("InfluxDB reader service terminated: %s", err))
	}
}

func newService(client influxdb2.Client, repocfg influxdb.RepoConfig, logger mflog.Logger) readers.MessageRepository {
	repo := influxdb.New(client, repocfg)
	repo = api.LoggingMiddleware(repo, logger)
	counter, latency := internal.MakeMetrics("influxdb", "message_reader")
	repo = api.MetricsMiddleware(repo, counter, latency)

	return repo
}
