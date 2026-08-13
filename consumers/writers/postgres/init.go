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
			{
				// device_id is the device's external serial as it appeared in the
				// payload, denormalised onto the row. It is deliberately not part of
				// the primary key: SenML normalisation folds the base name into name,
				// so rows from different devices already differ there, and publisher
				// stays the audit identity. NOT NULL DEFAULT '' rather than nullable
				// so that "no device" scans into senml.Message.DeviceId (a plain
				// string) without NULL handling, and so existing rows migrate without
				// a table rewrite.
				Id: "messages_3",
				Up: []string{
					`ALTER TABLE messages ADD COLUMN IF NOT EXISTS device_id TEXT NOT NULL DEFAULT ''`,
				},
				Down: []string{
					`ALTER TABLE messages DROP COLUMN IF EXISTS device_id`,
				},
			},
		},
	}
}
