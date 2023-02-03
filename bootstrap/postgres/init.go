// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of bootstrap service
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "configs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS configs (
						mainflux_thing TEXT UNIQUE NOT NULL,
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
						PRIMARY KEY (mainflux_thing, owner)
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
						FOREIGN KEY (config_id, config_owner) REFERENCES configs (mainflux_thing, owner) ON DELETE CASCADE ON UPDATE CASCADE,
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
		},
	}
}
