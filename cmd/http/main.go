package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	kitconsul "github.com/go-kit/kit/sd/consul"
	stdconsul "github.com/hashicorp/consul/api"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/http/api"
	"github.com/mainflux/mainflux/http/nats"
	broker "github.com/nats-io/go-nats"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	uuid "github.com/satori/go.uuid"
)

const (
	port    int    = 9002
	natsKey string = "nats"
)

var (
	kv     *stdconsul.KV
	logger log.Logger
)

func main() {
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		logger.Log("status", "Cannot start the service: CONSUL_ADDR not set.")
		os.Exit(1)
	}

	consul, err := stdconsul.NewClient(&stdconsul.Config{
		Address: consulAddr,
	})

	if err != nil {
		status := fmt.Sprintf("Cannot connect to Consul due to %s", err)
		logger.Log("status", status)
		os.Exit(1)
	}

	kv = consul.KV()

	asr := &stdconsul.AgentServiceRegistration{
		ID:                uuid.NewV4().String(),
		Name:              "http-adapter",
		Tags:              []string{},
		Port:              port,
		Address:           "",
		EnableTagOverride: false,
	}

	sd := kitconsul.NewClient(consul)
	if err = sd.Register(asr); err != nil {
		status := fmt.Sprintf("Cannot register service due to %s", err)
		logger.Log("status", status)
		os.Exit(1)
	}

	nc, err := broker.Connect(get(natsKey))
	if err != nil {
		status := fmt.Sprintf("Cannot connect to NATS due to %s", err.Error())
		logger.Log("status", status)
		os.Exit(1)
	}
	defer nc.Close()

	repo := nats.NewMessageRepository(nc)
	svc := adapter.NewService(repo)

	svc = api.NewLoggingService(logger, svc)

	fields := []string{"method"}
	svc = api.NewMetricService(
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "http_adapter",
			Subsystem: "api",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, fields),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "http_adapter",
			Subsystem: "api",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, fields),
		svc,
	)

	errChan := make(chan error, 10)

	go func() {
		p := fmt.Sprintf(":%d", port)
		logger.Log("status", "HTTP adapter started.")
		errChan <- http.ListenAndServe(p, api.MakeHandler(svc))
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			status := fmt.Sprintf("HTTP adapter stopped due to %s", err)
			logger.Log("status", status)
			sd.Deregister(asr)
			os.Exit(1)
		case <-sigChan:
			status := fmt.Sprintf("HTTP adapter terminated.")
			logger.Log("status", status)
			sd.Deregister(asr)
			os.Exit(0)
		}
	}
}

func get(key string) string {
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		status := fmt.Sprintf("Cannot retrieve %s due to %s", key, err)
		logger.Log("status", status)
		os.Exit(1)
	}

	return string(pair.Value)
}
