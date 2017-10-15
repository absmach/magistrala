package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	kitconsul "github.com/go-kit/kit/sd/consul"
	stdconsul "github.com/hashicorp/consul/api"
	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/nats"
	broker "github.com/nats-io/go-nats"
	uuid "github.com/satori/go.uuid"

	"go.uber.org/zap"
)

const (
	port    int    = 9003
	natsKey string = "nats"
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
		Name:              "coap-adapter",
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

	nc, err := broker.Connect(get(natsKey))
	if err != nil {
		logger.Fatal("Cannot connect to NATS.", zap.Error(err))
	}
	defer nc.Close()

	repo := nats.NewMessageRepository(nc)
	ca := adapter.NewCoAPAdapter(logger, repo)

	nc.Subscribe("msg.http", ca.BridgeHandler)
	nc.Subscribe("msg.mqtt", ca.BridgeHandler)

	errChan := make(chan error, 10)

	go func() {
		p := fmt.Sprintf(":%d", port)
		logger.Info("CoAP adapter started.")
		errChan <- ca.Serve(p)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case err := <-errChan:
			status := fmt.Sprintf("CoAP adapter terminated due to %s", err)
			logger.Fatal(status)
		case <-sigChan:
			logger.Info("CoAP adapter terminated.")
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
