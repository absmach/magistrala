// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/cassandra"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	"github.com/mainflux/mainflux/pkg/transformers"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	svcName = "cassandra-writer"
	sep     = ","

	defNatsURL     = "nats://localhost:4222"
	defLogLevel    = "error"
	defPort        = "8180"
	defCluster     = "127.0.0.1"
	defKeyspace    = "mainflux"
	defDBUser      = "mainflux"
	defDBPass      = "mainflux"
	defDBPort      = "9042"
	defConfigPath  = "/config.toml"
	defContentType = "application/senml+json"
	defTransformer = "senml"

	envNatsURL     = "MF_NATS_URL"
	envLogLevel    = "MF_CASSANDRA_WRITER_LOG_LEVEL"
	envPort        = "MF_CASSANDRA_WRITER_PORT"
	envCluster     = "MF_CASSANDRA_WRITER_DB_CLUSTER"
	envKeyspace    = "MF_CASSANDRA_WRITER_DB_KEYSPACE"
	envDBUser      = "MF_CASSANDRA_WRITER_DB_USER"
	envDBPass      = "MF_CASSANDRA_WRITER_DB_PASS"
	envDBPort      = "MF_CASSANDRA_WRITER_DB_PORT"
	envConfigPath  = "MF_CASSANDRA_WRITER_CONFIG_PATH"
	envContentType = "MF_CASSANDRA_WRITER_CONTENT_TYPE"
	envTransformer = "MF_CASSANDRA_WRITER_TRANSFORMER"
)

type config struct {
	natsURL     string
	logLevel    string
	port        string
	configPath  string
	contentType string
	transformer string
	dbCfg       cassandra.DBConfig
}

func main() {
	cfg := loadConfig()

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
	t := makeTransformer(cfg, logger)

	if err := consumers.Start(pubSub, repo, t, cfg.configPath, logger); err != nil {
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
		natsURL:     mainflux.Env(envNatsURL, defNatsURL),
		logLevel:    mainflux.Env(envLogLevel, defLogLevel),
		port:        mainflux.Env(envPort, defPort),
		configPath:  mainflux.Env(envConfigPath, defConfigPath),
		contentType: mainflux.Env(envContentType, defContentType),
		transformer: mainflux.Env(envTransformer, defTransformer),
		dbCfg:       dbCfg,
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

func makeTransformer(cfg config, logger logger.Logger) transformers.Transformer {
	switch strings.ToUpper(cfg.transformer) {
	case "SENML":
		logger.Info("Using SenML transformer")
		return senml.New(cfg.contentType)
	case "JSON":
		logger.Info("Using JSON transformer")
		return json.New()
	default:
		logger.Error(fmt.Sprintf("Can't create transformer: unknown transformer type %s", cfg.transformer))
		os.Exit(1)
		return nil
	}
}

func startHTTPServer(port string, errs chan error, logger logger.Logger) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Cassandra writer service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(svcName))
}
