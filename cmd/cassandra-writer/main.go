//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/writers"
	"github.com/mainflux/mainflux/writers/api"
	"github.com/mainflux/mainflux/writers/cassandra"
	"github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	svcName = "cassandra-writer"
	sep     = ","

	defNatsURL  = nats.DefaultURL
	defLogLevel = "error"
	defPort     = "8180"
	defCluster  = "127.0.0.1"
	defKeyspace = "mainflux"

	envNatsURL  = "MF_NATS_URL"
	envLogLevel = "MF_CASSANDRA_WRITER_LOG_LEVEL"
	envPort     = "MF_CASSANDRA_WRITER_PORT"
	envCluster  = "MF_CASSANDRA_WRITER_DB_CLUSTER"
	envKeyspace = "MF_CASSANDRA_WRITER_DB_KEYSPACE"
)

type config struct {
	natsURL  string
	logLevel string
	port     string
	cluster  string
	keyspace string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	nc := connectToNATS(cfg.natsURL, logger)
	defer nc.Close()

	session := connectToCassandra(cfg.cluster, cfg.keyspace, logger)
	defer session.Close()

	repo := newService(session, logger)
	if err := writers.Start(nc, repo, svcName, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Cassandra writer: %s", err))
	}

	errs := make(chan error, 2)

	go startHTTPServer(cfg.port, errs, logger)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Cassandra writer service terminated: %s", err))
}

func loadConfig() config {
	return config{
		natsURL:  mainflux.Env(envNatsURL, defNatsURL),
		logLevel: mainflux.Env(envLogLevel, defLogLevel),
		port:     mainflux.Env(envPort, defPort),
		cluster:  mainflux.Env(envCluster, defCluster),
		keyspace: mainflux.Env(envKeyspace, defKeyspace),
	}
}

func connectToNATS(url string, logger logger.Logger) *nats.Conn {
	nc, err := nats.Connect(url)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}

	return nc
}

func connectToCassandra(cluster, keyspace string, logger logger.Logger) *gocql.Session {
	session, err := cassandra.Connect(strings.Split(cluster, sep), keyspace)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Cassandra cluster: %s", err))
		os.Exit(1)
	}

	return session
}

func newService(session *gocql.Session, logger logger.Logger) writers.MessageRepository {
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

func startHTTPServer(port string, errs chan error, logger logger.Logger) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Cassandra writer service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(svcName))
}
