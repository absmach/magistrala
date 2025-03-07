// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package rabbitmq_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/absmach/supermq/pkg/events/rabbitmq"
	"github.com/ory/dockertest/v3"
)

var (
	rabbitmqURL string
	stream      = "tests.events"
	consumer    = "tests-consumer"
	pool        *dockertest.Pool
	container   *dockertest.Resource
)

func TestMain(m *testing.M) {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err = pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "rabbitmq",
		Tag:        "3.12.12",
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	handleInterrupt(pool, container)

	rabbitmqURL = fmt.Sprintf("amqp://%s:%s", "localhost", container.GetPort("5672/tcp"))

	if err := pool.Retry(func() error {
		_, err = rabbitmq.NewPublisher(context.Background(), rabbitmqURL, stream)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	if err := pool.Retry(func() error {
		_, err = rabbitmq.NewSubscriber(rabbitmqURL, logger)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()

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
