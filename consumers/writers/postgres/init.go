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
			{
				// Supports the MG-15 observed-device aggregation: distinct
				// device_id values grouped for a channel+publisher (a
				// gateway's roster), and the mirror grouped by publisher for
				// a channel+device_id (a device's gateways). Plain Postgres
				// has had no secondary index on this table until now — every
				// other filter here already relies on a full scan, since
				// even `channel` alone isn't indexed — so both directions
				// get one rather than leaving the inverse to a scan a new
				// index elsewhere would have made easy to overlook.
				Id: "messages_4",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_channel_publisher_device_id ON messages (channel, publisher, device_id)`,
					`CREATE INDEX IF NOT EXISTS idx_channel_device_id_publisher ON messages (channel, device_id, publisher)`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_channel_publisher_device_id`,
					`DROP INDEX IF EXISTS idx_channel_device_id_publisher`,
				},
			},
		},
	}
}
