// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/consumers/writers/api"
	"github.com/mainflux/mainflux/consumers/writers/mongodb"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging/nats"
	"github.com/mainflux/mainflux/pkg/transformers"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	svcName = "mongodb-writer"

	defLogLevel    = "error"
	defNatsURL     = "nats://localhost:4222"
	defPort        = "8180"
	defDB          = "mainflux"
	defDBHost      = "localhost"
	defDBPort      = "27017"
	defConfigPath  = "/config.toml"
	defContentType = "application/senml+json"
	defTransformer = "senml"

	envNatsURL     = "MF_NATS_URL"
	envLogLevel    = "MF_MONGO_WRITER_LOG_LEVEL"
	envPort        = "MF_MONGO_WRITER_PORT"
	envDB          = "MF_MONGO_WRITER_DB"
	envDBHost      = "MF_MONGO_WRITER_DB_HOST"
	envDBPort      = "MF_MONGO_WRITER_DB_PORT"
	envConfigPath  = "MF_MONGO_WRITER_CONFIG_PATH"
	envContentType = "MF_MONGO_WRITER_CONTENT_TYPE"
	envTransformer = "MF_MONGO_WRITER_TRANSFORMER"
)

type config struct {
	natsURL     string
	logLevel    string
	port        string
	dbName      string
	dbHost      string
	dbPort      string
	configPath  string
	contentType string
	transformer string
}

func main() {
	cfg := loadConfigs()

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
	t := makeTransformer(cfg, logger)

	if err := consumers.Start(pubSub, repo, t, cfg.configPath, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to start MongoDB writer: %s", err))
		os.Exit(1)
	}

	errs := make(chan error, 2)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	go startHTTPService(cfg.port, logger, errs)

	err = <-errs
	logger.Error(fmt.Sprintf("MongoDB writer service terminated: %s", err))
}

func loadConfigs() config {
	return config{
		natsURL:     mainflux.Env(envNatsURL, defNatsURL),
		logLevel:    mainflux.Env(envLogLevel, defLogLevel),
		port:        mainflux.Env(envPort, defPort),
		dbName:      mainflux.Env(envDB, defDB),
		dbHost:      mainflux.Env(envDBHost, defDBHost),
		dbPort:      mainflux.Env(envDBPort, defDBPort),
		configPath:  mainflux.Env(envConfigPath, defConfigPath),
		contentType: mainflux.Env(envContentType, defContentType),
		transformer: mainflux.Env(envTransformer, defTransformer),
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

func startHTTPService(port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Mongodb writer service started, exposed port %s", p))
	errs <- http.ListenAndServe(p, api.MakeHandler(svcName))
}
