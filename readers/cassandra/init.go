// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra

import (
	"github.com/gocql/gocql"
)

// DBConfig contains Cassandra DB specific parameters.
type DBConfig struct {
	Hosts    []string
	Keyspace string
	User     string
	Pass     string
	Port     int
}

// Connect establishes connection to the Cassandra cluster.
func Connect(cfg DBConfig) (*gocql.Session, error) {
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Keyspace = cfg.Keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cfg.User,
		Password: cfg.Pass,
	}
	cluster.Port = cfg.Port

	return cluster.CreateSession()
}
