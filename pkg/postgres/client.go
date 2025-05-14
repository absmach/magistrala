// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

var (
	errMigration               = errors.New("failed to apply migrations")
	errInvalidConnectionString = errors.New("invalid connection string")
)

type PoolConfig struct {
	// MaxConnLifetime is the duration since creation after which a connection will be automatically closed.
	MaxConnLifetime time.Duration `env:"MAX_CONN_LIFETIME" envDefault:"1h"`

	// pool_max_conn_lifetime_jitter
	MaxConnLifetimeJitter time.Duration `env:"MAX_CONN_LIFETIME_JITTER" envDefault:"0"`

	// MaxConnIdleTime is the duration after which an idle connection will be automatically closed by the health check.
	MaxConnIdleTime time.Duration `env:"MAX_CONN_IDLE_TIME" envDefault:"15m"`

	// MaxConnLifetime is the duration since creation after which a connection will be automatically closed.
	MaxConns uint16 `env:"MAX_CONNS" envDefault:"5"`

	// MinConns is the minimum size of the pool. After connection closes, the pool might dip below MinConns. A low
	// number of MinConns might mean the pool is empty after MaxConnLifetime until the health check has a chance
	// to create new connections.
	MinConns uint16 `env:"MIN_CONNS" envDefault:"1"`

	// MinIdleConns is the minimum number of idle connections in the pool. You can increase this to ensure that
	// there are always idle connections available. This can help reduce tail latencies during request processing,
	// as you can avoid the latency of establishing a new connection while handling requests. It is superior
	// to MinConns for this purpose.
	// Similar to MinConns, the pool might temporarily dip below MinIdleConns after connection closes.
	MinIdleConns uint16 `env:"MIN_IDLE_CONNS" envDefault:"1"`

	// HealthCheckPeriod is the duration between checks of the health of idle connections.
	HealthCheckPeriod time.Duration `env:"HEALTH_CHECK_PERIOD" envDefault:"1m"`
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
	Pool        PoolConfig `envPrefix:"POOL_"`
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
	pgxPoolConfig, err := pgxpool.ParseConfig(cfg.dbConnURL())
	if err != nil {
		return nil, errors.Wrap(errInvalidConnectionString, err)
	}

	pgxPoolConfig.MaxConnIdleTime = cfg.Pool.MaxConnIdleTime
	pgxPoolConfig.MaxConnLifetimeJitter = cfg.Pool.MaxConnLifetimeJitter
	pgxPoolConfig.MaxConnLifetime = cfg.Pool.MaxConnLifetime
	pgxPoolConfig.MaxConns = int32(cfg.Pool.MaxConns)
	pgxPoolConfig.MinConns = int32(cfg.Pool.MinConns)
	pgxPoolConfig.MinIdleConns = int32(cfg.Pool.MinIdleConns)
	pgxPoolConfig.HealthCheckPeriod = cfg.Pool.HealthCheckPeriod

	dbpool, err := pgxpool.NewWithConfig(context.Background(), pgxPoolConfig)
	if err != nil {
		return nil, err
	}

	sqlDB := stdlib.OpenDBFromPool(dbpool)

	return sqlx.NewDb(sqlDB, "pgx"), nil
}

func (cfg Config) dbConnURL() string {
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
	return strings.Join(urlParts, " ")
}
