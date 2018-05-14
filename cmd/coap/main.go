package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	"github.com/mainflux/mainflux"
	clientsapi "github.com/mainflux/mainflux/clients/api/grpc"
	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/api"
	"github.com/mainflux/mainflux/coap/nats"
	log "github.com/mainflux/mainflux/logger"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"

	broker "github.com/nats-io/go-nats"
)

const (
	defPort       int    = 5683
	defNatsURL    string = broker.DefaultURL
	defClientsURL string = "localhost:8181"
	envPort       string = "MF_COAP_ADAPTER_PORT"
	envNatsURL    string = "MF_NATS_URL"
	envClientsURL string = "MF_CLIENTS_URL"
)

type config struct {
	ClientsURL string
	NatsURL    string
	Port       int
}

func main() {
	cfg := config{
		ClientsURL: mainflux.Env(envClientsURL, defClientsURL),
		NatsURL:    mainflux.Env(envNatsURL, defNatsURL),
		Port:       defPort,
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

	pubsub := nats.New(nc, logger)
	svc := coap.New(pubsub)
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

	go func() {
		p := fmt.Sprintf(":%d", cfg.Port)
		logger.Info(fmt.Sprintf("CoAP adapter service started, exposed port %d", cfg.Port))
		errs <- api.ListenAndServe(svc, cc, p)
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	err = <-errs
	logger.Error(fmt.Sprintf("CoAP adapter terminated: %s", err))
}
