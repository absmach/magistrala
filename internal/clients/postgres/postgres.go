// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	"github.com/absmach/magistrala/pkg/errors"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

var (
	errConnect   = errors.New("failed to connect to postgresql server")
	errMigration = errors.New("failed to apply migrations")
)

type Config struct {
	Host        string `env:"HOST"           envDefault:"localhost"`
	Port        string `env:"PORT"           envDefault:"5432"`
	User        string `env:"USER"           envDefault:"magistrala"`
	Pass        string `env:"PASS"           envDefault:"magistrala"`
	Name        string `env:"NAME"           envDefault:""`
	SSLMode     string `env:"SSL_MODE"       envDefault:"disable"`
	SSLCert     string `env:"SSL_CERT"       envDefault:""`
	SSLKey      string `env:"SSL_KEY"        envDefault:""`
	SSLRootCert string `env:"SSL_ROOT_CERT"  envDefault:""`
}

// Setup creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate failure.
func Setup(cfg Config, migrations migrate.MemoryMigrationSource) (*sqlx.DB, error) {
	db, err := Connect(cfg)
	if err != nil {
		return nil, err
	}

	_, err = migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		return nil, errors.Wrap(errMigration, err)
	}

	return db, nil
}

// Connect creates a connection to the PostgreSQL instance.
func Connect(cfg Config) (*sqlx.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s", cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert)

	db, err := sqlx.Open("pgx", url)
	if err != nil {
		return nil, errors.Wrap(errConnect, err)
	}

	return db, nil
}
