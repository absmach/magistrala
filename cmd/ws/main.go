package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	clientsapi "github.com/mainflux/mainflux/clients/api/grpc"
	log "github.com/mainflux/mainflux/logger"
	adapter "github.com/mainflux/mainflux/ws"
	"github.com/mainflux/mainflux/ws/api"
	"github.com/mainflux/mainflux/ws/nats"
	broker "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	defPort       = "8180"
	defNatsURL    = broker.DefaultURL
	defClientsURL = "localhost:8181"
	envPort       = "MF_WS_ADAPTER_PORT"
	envNatsURL    = "MF_NATS_URL"
	envClientsURL = "MF_CLIENTS_URL"
)

type config struct {
	ClientsURL string
	NatsURL    string
	Port       string
}

func main() {
	cfg := config{
		ClientsURL: mainflux.Env(envClientsURL, defClientsURL),
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

	conn, err := grpc.Dial(cfg.ClientsURL, grpc.WithInsecure())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to users service: %s", err))
		os.Exit(1)
	}
	defer conn.Close()

	cc := clientsapi.NewClient(conn)
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
		logger.Info(fmt.Sprintf("WebSocket adapter service started, exposed port %s", cfg.Port))
		errs <- http.ListenAndServe(p, api.MakeHandler(svc, cc, logger))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("WebSocket adapter terminated: %s", err))
}
