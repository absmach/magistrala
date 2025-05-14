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
		},
	}
}
