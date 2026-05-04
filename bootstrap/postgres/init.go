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
			{
				Id: "configs_7",
				Up: []string{
					`ALTER TABLE IF EXISTS configs RENAME COLUMN magistrala_client TO client_id`,
					`ALTER TABLE IF EXISTS configs RENAME COLUMN magistrala_secret TO client_secret`,
					`CREATE UNIQUE INDEX IF NOT EXISTS configs_client_id_key ON configs (client_id)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS configs_client_id_domain_id_key ON configs (client_id, domain_id)`,
					`DROP TABLE IF EXISTS connections`,
					`DROP TABLE IF EXISTS channels`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS configs RENAME COLUMN client_id TO magistrala_client`,
					`ALTER TABLE IF EXISTS configs RENAME COLUMN client_secret TO magistrala_secret`,
				},
			},
			{
				Id: "configs_8",
				Up: []string{
					`DO $$
					BEGIN
						IF EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'configs' AND column_name = 'client_id'
						) AND NOT EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'configs' AND column_name = 'id'
						) THEN
							ALTER TABLE configs RENAME COLUMN client_id TO id;
						END IF;
					END $$`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS client_secret`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS client_secret TEXT`,
					`DO $$
					BEGIN
						IF EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'configs' AND column_name = 'id'
						) AND NOT EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'configs' AND column_name = 'client_id'
						) THEN
							ALTER TABLE configs RENAME COLUMN id TO client_id;
						END IF;
					END $$`,
				},
			},
			{
				Id: "configs_10",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS profiles (
							id               VARCHAR(36) PRIMARY KEY,
							domain_id        VARCHAR(36) NOT NULL,
							name             VARCHAR(1024) NOT NULL,
							description      TEXT,
							template_format  VARCHAR(64) NOT NULL DEFAULT 'go-template',
							content_template TEXT,
							defaults         JSONB,
							binding_slots    JSONB,
							version          INT NOT NULL DEFAULT 1,
							created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
							updated_at       TIMESTAMP NOT NULL DEFAULT NOW(),
							UNIQUE (domain_id, name)
						)`,
					`CREATE INDEX IF NOT EXISTS idx_profiles_domain_id ON profiles (domain_id)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS profiles`,
				},
			},
			{
				Id: "configs_11",
				Up: []string{
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS profile_id VARCHAR(36) REFERENCES profiles (id) ON DELETE SET NULL`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS render_context JSONB`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS render_context`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS profile_id`,
				},
			},
			{
				Id: "configs_12",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS bindings (
							config_id       TEXT NOT NULL,
							slot            VARCHAR(256) NOT NULL,
							type            VARCHAR(64) NOT NULL,
							resource_id     TEXT NOT NULL,
							snapshot        JSONB,
							secret_snapshot BYTEA,
							updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
							PRIMARY KEY (config_id, slot)
						)`,
					`CREATE INDEX IF NOT EXISTS idx_bindings_config_id ON bindings (config_id)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS bindings`,
				},
			},
			{
				Id: "configs_13",
				Up: []string{
					`DO $$
					BEGIN
						IF EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'configs' AND column_name = 'state'
						) AND NOT EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'configs' AND column_name = 'status'
						) THEN
							ALTER TABLE configs RENAME COLUMN state TO status;
						END IF;
					END $$`,
				},
				Down: []string{
					`DO $$
					BEGIN
						IF EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'configs' AND column_name = 'status'
						) AND NOT EXISTS (
							SELECT 1
							FROM information_schema.columns
							WHERE table_name = 'configs' AND column_name = 'state'
						) THEN
							ALTER TABLE configs RENAME COLUMN status TO state;
						END IF;
					END $$`,
				},
			},
			{
				Id: "configs_14",
				Up: []string{
					`DO $$
						BEGIN
							IF EXISTS (
								SELECT 1
								FROM information_schema.tables
								WHERE table_name = 'binding_snapshots'
							) AND NOT EXISTS (
								SELECT 1
								FROM information_schema.tables
								WHERE table_name = 'bindings'
							) THEN
								ALTER TABLE binding_snapshots RENAME TO bindings;
							END IF;
						END $$`,
					`DO $$
						BEGIN
							IF EXISTS (
								SELECT 1
								FROM pg_class
								WHERE relname = 'idx_binding_snapshots_config_id'
							) AND NOT EXISTS (
								SELECT 1
								FROM pg_class
								WHERE relname = 'idx_bindings_config_id'
							) THEN
								ALTER INDEX idx_binding_snapshots_config_id RENAME TO idx_bindings_config_id;
							END IF;
						END $$`,
				},
				Down: []string{
					`DO $$
						BEGIN
							IF EXISTS (
								SELECT 1
								FROM information_schema.tables
								WHERE table_name = 'bindings'
							) AND NOT EXISTS (
								SELECT 1
								FROM information_schema.tables
								WHERE table_name = 'binding_snapshots'
							) THEN
								ALTER TABLE bindings RENAME TO binding_snapshots;
							END IF;
						END $$`,
					`DO $$
						BEGIN
							IF EXISTS (
								SELECT 1
								FROM pg_class
								WHERE relname = 'idx_bindings_config_id'
							) AND NOT EXISTS (
								SELECT 1
								FROM pg_class
								WHERE relname = 'idx_binding_snapshots_config_id'
							) THEN
								ALTER INDEX idx_bindings_config_id RENAME TO idx_binding_snapshots_config_id;
							END IF;
						END $$`,
				},
			},
			{
				Id: "configs_15",
				Up: []string{
					`ALTER TABLE IF EXISTS profiles ADD COLUMN IF NOT EXISTS binding_slots JSONB`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS profiles DROP COLUMN IF EXISTS binding_slots`,
				},
			},
		},
	}
}
