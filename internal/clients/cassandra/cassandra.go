// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/caarlos0/env/v10"
	"github.com/gocql/gocql"
)

var (
	errConfig  = errors.New("failed to load Cassandra configuration")
	errConnect = errors.New("failed to connect to Cassandra database")
	errInit    = errors.New("failed to execute initialization query in Cassandra ")
)

// Config contains Cassandra DB specific parameters.
type Config struct {
	Hosts    []string `env:"CLUSTER"     envDefault:"127.0.0.1" envSeparator:","`
	Keyspace string   `env:"KEYSPACE"    envDefault:"magistrala"`
	User     string   `env:"USER"        envDefault:""`
	Pass     string   `env:"PASS"        envDefault:""`
	Port     int      `env:"PORT"        envDefault:"9042"`
}

// Setup load configuration from environment and creates new cassandra connection.
func Setup(envPrefix string) (*gocql.Session, error) {
	return SetupDB(envPrefix, "")
}

// SetupDB load configuration from environment,
// creates new cassandra connection and executes
// the initial query in database.
func SetupDB(envPrefix, initQuery string) (*gocql.Session, error) {
	cfg := Config{}
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: envPrefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	cs, err := Connect(cfg)
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
