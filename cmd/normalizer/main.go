package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mainflux/mainflux/normalizer"
	"github.com/mainflux/mainflux/normalizer/nats"
	broker "github.com/nats-io/go-nats"
	"go.uber.org/zap"
)

const (
	subject    string = "adapter.*"
	queue      string = "normalizers"
	defNatsURL string = broker.DefaultURL
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

	repo := nats.NewMessageRepository(nc)
	svc := normalizer.NewService(repo)

	nc.QueueSubscribe(subject, queue, func(m *broker.Msg) {
		msg := normalizer.Message{}

		if err := json.Unmarshal(m.Data, &msg); err != nil {
			logger.Error("Failed to unmarshal JSON", zap.Error(err))
			return
		}

		if msgs, err := normalizer.Normalize(msg); err != nil {
			logger.Error("Failed to normalize message", zap.Error(err))
		} else {
			svc.Send(msgs)
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

func connectToNats(cfg *config) *broker.Conn {
	nc, err := broker.Connect(cfg.NatsURL)
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
