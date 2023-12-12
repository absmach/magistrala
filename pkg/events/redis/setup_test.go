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

var (
	redisClient *redis.Client
	redisURL    string
	pool        *dockertest.Pool
	container   *dockertest.Resource
)

func TestMain(m *testing.M) {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	opts := dockertest.RunOptions{
		Name:       "tests-redis-events",
		Repository: "redis",
		Tag:        "7.2.0-alpine",
	}
	container, err = pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	handleInterrupt(pool, container)

	redisURL = fmt.Sprintf("redis://localhost:%s/0", container.GetPort("6379/tcp"))
	ropts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Could not parse redis URL: %s", err)

	}

	if err := pool.Retry(func() error {
		redisClient = redis.NewClient(ropts)

		return redisClient.Ping(ctx).Err()
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
