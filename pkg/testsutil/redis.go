// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package testsutil

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
)

type redisContainer struct {
	Client *redis.Client
	URL    string
}

func SetupRedis() (*redisContainer, func(), error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	container, err := pool.Run("docker.io/redis", "8.2.2-alpine3.22", nil)
	if err != nil {
		return nil, nil, fmt.Errorf("could not start container: %w", err)
	}

	storeURL := fmt.Sprintf("redis://localhost:%s/0", container.GetPort("6379/tcp"))
	opts, err := redis.ParseURL(storeURL)
	if err != nil {
		_ = pool.Purge(container)
		return nil, nil, fmt.Errorf("could not parse redis URL: %w", err)
	}
	opts.MaintNotificationsConfig = &maintnotifications.Config{
		Mode: maintnotifications.ModeDisabled,
	}

	var storeClient *redis.Client
	if err := pool.Retry(func() error {
		storeClient = redis.NewClient(opts)
		return storeClient.Ping(context.Background()).Err()
	}); err != nil {
		_ = pool.Purge(container)
		return nil, nil, fmt.Errorf("could not connect to docker: %w", err)
	}

	cleanup := func() {
		if err := pool.Purge(container); err != nil {
			log.Fatalf("Could not purge container: %s", err)
		}
	}

	return &redisContainer{
		Client: storeClient,
		URL:    storeURL,
	}, cleanup, nil
}

func RunRedisTest(m *testing.M, storeClient **redis.Client, storeURL *string) int {
	container, cleanup, err := SetupRedis()
	if err != nil {
		log.Fatalf("Failed to setup Redis: %s", err)
	}
	defer cleanup()

	*storeClient = container.Client
	*storeURL = container.URL

	return m.Run()
}
