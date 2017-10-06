package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/writer"
	"github.com/mainflux/mainflux/writer/cassandra"
	nats "github.com/nats-io/go-nats"
	"go.uber.org/zap"
)

const (
	sep         string = ","
	subject     string = "msg.*"
	queue       string = "message_writers"
	defCluster  string = "127.0.0.1"
	defKeyspace string = "message_writer"
	defNatsURL  string = nats.DefaultURL
	envCluster  string = "MESSAGE_WRITER_DB_CLUSTER"
	envKeyspace string = "MESSAGE_WRITER_DB_KEYSPACE"
	envNatsURL  string = "MESSAGE_WRITER_NATS_URL"
)

var logger *zap.Logger

type config struct {
	Cluster  string
	Keyspace string
	NatsURL  string
}

func main() {
	cfg := loadConfig()

	logger, _ = zap.NewProduction()
	defer logger.Sync()

	session := connectToCassandra(cfg)
	defer session.Close()

	nc := connectToNats(cfg)
	defer nc.Close()

	repo := makeRepository(session)

	nc.QueueSubscribe(subject, queue, func(m *nats.Msg) {
		msg := writer.RawMessage{}

		if err := json.Unmarshal(m.Data, &msg); err != nil {
			logger.Error("Failed to unmarshal raw message.", zap.Error(err))
			return
		}

		if err := repo.Save(msg); err != nil {
			logger.Error("Failed to save message.", zap.Error(err))
			return
		}
	})

	forever()
}

func loadConfig() *config {
	return &config{
		Cluster:  env(envCluster, defCluster),
		Keyspace: env(envKeyspace, defKeyspace),
		NatsURL:  env(envNatsURL, defNatsURL),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func connectToCassandra(cfg *config) *gocql.Session {
	hosts := strings.Split(cfg.Cluster, sep)

	s, err := cassandra.Connect(hosts, cfg.Keyspace)
	if err != nil {
		logger.Error("Failed to connect to DB", zap.Error(err))
	}

	return s
}

func makeRepository(session *gocql.Session) writer.MessageRepository {
	if err := cassandra.Initialize(session); err != nil {
		logger.Error("Failed to initialize message repository.", zap.Error(err))
	}

	return cassandra.NewMessageRepository(session)
}

func connectToNats(cfg *config) *nats.Conn {
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		logger.Error("Failed to connect to NATS.", zap.Error(err))
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
