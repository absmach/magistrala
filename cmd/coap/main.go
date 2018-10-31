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

	gocoap "github.com/dustin/go-coap"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/api"
	"github.com/mainflux/mainflux/coap/nats"
	logger "github.com/mainflux/mainflux/logger"
	thingsapi "github.com/mainflux/mainflux/things/api/grpc"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	broker "github.com/nats-io/go-nats"
)

const (
	defPort      = "5683"
	defNatsURL   = broker.DefaultURL
	defThingsURL = "localhost:8181"
	defLogLevel  = "error"

	envPort      = "MF_COAP_ADAPTER_PORT"
	envNatsURL   = "MF_NATS_URL"
	envThingsURL = "MF_THINGS_URL"
	envLogLevel  = "MF_COAP_ADAPTER_LOG_LEVEL"
)

type config struct {
	port      string
	natsURL   string
	thingsURL string
	logLevel  string
}

func main() {
	cfg := loadConfig()

	logger, err := logger.New(os.Stdout, cfg.logLevel)
	if err != nil {
		log.Fatalf(err.Error())
	}

	nc, err := broker.Connect(cfg.natsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nc.Close()

	conn, err := grpc.Dial(cfg.thingsURL, grpc.WithInsecure())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to users service: %s", err))
		os.Exit(1)
	}
	defer conn.Close()

	cc := thingsapi.NewClient(conn)
	respChan := make(chan string, 10000)
	pubsub := nats.New(nc)
	svc := coap.New(pubsub, respChan)
	svc = api.LoggingMiddleware(svc, logger)

	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "coap_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "coap_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	errs := make(chan error, 2)

	go startHTTPServer(cfg.port, logger, errs)
	go startCOAPServer(cfg.port, svc, cc, respChan, logger, errs)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("CoAP adapter terminated: %s", err))
}

func loadConfig() config {
	return config{
		thingsURL: mainflux.Env(envThingsURL, defThingsURL),
		natsURL:   mainflux.Env(envNatsURL, defNatsURL),
		port:      mainflux.Env(envPort, defPort),
		logLevel:  mainflux.Env(envLogLevel, defLogLevel),
	}
}

func startHTTPServer(port string, logger logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	logger.Info(fmt.Sprintf("CoAP service started, exposed port %s", port))
	errs <- http.ListenAndServe(p, api.MakeHTTPHandler())
}

func startCOAPServer(port string, svc coap.Service, auth mainflux.ThingsServiceClient, respChan chan<- string, l logger.Logger, errs chan error) {
	p := fmt.Sprintf(":%s", port)
	l.Info(fmt.Sprintf("CoAP adapter service started, exposed port %s", port))
	errs <- gocoap.ListenAndServe("udp", p, api.MakeCOAPHandler(svc, auth, l, respChan))
}
