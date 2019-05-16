//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/writers"
	"github.com/mainflux/mainflux/writers/api"
	"github.com/mainflux/mainflux/writers/mongodb"
	nats "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	svcName = "mongodb-writer"

	defNatsURL     = nats.DefaultURL
	defLogLevel    = "error"
	defPort        = "8180"
	defDBName      = "mainflux"
	defDBHost      = "localhost"
	defDBPort      = "27017"
	defChanCfgPath = "/config/channels.toml"

	envNatsURL     = "MF_NATS_URL"
	envLogLevel    = "MF_MONGO_WRITER_LOG_LEVEL"
	envPort        = "MF_MONGO_WRITER_PORT"
	envDBName      = "MF_MONGO_WRITER_DB_NAME"
	envDBHost      = "MF_MONGO_WRITER_DB_HOST"
	envDBPort      = "MF_MONGO_WRITER_DB_PORT"
	envChanCfgPath = "MF_MONGO_WRITER_CHANNELS_CONFIG"
)

type config struct {
	natsURL  string
	logLevel string
	port     string
	dbName   string
	dbHost   string
	dbPort   string
	channels map[string]bool
}

func main() {
	cfg := loadConfigs()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(cfg.natsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nc.Close()

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
	if err := writers.Start(nc, repo, svcName, cfg.channels, logger); err != nil {
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
	chanCfgPath := mainflux.Env(envChanCfgPath, defChanCfgPath)
	return config{
		natsURL:  mainflux.Env(envNatsURL, defNatsURL),
		logLevel: mainflux.Env(envLogLevel, defLogLevel),
		port:     mainflux.Env(envPort, defPort),
		dbName:   mainflux.Env(envDBName, defDBName),
		dbHost:   mainflux.Env(envDBHost, defDBHost),
		dbPort:   mainflux.Env(envDBPort, defDBPort),
		channels: loadChansConfig(chanCfgPath),
	}
}

type channels struct {
	List []string `toml:"filter"`
}

type chanConfig struct {
	Channels channels `toml:"channels"`
}

func loadChansConfig(chanConfigPath string) map[string]bool {
	data, err := ioutil.ReadFile(chanConfigPath)
	if err != nil {
		log.Fatal(err)
	}

	var chanCfg chanConfig
	if err := toml.Unmarshal(data, &chanCfg); err != nil {
		log.Fatal(err)
	}

	chans := map[string]bool{}
	for _, ch := range chanCfg.Channels.List {
		chans[ch] = true
	}

	return chans
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

func startHTTPService(port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("Mongodb writer service started, exposed port %s", p))
	errs <- http.ListenAndServe(p, api.MakeHandler(svcName))
}
