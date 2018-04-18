package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	manager "github.com/mainflux/mainflux/manager/client"
	adapter "github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/api"
	"github.com/mainflux/mainflux/ws/nats"
	broker "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

const (
	defPort       = "8180"
	defNatsURL    = broker.DefaultURL
	defManagerURL = "http://localhost:8180"
	envPort       = "MF_WS_ADAPTER_PORT"
	envNatsURL    = "MF_NATS_URL"
	envManagerURL = "MF_MANAGER_URL"
)

type config struct {
	ManagerURL string
	NatsURL    string
	Port       string
}

func main() {
	cfg := config{
		ManagerURL: mainflux.Env(envManagerURL, defManagerURL),
		NatsURL:    mainflux.Env(envNatsURL, defNatsURL),
		Port:       mainflux.Env(envPort, defPort),
	}

	logger := log.New(os.Stdout)

	nc, err := broker.Connect(cfg.NatsURL)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to NATS: %s", err))
		os.Exit(1)
	}
	defer nc.Close()

	pubsub := nats.New(nc)
	svc := adapter.New(pubsub)
	svc = api.LoggingMiddleware(svc, logger)
	svc = api.MetricsMiddleware(
		svc,
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "ws_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, []string{"method"}),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "ws_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, []string{"method"}),
	)

	errs := make(chan error, 2)

	go func() {
		p := fmt.Sprintf(":%s", cfg.Port)
		mc := manager.NewClient(cfg.ManagerURL)
		logger.Info(fmt.Sprintf("WebSocket adapter service started, exposed port %s", cfg.Port))
		errs <- http.ListenAndServe(p, api.MakeHandler(svc, mc, logger))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("WebSocket adapter terminated: %s", err))
}
