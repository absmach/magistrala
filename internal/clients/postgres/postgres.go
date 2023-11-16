// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/caarlos0/env/v10"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

var (
	errConfig    = errors.New("failed to load postgresql configuration")
	errConnect   = errors.New("failed to connect to postgresql server")
	errMigration = errors.New("failed to apply migrations")
)

type Config struct {
	Host        string `env:"HOST,notEmpty"           envDefault:"localhost"`
	Port        string `env:"PORT,notEmpty"           envDefault:"5432"`
	User        string `env:"USER,notEmpty"           envDefault:"magistrala"`
	Pass        string `env:"PASS,notEmpty"           envDefault:"magistrala"`
	Name        string `env:"NAME"                    envDefault:""`
	SSLMode     string `env:"SSL_MODE,notEmpty"       envDefault:"disable"`
	SSLCert     string `env:"SSL_CERT"                envDefault:""`
	SSLKey      string `env:"SSL_KEY"                 envDefault:""`
	SSLRootCert string `env:"SSL_ROOT_CERT"           envDefault:""`
}

// Setup creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate failure.
func Setup(prefix string, migrations migrate.MemoryMigrationSource) (*sqlx.DB, error) {
	return SetupWithConfig(prefix, migrations, Config{})
}

// SetupWithConfig creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate failure.
func SetupWithConfig(prefix string, migrations migrate.MemoryMigrationSource, defConfig Config) (*sqlx.DB, error) {
	cfg := defConfig
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: prefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	return SetupDB(cfg, migrations)
}

// SetupDB creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate failure.
func SetupDB(cfg Config, migrations migrate.MemoryMigrationSource) (*sqlx.DB, error) {
	db, err := Connect(cfg)
	if err != nil {
		return nil, err
	}
	if err := MigrateDB(db, migrations); err != nil {
		return nil, err
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

// MigrateDB applies any unapplied database migrations.
func MigrateDB(db *sqlx.DB, migrations migrate.MemoryMigrationSource) error {
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		return errors.Wrap(errMigration, err)
	}
	return nil
}

func (c *Config) LoadEnv(prefix string) error {
	if err := env.ParseWithOptions(c, env.Options{Prefix: prefix}); err != nil {
		return errors.Wrap(errConfig, err)
	}
	return nil
}
