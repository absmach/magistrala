// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/postgres"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	"github.com/mainflux/mainflux/pkg/transformers"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	svcName = "postgres-writer"
	sep     = ","

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
	defContentType   = "application/senml+json"
	defTransformer   = "senml"

	envNatsURL       = "MF_NATS_URL"
	envLogLevel      = "MF_POSTGRES_WRITER_LOG_LEVEL"
	envPort          = "MF_POSTGRES_WRITER_PORT"
	envDBHost        = "MF_POSTGRES_WRITER_DB_HOST"
	envDBPort        = "MF_POSTGRES_WRITER_DB_PORT"
	envDBUser        = "MF_POSTGRES_WRITER_DB_USER"
	envDBPass        = "MF_POSTGRES_WRITER_DB_PASS"
	envDB            = "MF_POSTGRES_WRITER_DB"
	envDBSSLMode     = "MF_POSTGRES_WRITER_DB_SSL_MODE"
	envDBSSLCert     = "MF_POSTGRES_WRITER_DB_SSL_CERT"
	envDBSSLKey      = "MF_POSTGRES_WRITER_DB_SSL_KEY"
	envDBSSLRootCert = "MF_POSTGRES_WRITER_DB_SSL_ROOT_CERT"
	envConfigPath    = "MF_POSTGRES_WRITER_CONFIG_PATH"
	envContentType   = "MF_POSTGRES_WRITER_CONTENT_TYPE"
	envTransformer   = "MF_POSTGRES_WRITER_TRANSFORMER"
)

type config struct {
	natsURL     string
	logLevel    string
	port        string
	configPath  string
	contentType string
	transformer string
	dbConfig    postgres.Config
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

	db := connectToDB(cfg.dbConfig, logger)
	defer db.Close()

	repo := newService(db, logger)
	t := makeTransformer(cfg, logger)

	if err = consumers.Start(pubSub, repo, t, cfg.configPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to create Postgres writer: %s", err))
	}

	errs := make(chan error, 2)

	go startHTTPServer(cfg.port, errs, logger)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("Postgres writer service terminated: %s", err))
}

func loadConfig() config {
	dbConfig := postgres.Config{
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
		natsURL:     mainflux.Env(envNatsURL, defNatsURL),
		logLevel:    mainflux.Env(envLogLevel, defLogLevel),
		port:        mainflux.Env(envPort, defPort),
		configPath:  mainflux.Env(envConfigPath, defConfigPath),
		contentType: mainflux.Env(envContentType, defContentType),
		transformer: mainflux.Env(envTransformer, defTransformer),
		dbConfig:    dbConfig,
	}
}

func connectToDB(dbConfig postgres.Config, logger logger.Logger) *sqlx.DB {
	db, err := postgres.Connect(dbConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to Postgres: %s", err))
		os.Exit(1)
	}
	return db
}

func newService(db *sqlx.DB, logger logger.Logger) consumers.Consumer {
	svc := postgres.New(db)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "postgres",
			Subsystem: "message_writer",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "postgres",
			Subsystem: "message_writer",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	return svc
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
	logger.Info(fmt.Sprintf("Postgres writer service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHandler(svcName))
}
