//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/writers"
	"github.com/mainflux/mainflux/writers/api"
	"github.com/mainflux/mainflux/writers/influxdb"
	nats "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	svcName = "influxdb-writer"

	defNatsURL      = nats.DefaultURL
	defLogLevel     = "error"
	defPort         = "8180"
	defBatchSize    = "5000"
	defBatchTimeout = "5"
	defDBName       = "mainflux"
	defDBHost       = "localhost"
	defDBPort       = "8086"
	defDBUser       = "mainflux"
	defDBPass       = "mainflux"
	defChanCfgPath  = "/config/channels.toml"

	envNatsURL      = "MF_NATS_URL"
	envLogLevel     = "MF_INFLUX_WRITER_LOG_LEVEL"
	envPort         = "MF_INFLUX_WRITER_PORT"
	envBatchSize    = "MF_INFLUX_WRITER_BATCH_SIZE"
	envBatchTimeout = "MF_INFLUX_WRITER_BATCH_TIMEOUT"
	envDBName       = "MF_INFLUX_WRITER_DB_NAME"
	envDBHost       = "MF_INFLUX_WRITER_DB_HOST"
	envDBPort       = "MF_INFLUX_WRITER_DB_PORT"
	envDBUser       = "MF_INFLUX_WRITER_DB_USER"
	envDBPass       = "MF_INFLUX_WRITER_DB_PASS"
	envChanCfgPath  = "MF_INFLUX_WRITER_CHANNELS_CONFIG"
)

type config struct {
	natsURL      string
	logLevel     string
	port         string
	batchSize    string
	batchTimeout string
	dbName       string
	dbHost       string
	dbPort       string
	dbUser       string
	dbPass       string
	channels     map[string]bool
}

func main() {
	cfg, clientCfg := loadConfigs()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	nc, err := nats.Connect(cfg.natsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nc.Close()

	client, err := influxdata.NewHTTPClient(clientCfg)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create InfluxDB client: %s", err))
		os.Exit(1)
	}
	defer client.Close()

	batchTimeout, err := strconv.Atoi(cfg.batchTimeout)
	if err != nil {
		logger.Error(fmt.Sprintf("Invalid value for batch timeout: %s", err))
		os.Exit(1)
	}

	batchSize, err := strconv.Atoi(cfg.batchSize)
	if err != nil {
		logger.Error(fmt.Sprintf("Invalid value of batch size: %s", err))
		os.Exit(1)
	}

	timeout := time.Duration(batchTimeout) * time.Second
	repo, err := influxdb.New(client, cfg.dbName, batchSize, timeout)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create InfluxDB writer: %s", err))
		os.Exit(1)
	}

	counter, latency := makeMetrics()
	repo = api.LoggingMiddleware(repo, logger)
	repo = api.MetricsMiddleware(repo, counter, latency)
	if err := writers.Start(nc, repo, svcName, cfg.channels, logger); err != nil {
		logger.Error(fmt.Sprintf("Failed to start InfluxDB writer: %s", err))
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
	logger.Error(fmt.Sprintf("InfluxDB writer service terminated: %s", err))
}

func loadConfigs() (config, influxdata.HTTPConfig) {
	chanCfgPath := mainflux.Env(envChanCfgPath, defChanCfgPath)
	cfg := config{
		natsURL:      mainflux.Env(envNatsURL, defNatsURL),
		logLevel:     mainflux.Env(envLogLevel, defLogLevel),
		port:         mainflux.Env(envPort, defPort),
		batchSize:    mainflux.Env(envBatchSize, defBatchSize),
		batchTimeout: mainflux.Env(envBatchTimeout, defBatchTimeout),
		dbName:       mainflux.Env(envDBName, defDBName),
		dbHost:       mainflux.Env(envDBHost, defDBHost),
		dbPort:       mainflux.Env(envDBPort, defDBPort),
		dbUser:       mainflux.Env(envDBUser, defDBUser),
		dbPass:       mainflux.Env(envDBPass, defDBPass),
		channels:     loadChansConfig(chanCfgPath),
	}

	clientCfg := influxdata.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%s", cfg.dbHost, cfg.dbPort),
		Username: cfg.dbUser,
		Password: cfg.dbPass,
	}

	return cfg, clientCfg
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

func startHTTPService(port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("InfluxDB writer service started, exposed port %s", p))
	errs <- http.ListenAndServe(p, api.MakeHandler(svcName))
}
