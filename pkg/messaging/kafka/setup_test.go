// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package kafka_test

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/kafka"
	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var (
	publisher messaging.Publisher
	pubsub    messaging.PubSub
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "spotify/kafka",
		Tag:        "latest",
		Env: []string{
			"ADVERTISED_HOST=127.0.0.1",
			"ADVERTISED_PORT=9092",
		},
		ExposedPorts: []string{
			"9092",
			"2181",
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"9092/tcp": {{HostIP: "localhost", HostPort: "9092/tcp"}},
			"2181/tcp": {{HostIP: "localhost", HostPort: "2181/tcp"}},
		},
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}
	handleInterrupt(pool, container)
	address := fmt.Sprintf("%s:%s", "localhost", container.GetPort("9092/tcp"))

	// As kafka doesn't support a readiness endpoint we have to ensure that kafka is ready before we test it.
	// When you immediately start testing it will throw an EOF error thus we should wait for sometime
	// before starting the tests after bringing the docker container up
	time.Sleep(10 * time.Second)

	if err := pool.Retry(func() error {
		publisher, err = kafka.NewPublisher(address)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	logger, err := mglog.New(os.Stdout, "error")
	if err != nil {
		log.Fatalf(err.Error())
	}
	if err := pool.Retry(func() error {
		pubsub, err = kafka.NewPubSub(address, "", logger)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()
	if err := pubsub.Close(); err != nil {
		log.Fatalf("Could not close pubsub: %s", err)
	}
	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
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
