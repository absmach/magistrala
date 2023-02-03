// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of Things service
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "things_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS things (
						id       UUID,
						owner    VARCHAR(254),
						key      VARCHAR(4096) UNIQUE NOT NULL,
						name     VARCHAR(1024),
						metadata JSON,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS channels (
						id       UUID,
						owner    VARCHAR(254),
						name     VARCHAR(1024),
						metadata JSON,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    UUID,
						channel_owner VARCHAR(254),
						thing_id      UUID,
						thing_owner   VARCHAR(254),
						FOREIGN KEY (channel_id, channel_owner) REFERENCES channels (id, owner) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (thing_id, thing_owner) REFERENCES things (id, owner) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (channel_id, channel_owner, thing_id, thing_owner)
					)`,
				},
				Down: []string{
					"DROP TABLE connections",
					"DROP TABLE things",
					"DROP TABLE channels",
				},
			},
			{
				Id: "things_2",
				Up: []string{
					`ALTER TABLE IF EXISTS things ALTER COLUMN
					 metadata TYPE JSONB using metadata::text::jsonb`,
				},
			},
			{
				Id: "things_3",
				Up: []string{
					`ALTER TABLE IF EXISTS channels ALTER COLUMN
					 metadata TYPE JSONB using metadata::text::jsonb`,
				},
			},
			{
				Id: "things_4",
				Up: []string{
					`ALTER TABLE IF EXISTS things ADD CONSTRAINT things_id_key UNIQUE (id)`,
				},
			},
		},
	}

}
