// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mqtt_test

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	mqttpubsub "github.com/absmach/magistrala/pkg/messaging/mqtt"
	mglog "github.com/absmach/supermq/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	pubsub  messaging.PubSub
	logger  *slog.Logger
	address string
)

const (
	username      = "magistrala-mqtt"
	qos           = 2
	port          = "1883/tcp"
	brokerTimeout = 30 * time.Second
	poolMaxWait   = 120 * time.Second
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "eclipse-mosquitto",
		Tag:        "1.6.15",
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	handleInterrupt(pool, container)

	address = fmt.Sprintf("%s:%s", "localhost", container.GetPort(port))
	pool.MaxWait = poolMaxWait

	logger, err = mglog.New(os.Stdout, "debug")
	if err != nil {
		log.Fatal(err.Error())
	}

	if err := pool.Retry(func() error {
		pubsub, err = mqttpubsub.NewPubSub(address, 2, brokerTimeout, logger)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()
	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)

	defer func() {
		err = pubsub.Close()
		if err != nil {
			log.Fatal(err.Error())
		}
	}()
}

func handleInterrupt(pool *dockertest.Pool, container *dockertest.Resource) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		if err := pool.Purge(container); err != nil {
			log.Fatalf("Could not purge container: %s", err)
		}
		os.Exit(0)
	}()
}

func newClient(address, id string, timeout time.Duration) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		SetUsername(username).
		AddBroker(address).
		SetClientID(id)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Error() != nil {
		return nil, token.Error()
	}

	ok := token.WaitTimeout(timeout)
	if !ok {
		return nil, mqttpubsub.ErrConnect
	}

	if token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}
