//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package postgres_test contains tests for PostgreSQL repository
// implementations.
package postgres_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things/postgres"
	"gopkg.in/ory-am/dockertest.v3"
)

const (
	wrongID    = 0
	wrongValue = "wrong-value"
)

var (
	testLog, _ = logger.New(os.Stdout, logger.Info.String())
	db         *sql.DB
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	cfg := []string{
		"POSTGRES_USER=test",
		"POSTGRES_PASSWORD=test",
		"POSTGRES_DB=test",
	}
	container, err := pool.Run("postgres", "10.2-alpine", cfg)
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	port := container.GetPort("5432/tcp")

	if err := pool.Retry(func() error {
		url := fmt.Sprintf("host=localhost port=%s user=test dbname=test password=test sslmode=disable", port)
		db, err = sql.Open("postgres", url)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	if db, err = postgres.Connect("localhost", port, "test", "test", "test", "disable"); err != nil {
		log.Fatalf("Could not setup test DB connection: %s", err)
	}
	defer db.Close()

	code := m.Run()

	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}
