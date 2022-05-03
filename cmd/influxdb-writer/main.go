// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/influxdb"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "influxdb-writer"
	stopWaitTime = 5 * time.Second

	defNatsURL    = "nats://localhost:4222"
	defLogLevel   = "error"
	defPort       = "8180"
	defDB         = "mainflux"
	defDBHost     = "localhost"
	defDBPort     = "8086"
	defDBUser     = "mainflux"
	defDBPass     = "mainflux"
	defConfigPath = "/config.toml"

	envNatsURL    = "MF_NATS_URL"
	envLogLevel   = "MF_INFLUX_WRITER_LOG_LEVEL"
	envPort       = "MF_INFLUX_WRITER_PORT"
	envDB         = "MF_INFLUXDB_DB"
	envDBHost     = "MF_INFLUXDB_HOST"
	envDBPort     = "MF_INFLUXDB_PORT"
	envDBUser     = "MF_INFLUXDB_ADMIN_USER"
	envDBPass     = "MF_INFLUXDB_ADMIN_PASSWORD"
	envConfigPath = "MF_INFLUX_WRITER_CONFIG_PATH"
)

type config struct {
	natsURL    string
	logLevel   string
	port       string
	dbName     string
	dbHost     string
	dbPort     string
	dbUser     string
	dbPass     string
	configPath string
}

func main() {
	cfg, clientCfg := loadConfigs()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	pubSub, err := nats.NewPubSub(cfg.natsURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	client, err := influxdata.NewHTTPClient(clientCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create InfluxDB client: %s", err))
		os.Exit(1)
	}
	defer client.Close()

	repo := influxdb.New(client, cfg.dbName)

	counter, latency := makeMetrics()
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(repo, counter, latency)

	if err := consumers.Start(pubSub, repo, cfg.configPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to start InfluxDB writer: %s", err))
		os.Exit(1)
	}

	g.Go(func() error {
		return startHTTPService(ctx, cfg.port, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("InfluxDB reader service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("InfluxDB reader service terminated: %s", err))
	}
}

func loadConfigs() (config, influxdata.HTTPConfig) {
	cfg := config{
		natsURL:    mainflux.Env(envNatsURL, defNatsURL),
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		port:       mainflux.Env(envPort, defPort),
		dbName:     mainflux.Env(envDB, defDB),
		dbHost:     mainflux.Env(envDBHost, defDBHost),
		dbPort:     mainflux.Env(envDBPort, defDBPort),
		dbUser:     mainflux.Env(envDBUser, defDBUser),
		dbPass:     mainflux.Env(envDBPass, defDBPass),
		configPath: mainflux.Env(envConfigPath, defConfigPath),
	}

	clientCfg := influxdata.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%s", cfg.dbHost, cfg.dbPort),
		Username: cfg.dbUser,
		Password: cfg.dbPass,
	}

	return cfg, clientCfg
}

func makeMetrics() (*kitprometheus.Counter, *kitprometheus.Summary) {
	counter := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "influxdb",
		Subsystem: "message_writer",
		Name:      "request_count",
		Help:      "Number of database inserts.",
	}, []string{"method"})

	latency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "influxdb",
		Subsystem: "message_writer",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of inserts in microseconds.",
	}, []string{"method"})

	return counter, latency
}

func startHTTPService(ctx context.Context, port string, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(svcName)}

	logger.Info(fmt.Sprintf("InfluxDB writer service started, exposed port %s", p))

	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("InfluxDB writer service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("influxDB writer service occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("InfluxDB writer service  shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
