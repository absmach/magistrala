// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cassandra

const (
	// Table contains query for default table created in cassandra db.
	Table = `CREATE TABLE IF NOT EXISTS messages (
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
        PRIMARY KEY (publisher, time, subtopic, name)
    ) WITH CLUSTERING ORDER BY (time DESC)`

	jsonTable = `CREATE TABLE IF NOT EXISTS %s (
        id uuid,
        channel text,
        subtopic text,
        publisher text,
        protocol text,
        created bigint,
        payload text,
        PRIMARY KEY (publisher, created, subtopic)
    ) WITH CLUSTERING ORDER BY (created DESC)`
)
