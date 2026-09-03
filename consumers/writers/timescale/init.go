// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale

import migrate "github.com/rubenv/sql-migrate"

// Migration of timescale-writer.
//
// sql-migrate orders migrations lexicographically unless the id starts with a
// number, so the sequence is zero padded: without the padding "messages_10"
// would sort before "messages_2". Keep the same width when adding migrations,
// and never renumber an id that has already shipped.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				// device_id is the device's external serial as it appeared in the
				// payload, denormalised onto the row. It is deliberately kept out of
				// the primary key: changing the key of a populated hypertable is the
				// expensive, rewrite-prone operation, and it buys nothing here since
				// SenML normalisation folds the base name into name, so rows from
				// different devices already differ in the existing key. publisher
				// remains the audit identity. NOT NULL DEFAULT '' rather than
				// nullable so "no device" scans into senml.Message.DeviceId (a plain
				// string) without NULL handling.
				//
				// idx_channel_publisher_device_id_time supports the MG-15
				// observed-device aggregation: distinct device_id values grouped for
				// a channel+publisher (a gateway's roster). The mirror direction --
				// distinct publisher values grouped for a channel+device_id -- reuses
				// idx_channel_device_id_name_time, whose leading (channel, device_id)
				// pair already gives that query an index scan instead of a full
				// partition scan, so it needs no index of its own.
				//
				// Every index carries trailing time DESC so that MAX(time) can come
				// from the index too.
				Id: "messages_0001",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS messages (
                        time BIGINT NOT NULL,
                        channel       UUID,
                        subtopic      VARCHAR(254),
                        publisher     VARCHAR(254),
                        protocol      TEXT,
                        name          VARCHAR(254),
                        unit          TEXT,
                        value         FLOAT,
                        string_value  TEXT,
                        bool_value    BOOL,
                        data_value    BYTEA,
                        sum           FLOAT,
                        update_time   FLOAT,
                        device_id     TEXT NOT NULL DEFAULT '',
                        PRIMARY KEY (time, channel, subtopic, protocol, publisher, name)
                    );`,

					// Creating HyperTable with chunks interval of 1 day = 86400000000000 Nanoseconds
					"SELECT create_hypertable('messages', by_range('time', 86400000000000 ), if_not_exists => TRUE, migrate_data => TRUE);",

					// Index on channel, time
					"CREATE INDEX IF NOT EXISTS idx_channel_time  ON messages (channel, time DESC) WITH (timescaledb.transaction_per_chunk);",

					// Index on channel, name, time
					"CREATE INDEX IF NOT EXISTS idx_channel_name_time  ON messages (channel, name, time DESC) WITH (timescaledb.transaction_per_chunk);",

					// Index on channel, subtopic, name, time
					"CREATE INDEX IF NOT EXISTS idx_channel_subtopic_name_time  ON messages (channel, subtopic, name, time DESC) WITH (timescaledb.transaction_per_chunk);",

					// Index on channel, publisher, name, time
					"CREATE INDEX IF NOT EXISTS idx_channel_publisher_name_time  ON messages (channel, publisher, name, time DESC) WITH (timescaledb.transaction_per_chunk);",

					// Index on channel, subtopic, publisher, name, time
					"CREATE INDEX IF NOT EXISTS idx_channel_subtopic_publisher_name_time  ON messages (channel, subtopic, publisher, name, time DESC) WITH (timescaledb.transaction_per_chunk);",

					// Index on channel, device_id, name, time
					"CREATE INDEX IF NOT EXISTS idx_channel_device_id_name_time  ON messages (channel, device_id, name, time DESC) WITH (timescaledb.transaction_per_chunk);",

					// Index on channel, publisher, device_id, time
					"CREATE INDEX IF NOT EXISTS idx_channel_publisher_device_id_time ON messages (channel, publisher, device_id, time DESC) WITH (timescaledb.transaction_per_chunk);",
				},
				DisableTransactionUp: true,
				Down: []string{
					"DROP TABLE messages",
				},
			},
		},
	}
}
