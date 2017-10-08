package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/nats"

	broker "github.com/nats-io/go-nats"
	"go.uber.org/zap"
)

const (
	port       int    = 5683
	defNatsURL string = broker.DefaultURL
	envNatsURL string = "COAP_ADAPTER_NATS_URL"
)

type config struct {
	Port    int
	NatsURL string
}

func main() {
	cfg := loadConfig()

	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any

	nc := connectToNats(cfg, logger)
	defer nc.Close()

	repo := nats.NewMessageRepository(nc)
	ca := adapter.NewCoAPAdapter(logger, repo)

	nc.Subscribe("msg.http", ca.BridgeHandler)
	nc.Subscribe("msg.mqtt", ca.BridgeHandler)

	errs := make(chan error, 2)

	go func() {
		coapAddr := fmt.Sprintf(":%d", cfg.Port)
		errs <- ca.Serve(coapAddr)
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	c := <-errs
	logger.Info("terminated", zap.String("error", c.Error()))
}

func loadConfig() *config {
	return &config{
		NatsURL: env(envNatsURL, defNatsURL),
		Port:    port,
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func connectToNats(cfg *config, logger *zap.Logger) *broker.Conn {
	nc, err := broker.Connect(cfg.NatsURL)
	if err != nil {
		logger.Error("Failed to connect to NATS", zap.Error(err))
		os.Exit(1)
	}

	return nc
}
