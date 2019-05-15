//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cassandra

import "github.com/gocql/gocql"

const table = `CREATE TABLE IF NOT EXISTS messages (
        id uuid,
        channel text,
        subtopic text,
    	publisher text,
        protocol text,
    	name text,
    	unit text,
    	value double,
    	string_value text,
        bool_value boolean,
        data_value text,
    	value_sum double,
    	time double,
    	update_time double,
    	link text,
        PRIMARY KEY (channel, time, id)
	) WITH CLUSTERING ORDER BY (time DESC)`

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

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	if err := session.Query(table).Exec(); err != nil {
		return nil, err
	}

	return session, nil
}
