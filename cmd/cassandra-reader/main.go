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
	"strconv"
	"strings"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
	"github.com/mainflux/mainflux/readers/api"
	"github.com/mainflux/mainflux/readers/cassandra"
	thingsapi "github.com/mainflux/mainflux/things/api/grpc"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	sep = ","

	defLogLevel  = "error"
	defPort      = "8180"
	defCluster   = "127.0.0.1"
	defKeyspace  = "mainflux"
	defThingsURL = "localhost:8181"
	defClientTLS = "false"
	defCACerts   = ""

	envLogLevel  = "MF_CASSANDRA_READER_LOG_LEVEL"
	envPort      = "MF_CASSANDRA_READER_PORT"
	envCluster   = "MF_CASSANDRA_READER_DB_CLUSTER"
	envKeyspace  = "MF_CASSANDRA_READER_DB_KEYSPACE"
	envThingsURL = "MF_THINGS_URL"
	envClientTLS = "MF_CASSANDRA_READER_CLIENT_TLS"
	envCACerts   = "MF_CASSANDRA_READER_CA_CERTS"
)

type config struct {
	logLevel  string
	port      string
	cluster   string
	keyspace  string
	thingsURL string
	clientTLS bool
	caCerts   string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	session := connectToCassandra(cfg.cluster, cfg.keyspace, logger)
	defer session.Close()

	conn := connectToThings(cfg, logger)
	defer conn.Close()

	tc := thingsapi.NewClient(conn)
	repo := newService(session, logger)

	errs := make(chan error, 2)

	go startHTTPServer(repo, tc, cfg.port, errs, logger)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Cassandra reader service terminated: %s", err))
}

func loadConfig() config {
	tls, err := strconv.ParseBool(mainflux.Env(envClientTLS, defClientTLS))
	if err != nil {
		log.Fatalf("Invalid value passed for %s\n", envClientTLS)
	}

	return config{
		logLevel:  mainflux.Env(envLogLevel, defLogLevel),
		port:      mainflux.Env(envPort, defPort),
		cluster:   mainflux.Env(envCluster, defCluster),
		keyspace:  mainflux.Env(envKeyspace, defKeyspace),
		thingsURL: mainflux.Env(envThingsURL, defThingsURL),
		clientTLS: tls,
		caCerts:   mainflux.Env(envCACerts, defCACerts),
	}
}

func connectToCassandra(cluster, keyspace string, logger logger.Logger) *gocql.Session {
	session, err := cassandra.Connect(strings.Split(cluster, sep), keyspace)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Cassandra cluster: %s", err))
		os.Exit(1)
	}

	return session
}

func connectToThings(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.clientTLS {
		if cfg.caCerts != "" {
			tpc, err := credentials.NewClientTLSFromFile(cfg.caCerts, "")
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to load certs: %s", err))
				os.Exit(1)
			}
			opts = append(opts, grpc.WithTransportCredentials(tpc))
		}
	} else {
		logger.Info("gRPC communication is not encrypted")
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(cfg.thingsURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
}

func newService(session *gocql.Session, logger logger.Logger) readers.MessageRepository {
	repo := cassandra.New(session)
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(
		repo,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "cassandra",
			Subsystem: "message_reader",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "cassandra",
			Subsystem: "message_reader",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return repo
}

func startHTTPServer(repo readers.MessageRepository, tc mainflux.ThingsServiceClient, port string, errs chan error, logger logger.Logger) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Cassandra reader service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(repo, tc, "cassandra-reader"))
}
