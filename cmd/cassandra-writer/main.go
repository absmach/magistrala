// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/cassandra"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

const (
	svcName      = "cassandra-writer"
	sep          = ","
	stopWaitTime = 5 * time.Second

	defNatsURL    = "nats://localhost:4222"
	defLogLevel   = "error"
	defPort       = "8180"
	defCluster    = "127.0.0.1"
	defKeyspace   = "mainflux"
	defDBUser     = "mainflux"
	defDBPass     = "mainflux"
	defDBPort     = "9042"
	defConfigPath = "/config.toml"

	envNatsURL    = "MF_NATS_URL"
	envLogLevel   = "MF_CASSANDRA_WRITER_LOG_LEVEL"
	envPort       = "MF_CASSANDRA_WRITER_PORT"
	envCluster    = "MF_CASSANDRA_WRITER_DB_CLUSTER"
	envKeyspace   = "MF_CASSANDRA_WRITER_DB_KEYSPACE"
	envDBUser     = "MF_CASSANDRA_WRITER_DB_USER"
	envDBPass     = "MF_CASSANDRA_WRITER_DB_PASS"
	envDBPort     = "MF_CASSANDRA_WRITER_DB_PORT"
	envConfigPath = "MF_CASSANDRA_WRITER_CONFIG_PATH"
)

type config struct {
	natsURL    string
	logLevel   string
	port       string
	configPath string
	dbCfg      cassandra.DBConfig
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

	session := connectToCassandra(cfg.dbCfg, logger)
	defer session.Close()

	repo := newService(session, logger)

	if err := consumers.Start(svcName, pubSub, repo, cfg.configPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Cassandra writer: %s", err))
	}

	go startHTTPServer(ctx, cfg.port, logger)

	g.Go(func() error {
		if sig := errors.SignalHandler(ctx); sig != nil {
			cancel()
			logger.Info(fmt.Sprintf("Cassandra writer service shutdown by signal: %s", sig))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("Cassandra writer service terminated: %s", err))
	}
}

func loadConfig() config {
	dbPort, err := strconv.Atoi(mainflux.Env(envDBPort, defDBPort))
	if err != nil {
		log.Fatal(err)
	}

	dbCfg := cassandra.DBConfig{
		Hosts:    strings.Split(mainflux.Env(envCluster, defCluster), sep),
		Keyspace: mainflux.Env(envKeyspace, defKeyspace),
		User:     mainflux.Env(envDBUser, defDBUser),
		Pass:     mainflux.Env(envDBPass, defDBPass),
		Port:     dbPort,
	}

	return config{
		natsURL:    mainflux.Env(envNatsURL, defNatsURL),
		logLevel:   mainflux.Env(envLogLevel, defLogLevel),
		port:       mainflux.Env(envPort, defPort),
		configPath: mainflux.Env(envConfigPath, defConfigPath),
		dbCfg:      dbCfg,
	}
}

func connectToCassandra(dbCfg cassandra.DBConfig, logger logger.Logger) *gocql.Session {
	session, err := cassandra.Connect(dbCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Cassandra cluster: %s", err))
		os.Exit(1)
	}

	return session
}

func newService(session *gocql.Session, logger logger.Logger) consumers.Consumer {
	repo := cassandra.New(session)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "cassandra",
			Subsystem: "message_writer",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "cassandra",
			Subsystem: "message_writer",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}

func startHTTPServer(ctx context.Context, port string, logger logger.Logger) error {
	p := fmt.Sprintf(":%s", port)
	errCh := make(chan error)
	server := &http.Server{Addr: p, Handler: api.MakeHandler(svcName)}
	logger.Info(fmt.Sprintf("Cassandra writer service started, exposed port %s", port))
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), stopWaitTime)
		defer cancelShutdown()
		if err := server.Shutdown(ctxShutdown); err != nil {
			logger.Error(fmt.Sprintf("Cassandra writer  service error occurred during shutdown at %s: %s", p, err))
			return fmt.Errorf("cassandra writer service error occurred during shutdown at %s: %w", p, err)
		}
		logger.Info(fmt.Sprintf("Cassandra writer service shutdown of http at %s", p))
		return nil
	case err := <-errCh:
		return err
	}
}
