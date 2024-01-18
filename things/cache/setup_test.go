// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/ory/dockertest/v3"
)

var (
	redisClient *redis.Client
	redisURL    string
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.Run("redis", "7.2.0-alpine", nil)
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	redisURL = fmt.Sprintf("redis://localhost:%s/0", container.GetPort("6379/tcp"))
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Could not parse redis URL: %s", err)
	}

	if err := pool.Retry(func() error {
		redisClient = redis.NewClient(opts)

		return redisClient.Ping(context.Background()).Err()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()

	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}
