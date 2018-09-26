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
	"syscall"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/normalizer"
	"github.com/mainflux/mainflux/normalizer/api"
	"github.com/mainflux/mainflux/normalizer/nats"
	broker "github.com/nats-io/go-nats"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	defNatsURL  string = broker.DefaultURL
	defLogLevel string = "error"
	defPort     string = "8180"
	envNatsURL  string = "MF_NATS_URL"
	envLogLevel string = "MF_NORMALIZER_LOG_LEVEL"
	envPort     string = "MF_NORMALIZER_PORT"
)

type config struct {
	NatsURL  string
	LogLevel string
	Port     string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}
	nc, err := broker.Connect(cfg.NatsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nc.Close()

	svc := normalizer.New()
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "normalizer",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "normalizer",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	errs := make(chan error, 2)

	go func() {
		p := fmt.Sprintf(":%s", cfg.Port)
		logger.Info(fmt.Sprintf("Normalizer service started, exposed port %s", cfg.Port))
		errs <- http.ListenAndServe(p, api.MakeHandler())
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	nats.Subscribe(svc, nc, logger)

	err = <-errs
	logger.Error(fmt.Sprintf("Normalizer service terminated: %s", err))
}

func loadConfig() config {
	return config{
		NatsURL:  mainflux.Env(envNatsURL, defNatsURL),
		LogLevel: mainflux.Env(envLogLevel, defLogLevel),
		Port:     mainflux.Env(envPort, defPort),
	}
}
