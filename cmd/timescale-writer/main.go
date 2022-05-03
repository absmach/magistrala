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
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/timescale"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "timescaledb-writer"
	stopWaitTime = 5 * time.Second

	defLogLevel      = "error"
	defNatsURL       = "nats://localhost:4222"
	defPort          = "8180"
	defDBHost        = "localhost"
	defDBPort        = "5432"
	defDBUser        = "mainflux"
	defDBPass        = "mainflux"
	defDB            = "mainflux"
	defDBSSLMode     = "disable"
	defDBSSLCert     = ""
	defDBSSLKey      = ""
	defDBSSLRootCert = ""
	defConfigPath    = "/config.toml"

	envNatsURL       = "MF_NATS_URL"
	envLogLevel      = "MF_TIMESCALE_WRITER_LOG_LEVEL"
	envPort          = "MF_TIMESCALE_WRITER_PORT"
	envDBHost        = "MF_TIMESCALE_WRITER_DB_HOST"
	envDBPort        = "MF_TIMESCALE_WRITER_DB_PORT"
	envDBUser        = "MF_TIMESCALE_WRITER_DB_USER"
	envDBPass        = "MF_TIMESCALE_WRITER_DB_PASS"
	envDB            = "MF_TIMESCALE_WRITER_DB"
	envDBSSLMode     = "MF_TIMESCALE_WRITER_DB_SSL_MODE"
	envDBSSLCert     = "MF_TIMESCALE_WRITER_DB_SSL_CERT"
	envDBSSLKey      = "MF_TIMESCALE_WRITER_DB_SSL_KEY"
	envDBSSLRootCert = "MF_TIMESCALE_WRITER_DB_SSL_ROOT_CERT"
	envConfigPath    = "MF_TIMESCALE_WRITER_CONFIG_PATH"
)

type config struct {
	natsURL    string
	logLevel   string
	port       string
	configPath string
	dbConfig   timescale.Config
}

func main() {
	cfg := loadConfig()
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

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	repo := newService(db, logger)

	if err = consumers.Start(pubSub, repo, cfg.configPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Timescale writer: %s", err))
	}

	g.Go(func() error {
		return startHTTPServer(ctx, cfg.port, logger)
	})

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Timescale writer service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Timescale writer service terminated: %s", err))
	}
}

func loadConfig() config {
	dbConfig := timescale.Config{
		Host:        mainflux.Env(envDBHost, defDBHost),
		Port:        mainflux.Env(envDBPort, defDBPort),
		User:        mainflux.Env(envDBUser, defDBUser),
		Pass:        mainflux.Env(envDBPass, defDBPass),
		Name:        mainflux.Env(envDB, defDB),
		SSLMode:     mainflux.Env(envDBSSLMode, defDBSSLMode),
		SSLCert:     mainflux.Env(envDBSSLCert, defDBSSLCert),
		SSLKey:      mainflux.Env(envDBSSLKey, defDBSSLKey),
		SSLRootCert: mainflux.Env(envDBSSLRootCert, defDBSSLRootCert),
	}

	return config{
		natsURL:    mainflux.Env(envNatsURL, defNatsURL),
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		port:       mainflux.Env(envPort, defPort),
		configPath: mainflux.Env(envConfigPath, defConfigPath),
		dbConfig:   dbConfig,
	}
}

func connectToDB(dbConfig timescale.Config, logger logger.Logger) *sqlx.DB {
	db, err := timescale.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(db *sqlx.DB, logger logger.Logger) consumers.Consumer {
	svc := timescale.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "timescale",
			Subsystem: "message_writer",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "timescale",
			Subsystem: "message_writer",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
}

func startHTTPServer(ctx context.Context, port string, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(svcName)}

	logger.Info(fmt.Sprintf("Timescale writer service started, exposed port %s", port))
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("timescale writer service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("timescale writer service occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("Timescale writer service  shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}

}
