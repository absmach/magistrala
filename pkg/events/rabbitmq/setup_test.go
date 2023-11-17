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

type client struct {
	url       string
	pool      *dockertest.Pool
	container *dockertest.Resource
}

var (
	rabbitmqURL string
	stream      = "tests.events"
	consumer    = "tests-consumer"
	ctx         = context.TODO()
)

func TestMain(m *testing.M) {
	client, err := startContainer()
	if err != nil {
		log.Fatalf(err.Error())
	}
	rabbitmqURL = client.url

	code := m.Run()

	if err := client.pool.Purge(client.container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}

func handleInterrupt(pool *dockertest.Pool, container *dockertest.Resource) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		if err := pool.Purge(container); err != nil {
			log.Fatalf("Could not purge container: %s", err)
		}
		os.Exit(0)
	}()
}

func startContainer() (client, error) {
	var cli client
	var err error
	cli.pool, err = dockertest.NewPool("")
	if err != nil {
		return client{}, fmt.Errorf("Could not connect to docker: %s", err)
	}

	cli.container, err = cli.pool.Run("rabbitmq", "3.9.20", []string{})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	handleInterrupt(cli.pool, cli.container)

	cli.url = fmt.Sprintf("amqp://%s:%s", "localhost", cli.container.GetPort("5672/tcp"))

	if err := cli.pool.Retry(func() error {
		_, err = rabbitmq.NewPublisher(ctx, cli.url, stream)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	if err := cli.pool.Retry(func() error {
		_, err = rabbitmq.NewSubscriber(cli.url, stream, consumer, logger)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	return cli, nil
}
