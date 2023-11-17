// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package nats_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/nats"
	"github.com/ory/dockertest/v3"
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
		Repository: "nats",
		Tag:        "2.9.21-alpine",
		Cmd:        []string{"-DVV", "-js"},
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}
	handleInterrupt(pool, container)

	address := fmt.Sprintf("nats://%s:%s", "localhost", container.GetPort("4222/tcp"))
	if err := pool.Retry(func() error {
		publisher, err = nats.NewPublisher(context.Background(), address)
		return err
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	logger, err := mglog.New(os.Stdout, "error")
	if err != nil {
		log.Fatalf(err.Error())
	}
	if err := pool.Retry(func() error {
		pubsub, err = nats.NewPubSub(context.Background(), address, logger)
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
