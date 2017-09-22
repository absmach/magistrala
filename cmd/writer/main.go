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
	log "github.com/sirupsen/logrus"
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

type config struct {
	Cluster  string
	Keyspace string
	NatsURL  string
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
}

func main() {
	cfg := loadConfig()

	session := connectToCassandra(cfg)
	defer session.Close()

	nc := connectToNats(cfg)
	defer nc.Close()

	repo := makeRepository(session)

	nc.QueueSubscribe(subject, queue, func(m *nats.Msg) {
		msg := writer.Message{}

		if err := json.Unmarshal(m.Data, &msg); err == nil {
			repo.Save(msg)
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
		log.WithField("error", err).Fatalf("Failed to connect to DB.")
	}

	return s
}

func makeRepository(session *gocql.Session) writer.MessageRepository {
	if err := cassandra.Initialize(session); err != nil {
		log.WithField("error", err).Fatalf("Failed to initialize message repository.")
	}

	return cassandra.NewMessageRepository(session)
}

func connectToNats(cfg *config) *nats.Conn {
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.WithField("error", err).Fatalf("Failed to connect to NATS.")
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
