// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	dockertest "github.com/ory/dockertest/v3"
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		testLog.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	cfg := []string{
		"MONGO_INITDB_DATABASE=test",
	}

	container, err := pool.Run("mongo", "4.4.6", cfg)
	if err != nil {
		testLog.Error(fmt.Sprintf("Could not start container: %s", err))
	}

	port = container.GetPort("27017/tcp")
	addr = fmt.Sprintf("mongodb://localhost:%s", port)

	if err := pool.Retry(func() error {
		_, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
		return err
	}); err != nil {
		testLog.Error(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	code := m.Run()

	if err := pool.Purge(container); err != nil {
		testLog.Error(fmt.Sprintf("Could not purge container: %s", err))
	}

	os.Exit(code)
}
