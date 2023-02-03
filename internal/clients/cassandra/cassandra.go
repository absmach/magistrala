// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	errConfig  = errors.New("failed to load Cassandra configuration")
	errConnect = errors.New("failed to connect to Cassandra database")
	errInit    = errors.New("failed to execute initialization query in Cassandra ")
)

// Config contains Cassandra DB specific parameters
type Config struct {
	Hosts    []string `env:"DB_CLUSTER"     envDefault:"127.0.0.1" envSeparator:","`
	Keyspace string   `env:"DB_KEYSPACE"    envDefault:"mainflux"`
	User     string   `env:"DB_USER"        envDefault:""`
	Pass     string   `env:"DB_PASS"        envDefault:""`
	Port     int      `env:"DB_PORT"        envDefault:"9042"`
}

// Setup load configuration from environment and creates new cassandra connection
func Setup(envPrefix string) (*gocql.Session, error) {
	return SetupDB(envPrefix, "")
}

// SetupDB load configuration from environment,
// creates new cassandra connection and executes
// the initial query in database.
func SetupDB(envPrefix string, initQuery string) (*gocql.Session, error) {
	config := Config{}
	if err := env.Parse(&config, env.Options{Prefix: envPrefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	cs, err := Connect(config)
	if err != nil {
		return nil, err
	}
	if initQuery != "" {
		if err := InitDB(cs, initQuery); err != nil {
			return nil, errors.Wrap(errInit, err)
		}
	}
	return cs, nil
}

// Connect establishes connection to the Cassandra cluster.
func Connect(cfg Config) (*gocql.Session, error) {
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Keyspace = cfg.Keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cfg.User,
		Password: cfg.Pass,
	}
	cluster.Port = cfg.Port

	cassSess, err := cluster.CreateSession()
	if err != nil {
		return nil, errors.Wrap(errConnect, err)
	}
	return cassSess, nil
}

func InitDB(cs *gocql.Session, query string) error {
	return cs.Query(query).Exec()
}
