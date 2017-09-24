package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mainflux/mainflux/normalizer"
	nats "github.com/nats-io/go-nats"
	"go.uber.org/zap"
)

const (
	subSubject string = "adapter.*"
	pubSubject string = "normalizer.senml"
	queue      string = "normalizers"
	defNatsURL string = nats.DefaultURL
	envNatsURL string = "MESSAGE_WRITER_NATS_URL"
)

type config struct {
	NatsURL string
}

var logger *zap.Logger = nil

func main() {
	cfg := loadConfig()

	logger, _ = zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any

	nc := connectToNats(cfg)
	defer nc.Close()

	nc.QueueSubscribe(subSubject, queue, func(m *nats.Msg) {
		msg := normalizer.Message{}

		if err := json.Unmarshal(m.Data, &msg); err != nil {
			logger.Error("Failed to unmarshal JSON", zap.Error(err))
			return
		}

		if msgs, err := normalizer.Normalize(logger, msg); err != nil {
			logger.Error("Failed to normalize message", zap.Error(err))
		} else {
			for _, v := range msgs {
				if b, err := json.Marshal(v); err != nil {
					logger.Error("Failed to marshal writer message", zap.Error(err))
				} else {
					nc.Publish(pubSubject, b)
				}
			}
		}
	})

	logger.Info("Starting normalizer")

	forever()
}

func loadConfig() *config {
	return &config{
		NatsURL: env(envNatsURL, defNatsURL),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func connectToNats(cfg *config) *nats.Conn {
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		logger.Error("Failed to connect to NATS", zap.Error(err))
		os.Exit(1)
	}

	return nc
}

func forever() {
	errs := make(chan error, 1)

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	<-errs
}
