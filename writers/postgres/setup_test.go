//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package postgres_test contains tests for PostgreSQL repository
// implementations.
package postgres_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/writers/postgres"
	dockertest "gopkg.in/ory-am/dockertest.v3"
)

const (
	wrongID    = "0"
	wrongValue = "wrong-value"
)

var (
	testLog, _ = logger.New(os.Stdout, logger.Info.String())
	db         *sqlx.DB
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
		db, err = sqlx.Open("postgres", url)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	dbConfig := postgres.Config{
		Host:        "localhost",
		Port:        port,
		User:        "test",
		Pass:        "test",
		Name:        "test",
		SSLMode:     "disable",
		SSLCert:     "",
		SSLKey:      "",
		SSLRootCert: "",
	}

	db, err = postgres.Connect(dbConfig)
	if err != nil {
		log.Fatalf("Could not setup test DB connection: %s", err)
	}
	defer db.Close()

	code := m.Run()

	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}
