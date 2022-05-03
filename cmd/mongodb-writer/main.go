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
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/mongodb"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "mongodb-writer"
	stopWaitTime = 5 * time.Second

	defLogLevel   = "error"
	defNatsURL    = "nats://localhost:4222"
	defPort       = "8180"
	defDB         = "mainflux"
	defDBHost     = "localhost"
	defDBPort     = "27017"
	defConfigPath = "/config.toml"

	envNatsURL    = "MF_NATS_URL"
	envLogLevel   = "MF_MONGO_WRITER_LOG_LEVEL"
	envPort       = "MF_MONGO_WRITER_PORT"
	envDB         = "MF_MONGO_WRITER_DB"
	envDBHost     = "MF_MONGO_WRITER_DB_HOST"
	envDBPort     = "MF_MONGO_WRITER_DB_PORT"
	envConfigPath = "MF_MONGO_WRITER_CONFIG_PATH"
)

type config struct {
	natsURL    string
	logLevel   string
	port       string
	dbName     string
	dbHost     string
	dbPort     string
	configPath string
}

func main() {
	cfg := loadConfigs()
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	pubSub, err := nats.NewPubSub(cfg.natsURL, "", logger)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer pubSub.Close()

	addr := fmt.Sprintf("mongodb://%s:%s", cfg.dbHost, cfg.dbPort)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to database: %s", err))
		os.Exit(1)
	}

	db := client.Database(cfg.dbName)
	repo := mongodb.New(db)

	counter, latency := makeMetrics()
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(repo, counter, latency)

	if err := consumers.Start(pubSub, repo, cfg.configPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to start MongoDB writer: %s", err))
		os.Exit(1)
	}

	g.Go(func() error {
		return startHTTPService(ctx, cfg.port, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("MongoDB reader service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("MongoDB writer service terminated: %s", err))
	}

}

func loadConfigs() config {
	return config{
		natsURL:    mainflux.Env(envNatsURL, defNatsURL),
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		port:       mainflux.Env(envPort, defPort),
		dbName:     mainflux.Env(envDB, defDB),
		dbHost:     mainflux.Env(envDBHost, defDBHost),
		dbPort:     mainflux.Env(envDBPort, defDBPort),
		configPath: mainflux.Env(envConfigPath, defConfigPath),
	}
}

func makeMetrics() (*kitprometheus.Counter, *kitprometheus.Summary) {
	counter := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "mongodb",
		Subsystem: "message_writer",
		Name:      "request_count",
		Help:      "Number of database inserts.",
	}, []string{"method"})

	latency := kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Namespace: "mongodb",
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

	logger.Info(fmt.Sprintf("MongoDB writer service started, exposed port %s", p))

	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("MongoDB writer service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("mongodb writer service occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("MongoDB writer service  shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}

}
