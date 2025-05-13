// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/jmoiron/sqlx"
)

var (
	errMigration               = errors.New("failed to apply migrations")
	errInvalidConnectionString = errors.New("invalid connection string")
)

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
	pgxPoolConfig, err := pgxpool.ParseConfig(cfg.URL())
	if err != nil {
		return nil, errors.Wrap(errInvalidConnectionString, err)
	}

	dbpool, err := pgxpool.NewWithConfig(context.Background(), pgxPoolConfig)
	if err != nil {
		return nil, err
	}

	beforeConnect := stdlib.OptionBeforeConnect(func(ctx context.Context, pgxConfig *pgx.ConnConfig) error {
		return nil
	})

	afterConnect := stdlib.OptionAfterConnect(func(ctx context.Context, conn *pgx.Conn) error {
		return nil
	})

	resetSession := stdlib.OptionResetSession(func(ctx context.Context, c *pgx.Conn) error {
		return nil
	})

	sqlDB := stdlib.OpenDBFromPool(dbpool, beforeConnect, afterConnect, resetSession)

	return sqlx.NewDb(sqlDB, "pgx"), nil
}

func (cfg Config) URL() string {
	urlParts := []string{}

	if cfg.Host != "" {
		urlParts = append(urlParts, "host="+cfg.Host)
	}
	if cfg.Port != "" {
		urlParts = append(urlParts, "port="+cfg.Port)
	}
	if cfg.User != "" {
		urlParts = append(urlParts, "user="+cfg.User)
	}
	if cfg.Pass != "" {
		urlParts = append(urlParts, "password="+cfg.Pass)
	}
	if cfg.Name != "" {
		urlParts = append(urlParts, "dbname="+cfg.Name)
	}
	if cfg.SSLMode != "" {
		urlParts = append(urlParts, "sslmode="+cfg.SSLMode)
	}
	if cfg.SSLCert != "" {
		urlParts = append(urlParts, "sslcert="+cfg.SSLCert)
	}
	if cfg.SSLKey != "" {
		urlParts = append(urlParts, "sslkey="+cfg.SSLKey)
	}
	if cfg.SSLRootCert != "" {
		urlParts = append(urlParts, "sslrootcert="+cfg.SSLRootCert)
	}
	urlParts = append(urlParts, fmt.Sprintf("pool_max_conns=%d", cfg.Pool.MaxConns))
	urlParts = append(urlParts, fmt.Sprintf("pool_min_conns=%d", cfg.Pool.MinConns))
	if cfg.Pool.MaxConnLifetime != "" {
		urlParts = append(urlParts, "pool_max_conn_lifetime="+cfg.Pool.MaxConnLifetime)
	}
	if cfg.Pool.MaxConnIdleTime != "" {
		urlParts = append(urlParts, "pool_max_conn_idle_time="+cfg.Pool.MaxConnIdleTime)
	}
	if cfg.Pool.HealthCheckPeriod != "" {
		urlParts = append(urlParts, "pool_health_check_period="+cfg.Pool.HealthCheckPeriod)
	}
	urlParts = append(urlParts, fmt.Sprintf("pool_max_conn_lifetime_jitter=%d", cfg.Pool.MaxConnLifetimeJitter))

	return strings.Join(urlParts, " ")
}
