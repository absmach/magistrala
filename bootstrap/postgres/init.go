// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of bootstrap service.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "configs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS configs (
						mainflux_client TEXT UNIQUE NOT NULL,
						owner          VARCHAR(254),
						name           TEXT,
						mainflux_key   CHAR(36) UNIQUE NOT NULL,
						external_id    TEXT UNIQUE NOT NULL,
						external_key   TEXT NOT NULL,
						content  	   TEXT,
						client_cert	   TEXT,
						client_key 	   TEXT,
						ca_cert 	   TEXT,
						state          BIGINT NOT NULL,
						PRIMARY KEY (mainflux_client, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS unknown_configs (
						external_id  TEXT UNIQUE NOT NULL,
						external_key TEXT NOT NULL,
						PRIMARY KEY (external_id, external_key)
					)`,
					`CREATE TABLE IF NOT EXISTS channels (
						mainflux_channel TEXT UNIQUE NOT NULL,
						owner    		 VARCHAR(254),
						name     		 TEXT,
						metadata 		 JSON,
						PRIMARY KEY (mainflux_channel, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    TEXT,
						channel_owner VARCHAR(256),
						config_id     TEXT,
						config_owner  VARCHAR(256),
						FOREIGN KEY (channel_id, channel_owner) REFERENCES channels (mainflux_channel, owner) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (config_id, config_owner) REFERENCES configs (mainflux_client, owner) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (channel_id, channel_owner, config_id, config_owner)
					)`,
				},
				Down: []string{
					"DROP TABLE connections",
					"DROP TABLE configs",
					"DROP TABLE channels",
					"DROP TABLE unknown_configs",
				},
			},
			{
				Id: "configs_2",
				Up: []string{
					"DROP TABLE IF EXISTS unknown_configs",
				},
				Down: []string{
					"CREATE TABLE IF NOT EXISTS unknown_configs",
				},
			},
			{
				Id: "configs_3",
				Up: []string{
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_id VARCHAR(36)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS description VARCHAR(1024)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS created_at TIMESTAMP`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS updated_by VARCHAR(254)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS status SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0)`,
				},
			},
			{
				Id: "configs_4",
				Up: []string{
					`ALTER TABLE IF EXISTS configs RENAME COLUMN mainflux_client TO magistrala_client`,
					`ALTER TABLE IF EXISTS configs RENAME COLUMN mainflux_key TO magistrala_secret`,
					`ALTER TABLE IF EXISTS channels RENAME COLUMN mainflux_channel TO magistrala_channel`,
				},
			},
			{
				Id: "configs_5",
				Up: []string{
					`ALTER TABLE IF EXISTS configs RENAME COLUMN owner TO domain_id`,
					`ALTER TABLE IF EXISTS channels RENAME COLUMN owner TO domain_id`,
					`ALTER TABLE IF EXISTS configs ADD CONSTRAINT configs_name_domain_id_key UNIQUE (name, domain_id)`,
				},
			},
			{
				Id: "configs_6",
				Up: []string{
					`ALTER TABLE IF EXISTS connections DROP CONSTRAINT IF EXISTS connections_pkey`,
					`ALTER TABLE IF EXISTS connections DROP COLUMN IF EXISTS channel_owner`,
					`ALTER TABLE IF EXISTS connections DROP COLUMN IF EXISTS config_owner`,
					`ALTER TABLE IF EXISTS connections ADD COLUMN IF NOT EXISTS domain_id VARCHAR(256) NOT NULL`,
					`ALTER TABLE IF EXISTS connections ADD CONSTRAINT connections_pkey PRIMARY KEY (channel_id, config_id, domain_id)`,
					`ALTER TABLE IF EXISTS connections ADD FOREIGN KEY (channel_id, domain_id) REFERENCES channels (magistrala_channel, domain_id) ON DELETE CASCADE ON UPDATE CASCADE`,
					`ALTER TABLE IF EXISTS connections ADD FOREIGN KEY (config_id, domain_id) REFERENCES configs (magistrala_client, domain_id) ON DELETE CASCADE ON UPDATE CASCADE`,
				},
			},
		},
	}
}
