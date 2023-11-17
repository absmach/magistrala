// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import migrate "github.com/rubenv/sql-migrate"

// Migration of timescale-writer.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "messages_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS messages (
                        time BIGINT NOT NULL,
                        channel       UUID,
                        subtopic      VARCHAR(254),
                        publisher     UUID,
                        protocol      TEXT,
                        name          VARCHAR(254),
                        unit          TEXT,
                        value         FLOAT,
                        string_value  TEXT,
                        bool_value    BOOL,
                        data_value    BYTEA,
                        sum           FLOAT,
                        update_time   FLOAT,
                        PRIMARY KEY (time, publisher, subtopic, name)
                    );
                    SELECT create_hypertable('messages', 'time', create_default_indexes => FALSE, chunk_time_interval => 86400000, if_not_exists => TRUE);`,
				},
				Down: []string{
					"DROP TABLE messages",
				},
			},
		},
	}
}
