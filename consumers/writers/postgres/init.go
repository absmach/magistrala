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
                        device_id     TEXT NOT NULL DEFAULT '',
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
					`ALTER TABLE messages ADD PRIMARY KEY (time, publisher, subtopic, name, device_id)`,
				},
			},
			{
				Id: "messages_3",
				Up: []string{
					`ALTER TABLE messages ADD COLUMN IF NOT EXISTS device_id TEXT`,
					`UPDATE messages SET device_id = '' WHERE device_id IS NULL`,
					`ALTER TABLE messages ALTER COLUMN device_id SET DEFAULT ''`,
					`ALTER TABLE messages ALTER COLUMN device_id SET NOT NULL`,
					`ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_pkey`,
					`ALTER TABLE messages ADD PRIMARY KEY (time, publisher, subtopic, name, device_id)`,
				},
				Down: []string{
					`ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_pkey`,
					`ALTER TABLE messages ALTER COLUMN device_id DROP NOT NULL`,
					`ALTER TABLE messages ALTER COLUMN device_id DROP DEFAULT`,
					`ALTER TABLE messages ADD PRIMARY KEY (time, publisher, subtopic, name)`,
					`ALTER TABLE messages DROP COLUMN IF EXISTS device_id`,
				},
			},
		},
	}
}
