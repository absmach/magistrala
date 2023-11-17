// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !nats && !rabbitmq
// +build !nats,!rabbitmq

package redis_test

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
)

type client struct {
	*redis.Client
	url       string
	pool      *dockertest.Pool
	container *dockertest.Resource
}

var (
	redisClient *redis.Client
	redisURL    string
)

func TestMain(m *testing.M) {
	client, err := startContainer()
	if err != nil {
		log.Fatalf(err.Error())
	}
	redisClient = client.Client
	redisURL = client.url

	code := m.Run()

	if err := client.pool.Purge(client.container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}

func startContainer() (client, error) {
	var cli client
	pool, err := dockertest.NewPool("")
	if err != nil {
		return client{}, fmt.Errorf("Could not connect to docker: %s", err)
	}
	cli.pool = pool

	container, err := cli.pool.Run("redis", "7.2.0-alpine", nil)
	if err != nil {
		return client{}, fmt.Errorf("Could not start container: %s", err)
	}
	cli.container = container

	handleInterrupt(cli.pool, cli.container)

	cli.url = fmt.Sprintf("redis://localhost:%s/0", cli.container.GetPort("6379/tcp"))
	opts, err := redis.ParseURL(cli.url)
	if err != nil {
		return client{}, fmt.Errorf("Could not parse redis URL: %s", err)
	}

	if err := pool.Retry(func() error {
		cli.Client = redis.NewClient(opts)

		return cli.Client.Ping(ctx).Err()
	}); err != nil {
		return client{}, fmt.Errorf("Could not connect to docker: %s", err)
	}

	return cli, nil
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
