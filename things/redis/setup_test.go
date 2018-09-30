//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/go-redis/redis"
	dockertest "gopkg.in/ory-am/dockertest.v3"
)

const (
	wrongID    = 0
	wrongValue = "wrong-value"
)

var (
	cacheClient *redis.Client
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.Run("redis", "4.0.9-alpine", nil)
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	// When you're done, kill and remove the container
	defer pool.Purge(container)

	if err := pool.Retry(func() error {
		cacheClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("localhost:%s", container.GetPort("6379/tcp")),
			Password: "",
			DB:       0,
		})

		return cacheClient.Ping().Err()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	code := m.Run()

	os.Exit(code)
}
