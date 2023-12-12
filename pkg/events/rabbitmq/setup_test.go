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

	"github.com/absmach/magistrala/pkg/events/rabbitmq"
	"github.com/ory/dockertest/v3"
)

var (
	rabbitmqURL string
	stream      = "tests.events"
	consumer    = "tests-consumer"
	ctx         = context.TODO()
	pool        = &dockertest.Pool{}
	container   = &dockertest.Resource{}
)

func TestMain(m *testing.M) {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	opts := dockertest.RunOptions{
		Name:       "test-rabbitmq-events",
		Repository: "rabbitmq",
		Tag:        "3.9.20",
	}
	container, err = pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	handleInterrupt(pool, container)

	rabbitmqURL = fmt.Sprintf("amqp://%s:%s", "localhost", container.GetPort("5672/tcp"))

	if err := pool.Retry(func() error {
		_, err = rabbitmq.NewPublisher(ctx, rabbitmqURL, stream)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	if err := pool.Retry(func() error {
		_, err = rabbitmq.NewSubscriber(rabbitmqURL, stream, consumer, logger)
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
