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

// Connect establishes connection to the Cassandra cluster.
func Connect(hosts []string, keyspace string) (*gocql.Session, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	if err := session.Query(table).Exec(); err != nil {
		return nil, err
	}

	return session, nil
}
