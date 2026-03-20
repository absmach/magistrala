// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of postgres-writer.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "messages_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS messages (
                        id            UUID,
                        channel       UUID,
                        subtopic      VARCHAR(254),
                        publisher     UUID,
                        protocol      TEXT,
                        name          TEXT,
                        unit          TEXT,
                        value         FLOAT,
                        string_value  TEXT,
                        bool_value    BOOL,
                        data_value    BYTEA,
                        sum           FLOAT,
                        time          FLOAT,
                        update_time   FLOAT,
                        PRIMARY KEY (id)
                    )`,
				},
				Down: []string{
					"DROP TABLE messages",
				},
			},
			{
				Id: "messages_2",
				Up: []string{
					`ALTER TABLE messages DROP CONSTRAINT messages_pkey`,
					`ALTER TABLE messages ADD PRIMARY KEY (time, publisher, subtopic, name)`,
				},
			},
		},
	}
}
