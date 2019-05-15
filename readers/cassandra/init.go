//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cassandra

import (
	"github.com/gocql/gocql"
)

// DBConfig contains Cassandra DB specific parameters.
type DBConfig struct {
	Hosts    []string
	Keyspace string
	Username string
	Password string
	Port     int
}

// Connect establishes connection to the Cassandra cluster.
func Connect(cfg DBConfig) (*gocql.Session, error) {
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Keyspace = cfg.Keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Authenticator = gocql.PasswordAuthenticator{
		Username: cfg.Username,
		Password: cfg.Password,
	}
	cluster.Port = cfg.Port

	return cluster.CreateSession()
}
