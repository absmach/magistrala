// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package postgres_test contains tests for PostgreSQL repository
// implementations.
package postgres_test

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	apostgres "github.com/absmach/supermq/auth/postgres"
	"github.com/absmach/supermq/pkg/postgres"
	pgclient "github.com/absmach/supermq/pkg/postgres"
	"github.com/jmoiron/sqlx"
	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"go.opentelemetry.io/otel"
)

var (
	db       *sqlx.DB
	database postgres.Database
	tracer   = otel.Tracer("repo_tests")
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "16.2-alpine",
		Env: []string{
			"POSTGRES_USER=test",
			"POSTGRES_PASSWORD=test",
			"POSTGRES_DB=test",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start container: %s", err)
	}

	port := container.GetPort("5432/tcp")

	pool.MaxWait = 120 * time.Second
	if err := pool.Retry(func() error {
		url := fmt.Sprintf("host=localhost port=%s user=test dbname=test password=test sslmode=disable", port)
		db, err := sql.Open("pgx", url)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	dbConfig := pgclient.Config{
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

	if db, err = pgclient.Setup(dbConfig, *apostgres.Migration()); err != nil {
		log.Fatalf("Could not setup test DB connection: %s", err)
	}

	database = postgres.NewDatabase(db, dbConfig, tracer)

	code := m.Run()

	// Defers will not be run when using os.Exit
	db.Close()
	if err := pool.Purge(container); err != nil {
		log.Fatalf("Could not purge container: %s", err)
	}

	os.Exit(code)
}
