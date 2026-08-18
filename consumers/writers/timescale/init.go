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
                        PRIMARY KEY (time, channel, subtopic, protocol, publisher, name)
                    );`,

					// Creating HyperTable with chunks interval of 1 day = 86400000000000 Nanoseconds
					"SELECT create_hypertable('messages', by_range('time', 86400000000000 ), if_not_exists => TRUE, migrate_data => TRUE);",
				},
				Down: []string{
					"DROP TABLE messages",
				},
			},
			{
				Id: "messages_2",
				Up: []string{
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
				},
				DisableTransactionUp: true,
				Down: []string{
					"DROP INDEX IF EXISTS idx_channel_time ;",

					"DROP INDEX IF EXISTS idx_channel_name_time ;",

					"DROP INDEX IF EXISTS idx_channel_subtopic_name_time ;",

					"DROP INDEX IF EXISTS idx_channel_publisher_name_time ;",

					"DROP INDEX IF EXISTS idx_channel_subtopic_publisher_name_time ;",
				},
			},
			{
				// device_id is the device's external serial as it appeared in the
				// payload, denormalised onto the row. It is deliberately kept out of
				// the primary key: changing the key of a populated hypertable is the
				// expensive, rewrite-prone operation, and it buys nothing here since
				// SenML normalisation folds the base name into name, so rows from
				// different devices already differ in the existing key. publisher
				// remains the audit identity. NOT NULL DEFAULT '' rather than
				// nullable so "no device" scans into senml.Message.DeviceId (a plain
				// string) without NULL handling; a constant default makes the ALTER
				// metadata-only on PostgreSQL 11+, so existing chunks are not
				// rewritten. The index follows the convention of the ones above:
				// leading channel, trailing name, time DESC.
				Id: "messages_3",
				Up: []string{
					"ALTER TABLE messages ADD COLUMN IF NOT EXISTS device_id TEXT NOT NULL DEFAULT '';",

					// Index on channel, device_id, name, time
					"CREATE INDEX IF NOT EXISTS idx_channel_device_id_name_time  ON messages (channel, device_id, name, time DESC) WITH (timescaledb.transaction_per_chunk);",
				},
				DisableTransactionUp: true,
				Down: []string{
					"DROP INDEX IF EXISTS idx_channel_device_id_name_time ;",

					"ALTER TABLE messages DROP COLUMN IF EXISTS device_id;",
				},
			},
			{
				// Supports the MG-15 observed-device aggregation: distinct
				// device_id values grouped for a channel+publisher (a
				// gateway's roster). Without a (channel, publisher, ...)
				// index this GROUP BY falls back to a full partition scan,
				// which is exactly the cost this index exists to avoid.
				// Trailing time DESC follows the convention of the indexes
				// above, letting MAX(time) come from the index too.
				//
				// The mirror direction — distinct publisher values grouped
				// for a channel+device_id — reuses idx_channel_device_id_name_time
				// from messages_3 above: its leading (channel, device_id)
				// pair already gives that query an index scan instead of a
				// full partition scan, so it does not need a dedicated
				// index of its own.
				Id: "messages_4",
				Up: []string{
					"CREATE INDEX IF NOT EXISTS idx_channel_publisher_device_id_time ON messages (channel, publisher, device_id, time DESC) WITH (timescaledb.transaction_per_chunk);",
				},
				DisableTransactionUp: true,
				Down: []string{
					"DROP INDEX IF EXISTS idx_channel_publisher_device_id_time ;",
				},
			},
		},
	}
}
