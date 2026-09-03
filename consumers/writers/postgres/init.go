// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of postgres-writer.
//
// sql-migrate orders migrations lexicographically unless the id starts with a
// number, so the sequence is zero padded: without the padding "messages_10"
// would sort before "messages_2". Keep the same width when adding migrations,
// and never renumber an id that has already shipped.
//
// readers/postgres records into the same gorp_migrations table, so no id here
// may collide with one of its ids: the first service to apply an id makes the
// other silently skip its own migration under that id.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				// device_id is the device's external serial as it appeared in the
				// payload, denormalised onto the row. It is deliberately not part of
				// the primary key: SenML normalisation folds the base name into name,
				// so rows from different devices already differ there, and publisher
				// stays the audit identity. NOT NULL DEFAULT '' rather than nullable
				// so that "no device" scans into senml.Message.DeviceId (a plain
				// string) without NULL handling.
				//
				// The two indexes support the MG-15 observed-device aggregation:
				// distinct device_id values grouped for a channel+publisher (a
				// gateway's roster), and the mirror grouped by publisher for a
				// channel+device_id (a device's gateways). Plain Postgres has no
				// other secondary index on this table — every other filter here
				// relies on a full scan, since even `channel` alone isn't indexed —
				// so both directions get one rather than leaving the inverse to a
				// scan a single index would have made easy to overlook.
				Id: "messages_0001",
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
                        device_id     TEXT NOT NULL DEFAULT '',
                        PRIMARY KEY (time, publisher, subtopic, name)
                    )`,
					`CREATE INDEX IF NOT EXISTS idx_channel_publisher_device_id ON messages (channel, publisher, device_id)`,
					`CREATE INDEX IF NOT EXISTS idx_channel_device_id_publisher ON messages (channel, device_id, publisher)`,
				},
				Down: []string{
					"DROP TABLE messages",
				},
			},
		},
	}
}
