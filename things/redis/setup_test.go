// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-redis/redis/v8"
	dockertest "github.com/ory/dockertest/v3"
)

const (
	wrongValue = "wrong-value"
)

var redisClient *redis.Client

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.Run("redis", "5.0-alpine", nil)
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	if err := pool.Retry(func() error {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("localhost:%s", container.GetPort("6379/tcp")),
			Password: "",
			DB:       0,
		})

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
