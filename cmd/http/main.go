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

	"google.golang.org/grpc/credentials"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/http/api"
	"github.com/mainflux/mainflux/http/nats"
	"github.com/mainflux/mainflux/logger"
	thingsapi "github.com/mainflux/mainflux/things/api/grpc"
	broker "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	defCACerts   string = ""
	defPort      string = "8180"
	defLogLevel  string = "error"
	defNatsURL   string = broker.DefaultURL
	defThingsURL string = "localhost:8181"
	envCACerts   string = "MF_HTTP_ADAPTER_CA_CERTS"
	envPort      string = "MF_HTTP_ADAPTER_PORT"
	envLogLevel  string = "MF_HTTP_ADAPTER_LOG_LEVEL"
	envNatsURL   string = "MF_NATS_URL"
	envThingsURL string = "MF_THINGS_URL"
)

type config struct {
	ThingsURL string
	NatsURL   string
	LogLevel  string
	Port      string
	CACerts   string
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

	conn := connectToThingsService(cfg, logger)
	defer conn.Close()

	cc := thingsapi.NewClient(conn)
	pub := nats.NewMessagePublisher(nc)

	svc := adapter.New(pub)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "http_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "http_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	errs := make(chan error, 2)

	go func() {
		p := fmt.Sprintf(":%s", cfg.Port)
		logger.Info(fmt.Sprintf("HTTP adapter service started on port %s", cfg.Port))
		errs <- http.ListenAndServe(p, api.MakeHandler(svc, cc))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("HTTP adapter terminated: %s", err))
}

func loadConfig() config {
	return config{
		ThingsURL: mainflux.Env(envThingsURL, defThingsURL),
		NatsURL:   mainflux.Env(envNatsURL, defNatsURL),
		LogLevel:  mainflux.Env(envLogLevel, defLogLevel),
		Port:      mainflux.Env(envPort, defPort),
		CACerts:   mainflux.Env(envCACerts, defCACerts),
	}
}

func connectToThingsService(cfg config, logger logger.Logger) *grpc.ClientConn {
	var opts []grpc.DialOption
	if cfg.CACerts != "" {
		tpc, err := credentials.NewClientTLSFromFile(cfg.CACerts, "")
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to load certs: %s", err))
			os.Exit(1)
		}
		opts = append(opts, grpc.WithTransportCredentials(tpc))
	} else {
		logger.Info("gRPC communication is not encrypted")
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(cfg.ThingsURL, opts...)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to things service: %s", err))
		os.Exit(1)
	}
	return conn
}
