package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	kitconsul "github.com/go-kit/kit/sd/consul"
	stdconsul "github.com/hashicorp/consul/api"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/writer"
	"github.com/mainflux/mainflux/writer/cassandra"
	nats "github.com/nats-io/go-nats"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
)

const (
	port     int    = 9001
	dbKey    string = "cassandra"
	natsKey  string = "nats"
	sep      string = ","
	keyspace string = "message_writer"
	group    string = "writers"
	subject  string = "msg.*"
)

var (
	kv     *stdconsul.KV
	logger *zap.Logger
)

func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()

	consulAddr := os.Getenv("CONSUL_ADDR")
	if consulAddr == "" {
		logger.Fatal("Cannot start the service: CONSUL_ADDR not set.")
	}

	consul, err := stdconsul.NewClient(&stdconsul.Config{
		Address: consulAddr,
	})

	if err != nil {
		status := fmt.Sprintf("Cannot connect to Consul due to %s", err)
		logger.Fatal(status)
	}

	kv = consul.KV()

	asr := &stdconsul.AgentServiceRegistration{
		ID:                uuid.NewV4().String(),
		Name:              "writer",
		Tags:              []string{},
		Port:              port,
		Address:           "",
		EnableTagOverride: false,
	}

	sd := kitconsul.NewClient(consul)
	if err = sd.Register(asr); err != nil {
		status := fmt.Sprintf("Cannot register service due to %s", err)
		logger.Fatal(status)
	}

	hosts := strings.Split(get(dbKey), sep)
	session, err := cassandra.Connect(hosts, keyspace)
	if err != nil {
		logger.Fatal("Cannot connect to DB.", zap.Error(err))
	}
	defer session.Close()

	nc, err := nats.Connect(get(natsKey))
	if err != nil {
		logger.Fatal("Cannot connect to NATS.", zap.Error(err))
	}
	defer nc.Close()

	if err := cassandra.Initialize(session); err != nil {
		logger.Fatal("Cannot initialize message repository.", zap.Error(err))
	}

	repo := cassandra.NewMessageRepository(session)

	nc.QueueSubscribe(subject, group, func(m *nats.Msg) {
		msg := writer.RawMessage{}

		if err := json.Unmarshal(m.Data, &msg); err != nil {
			logger.Error("Failed to unmarshal raw message.", zap.Error(err))
			return
		}

		if err := repo.Save(msg); err != nil {
			logger.Error("Failed to save message.", zap.Error(err))
		}
	})

	errChan := make(chan error, 10)

	go func() {
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mainflux.Version(),
		}

		logger.Info("Writer started.")

		errChan <- server.ListenAndServe()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			status := fmt.Sprintf("Writer terminated due to %s", err)
			logger.Fatal(status)
		case <-sigChan:
			logger.Info("Writer terminated.")
		}
	}
}

func get(key string) string {
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		status := fmt.Sprintf("Cannot retrieve %s due to %s", key, err)
		logger.Fatal(status)
	}

	return string(pair.Value)
}
