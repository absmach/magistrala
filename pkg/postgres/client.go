// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/jmoiron/sqlx"
)

var errMigration = errors.New("failed to apply migrations")

type PoolConfig struct {
	// pool_max_conns
	MaxConns uint16 `env:"MAX_CONNS" envDefault:"5"`

	// pool_min_conns
	MinConns uint16 `env:"MIN_CONNS" envDefault:"1"`

	// pool_max_conn_lifetime , example: 1h30m
	MaxConnLifetime string `env:"MAX_CONN_LIFETIME" envDefault:"1h"`

	// pool_max_conn_idle_time, example: 30m
	MaxConnIdleTime string `env:"MAX_CONN_IDLE_TIME" envDefault:"15m"`

	// pool_health_check_period
	HealthCheckPeriod string `env:"HEALTH_CHECK_PERIOD" envDefault:"1m"`

	// pool_max_conn_lifetime_jitter
	MaxConnLifetimeJitter uint16 `env:"MAX_CONN_LIFETIME_JITTER" envDefault:"0"`
}

// Config defines the options that are used when connecting to a TimescaleSQL instance.
type Config struct {
	Host        string     `env:"HOST"           envDefault:"localhost"`
	Port        string     `env:"PORT"           envDefault:"5432"`
	User        string     `env:"USER"           envDefault:"supermq"`
	Pass        string     `env:"PASS"           envDefault:"supermq"`
	Name        string     `env:"NAME"           envDefault:""`
	SSLMode     string     `env:"SSL_MODE"       envDefault:"disable"`
	SSLCert     string     `env:"SSL_CERT"       envDefault:""`
	SSLKey      string     `env:"SSL_KEY"        envDefault:""`
	SSLRootCert string     `env:"SSL_ROOT_CERT"  envDefault:""`
	Pool        PoolConfig `envPrefix:"POOL"`
}

// Setup creates a connection to the Postgres instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate
// failure.
func Setup(cfg Config, migrations migrate.MemoryMigrationSource) (*sqlx.DB, error) {
	db, err := Connect(cfg)
	if err != nil {
		return nil, err
	}

	if _, err = migrate.Exec(db.DB, "postgres", migrations, migrate.Up); err != nil {
		return nil, errors.Wrap(errMigration, err)
	}

	return db, nil
}

// Connect creates a connection to the Postgres instance.
func Connect(cfg Config) (*sqlx.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s pool_max_conns=%d pool_min_conns=%d pool_max_conn_lifetime=%s pool_max_conn_idle_time=%s pool_health_check_period=%s pool_max_conn_lifetime_jitter=%d",
		cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert, cfg.Pool.MaxConns, cfg.Pool.MinConns, cfg.Pool.MaxConnLifetime, cfg.Pool.MaxConnIdleTime, cfg.Pool.HealthCheckPeriod, cfg.Pool.MaxConnLifetimeJitter)

	dbpool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return nil, err
	}

	sqlDB := stdlib.OpenDBFromPool(dbpool, nil)

	return sqlx.NewDb(sqlDB, "pgx"), nil
}
