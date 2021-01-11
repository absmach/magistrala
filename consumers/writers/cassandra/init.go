// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cassandra

import "github.com/gocql/gocql"

const (
	table = `CREATE TABLE IF NOT EXISTS messages (
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
        data_value blob,
        sum double,
        time double,
        update_time double,
        PRIMARY KEY (channel, time, id)
    ) WITH CLUSTERING ORDER BY (time DESC)`

	jsonTable = `CREATE TABLE IF NOT EXISTS %s (
        id uuid,
        channel text,
        subtopic text,
        publisher text,
        protocol text,
        created bigint,
        payload text,
        PRIMARY KEY (channel, created, id)
    ) WITH CLUSTERING ORDER BY (created DESC)`
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

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	if err := session.Query(table).Exec(); err != nil {
		return nil, err
	}

	return session, nil
}
