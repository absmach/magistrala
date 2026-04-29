// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of bootstrap service.
func Migration() (*migrate.MemoryMigrationSource, error) {
	bootstrapMigration := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "configs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS configs (
						mainflux_client TEXT UNIQUE NOT NULL,
						owner           VARCHAR(254),
						name            TEXT,
						mainflux_key    CHAR(36) UNIQUE NOT NULL,
						external_id     TEXT UNIQUE NOT NULL,
						external_key    TEXT NOT NULL,
						content         TEXT,
						client_cert     TEXT,
						client_key      TEXT,
						ca_cert         TEXT,
						state           BIGINT NOT NULL,
						PRIMARY KEY (mainflux_client, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS unknown_configs (
						external_id  TEXT UNIQUE NOT NULL,
						external_key TEXT NOT NULL,
						PRIMARY KEY (external_id, external_key)
					)`,
					`CREATE TABLE IF NOT EXISTS channels (
						mainflux_channel TEXT UNIQUE NOT NULL,
						owner            VARCHAR(254),
						name             TEXT,
						metadata         JSON,
						PRIMARY KEY (mainflux_channel, owner)
					)`,
				},
				Down: []string{
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
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS client_id TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS client_secret CHAR(36)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS id VARCHAR(36)`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS mainflux_client TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS mainflux_key CHAR(36)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS mainflux_channel TEXT`,
					`UPDATE configs SET client_id = mainflux_client WHERE client_id IS NULL AND mainflux_client IS NOT NULL`,
					`UPDATE configs SET client_secret = mainflux_key WHERE client_secret IS NULL AND mainflux_key IS NOT NULL`,
					`UPDATE channels SET id = mainflux_channel WHERE id IS NULL AND mainflux_channel IS NOT NULL`,
				},
			},
			{
				Id: "configs_5",
				Up: []string{
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS domain_id VARCHAR(256)`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS owner VARCHAR(254)`,
					`UPDATE configs SET domain_id = owner WHERE domain_id IS NULL AND owner IS NOT NULL`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS domain_id VARCHAR(256)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS owner VARCHAR(254)`,
					`UPDATE channels SET domain_id = owner WHERE domain_id IS NULL AND owner IS NOT NULL`,
					`CREATE UNIQUE INDEX IF NOT EXISTS configs_name_domain_id_key ON configs (name, domain_id)`,
				},
			},
			{
				Id: "configs_6",
				Up: []string{
					`ALTER TABLE IF EXISTS channels ALTER COLUMN metadata TYPE JSONB USING metadata::jsonb`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS id VARCHAR(36)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS magistrala_channel TEXT`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS mainflux_channel TEXT`,
					`UPDATE channels SET id = magistrala_channel WHERE id IS NULL AND magistrala_channel IS NOT NULL`,
					`UPDATE channels SET id = mainflux_channel WHERE id IS NULL AND mainflux_channel IS NOT NULL`,
					`ALTER TABLE IF EXISTS channels ALTER COLUMN id SET NOT NULL`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_group_id VARCHAR(36)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_id VARCHAR(36)`,
					`UPDATE channels SET parent_group_id = parent_id WHERE parent_group_id IS NULL AND parent_id IS NOT NULL`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS tags TEXT[]`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS created_by VARCHAR(254)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS channels_id_key ON channels (id)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS channels_id_domain_id_key ON channels (id, domain_id)`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS client_id TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS client_secret CHAR(36)`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS magistrala_client TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS magistrala_secret TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS mainflux_client TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS mainflux_key TEXT`,
					`UPDATE configs SET client_id = magistrala_client WHERE client_id IS NULL AND magistrala_client IS NOT NULL`,
					`UPDATE configs SET client_secret = magistrala_secret WHERE client_secret IS NULL AND magistrala_secret IS NOT NULL`,
					`UPDATE configs SET client_id = mainflux_client WHERE client_id IS NULL AND mainflux_client IS NOT NULL`,
					`UPDATE configs SET client_secret = mainflux_key WHERE client_secret IS NULL AND mainflux_key IS NOT NULL`,
					`DELETE FROM configs WHERE client_id IS NULL OR client_id = ''`,
					`DELETE FROM channels WHERE id IS NULL OR id = ''`,
					`ALTER TABLE IF EXISTS configs ALTER COLUMN client_id SET NOT NULL`,
					`ALTER TABLE IF EXISTS configs ALTER COLUMN client_secret SET NOT NULL`,
					`CREATE UNIQUE INDEX IF NOT EXISTS configs_client_id_key ON configs (client_id)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS configs_client_id_domain_id_key ON configs (client_id, domain_id)`,
					`CREATE TABLE IF NOT EXISTS config_channels (
						channel_id TEXT NOT NULL,
						config_id  TEXT NOT NULL,
						domain_id  VARCHAR(256) NOT NULL DEFAULT '',
						PRIMARY KEY (channel_id, config_id, domain_id),
						FOREIGN KEY (channel_id, domain_id) REFERENCES channels (id, domain_id) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (config_id, domain_id) REFERENCES configs (client_id, domain_id) ON DELETE CASCADE ON UPDATE CASCADE
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    VARCHAR(36),
						domain_id     VARCHAR(256),
						client_id     VARCHAR(36),
						type          SMALLINT,
						config_id     TEXT,
						channel_owner VARCHAR(256),
						config_owner  VARCHAR(256)
					)`,
					`ALTER TABLE IF EXISTS connections ADD COLUMN IF NOT EXISTS channel_id VARCHAR(36)`,
					`ALTER TABLE IF EXISTS connections ADD COLUMN IF NOT EXISTS domain_id VARCHAR(256)`,
					`ALTER TABLE IF EXISTS connections ADD COLUMN IF NOT EXISTS client_id VARCHAR(36)`,
					`ALTER TABLE IF EXISTS connections ADD COLUMN IF NOT EXISTS type SMALLINT`,
					`ALTER TABLE IF EXISTS connections ADD COLUMN IF NOT EXISTS config_id TEXT`,
					`ALTER TABLE IF EXISTS connections ADD COLUMN IF NOT EXISTS channel_owner VARCHAR(256)`,
					`ALTER TABLE IF EXISTS connections ADD COLUMN IF NOT EXISTS config_owner VARCHAR(256)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS connections_pkey
						ON connections (channel_id, domain_id, client_id, type)
						WHERE client_id IS NOT NULL AND type IS NOT NULL`,
					`INSERT INTO config_channels (channel_id, config_id, domain_id)
						SELECT channel_id, config_id, COALESCE(config_owner, channel_owner, domain_id, '')
						FROM connections
						WHERE channel_id IS NOT NULL AND config_id IS NOT NULL
						AND COALESCE(config_owner, channel_owner, domain_id, '') != ''
						ON CONFLICT (channel_id, config_id, domain_id) DO NOTHING`,
					`CREATE TABLE IF NOT EXISTS service_connections (
						channel_id TEXT NOT NULL,
						client_id  TEXT NOT NULL,
						domain_id  VARCHAR(256) NOT NULL DEFAULT '',
						conn_type  TEXT NOT NULL,
						PRIMARY KEY (channel_id, client_id, domain_id, conn_type)
					)`,
					`CREATE INDEX IF NOT EXISTS idx_service_connections_client_domain
						ON service_connections (client_id, domain_id)`,
				},
			},
			{
				Id: "configs_7",
				Up: []string{
					`DROP TABLE IF EXISTS service_connections`,
					`CREATE TABLE IF NOT EXISTS clients (
						id			       VARCHAR(36) PRIMARY KEY,
						name		       VARCHAR(1024),
						domain_id	       VARCHAR(36) NOT NULL,
						parent_group_id    VARCHAR(36) DEFAULT NULL,
						identity	       VARCHAR(254),
						secret		       VARCHAR(4096) NOT NULL,
						tags		       TEXT[],
						metadata	       JSONB,
						created_at	       TIMESTAMP,
						updated_at	       TIMESTAMP,
						updated_by         VARCHAR(254),
						status		       SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE		       (domain_id, secret),
						UNIQUE		       (domain_id, name),
						UNIQUE		       (domain_id, id)
					)`,
					`ALTER TABLE IF EXISTS clients ADD COLUMN IF NOT EXISTS name VARCHAR(1024)`,
					`ALTER TABLE IF EXISTS clients ADD COLUMN IF NOT EXISTS domain_id VARCHAR(36) NOT NULL DEFAULT ''`,
					`ALTER TABLE IF EXISTS clients ADD COLUMN IF NOT EXISTS parent_group_id VARCHAR(36) DEFAULT NULL`,
					`ALTER TABLE IF EXISTS clients ADD COLUMN IF NOT EXISTS identity VARCHAR(254)`,
					`ALTER TABLE IF EXISTS clients ADD COLUMN IF NOT EXISTS tags TEXT[]`,
					`ALTER TABLE IF EXISTS clients ADD COLUMN IF NOT EXISTS metadata JSONB`,
					`ALTER TABLE IF EXISTS clients ADD COLUMN IF NOT EXISTS updated_by VARCHAR(254)`,
					`ALTER TABLE IF EXISTS clients ADD COLUMN IF NOT EXISTS status SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS clients_domain_id_secret_key ON clients (domain_id, secret)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS clients_domain_id_id_key ON clients (domain_id, id)`,
				},
				Down: []string{
					`CREATE TABLE IF NOT EXISTS service_connections (
						channel_id TEXT NOT NULL,
						client_id  TEXT NOT NULL,
						domain_id  VARCHAR(256) NOT NULL DEFAULT '',
						conn_type  TEXT NOT NULL,
						PRIMARY KEY (channel_id, client_id, domain_id, conn_type)
					)`,
					`CREATE INDEX IF NOT EXISTS idx_service_connections_client_domain
						ON service_connections (client_id, domain_id)`,
					`CREATE TABLE IF NOT EXISTS clients (
						id         TEXT PRIMARY KEY,
						domain_id  VARCHAR(256) NOT NULL DEFAULT '',
						secret     TEXT NOT NULL,
						status     SMALLINT NOT NULL DEFAULT 0,
						created_at TIMESTAMP NOT NULL,
						updated_at TIMESTAMP NOT NULL
					)`,
				},
			},
			{
				Id: "configs_8",
				Up: []string{
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS magistrala_client TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS magistrala_secret TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS mainflux_client TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS mainflux_key TEXT`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS magistrala_channel TEXT`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS mainflux_channel TEXT`,
					`UPDATE configs SET client_id = magistrala_client WHERE (client_id IS NULL OR client_id = '') AND magistrala_client IS NOT NULL`,
					`UPDATE configs SET client_secret = magistrala_secret WHERE (client_secret IS NULL OR client_secret = '') AND magistrala_secret IS NOT NULL`,
					`UPDATE configs SET client_id = mainflux_client WHERE (client_id IS NULL OR client_id = '') AND mainflux_client IS NOT NULL`,
					`UPDATE configs SET client_secret = mainflux_key WHERE (client_secret IS NULL OR client_secret = '') AND mainflux_key IS NOT NULL`,
					`UPDATE channels SET id = magistrala_channel WHERE (id IS NULL OR id = '') AND magistrala_channel IS NOT NULL`,
					`UPDATE channels SET id = mainflux_channel WHERE (id IS NULL OR id = '') AND mainflux_channel IS NOT NULL`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS magistrala_client CASCADE`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS magistrala_secret`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS mainflux_client CASCADE`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS mainflux_key`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS owner CASCADE`,
					`ALTER TABLE IF EXISTS channels DROP COLUMN IF EXISTS magistrala_channel CASCADE`,
					`ALTER TABLE IF EXISTS channels DROP COLUMN IF EXISTS mainflux_channel CASCADE`,
					`ALTER TABLE IF EXISTS channels DROP COLUMN IF EXISTS owner CASCADE`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS magistrala_client TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS magistrala_secret TEXT`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS magistrala_channel TEXT`,
					`UPDATE configs SET magistrala_client = client_id WHERE magistrala_client IS NULL`,
					`UPDATE configs SET magistrala_secret = client_secret WHERE magistrala_secret IS NULL`,
					`UPDATE channels SET magistrala_channel = id WHERE magistrala_channel IS NULL`,
				},
			},
			{
				Id: "configs_9",
				Up: []string{
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_id VARCHAR(36)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_group_id VARCHAR(36)`,
					`UPDATE channels SET parent_group_id = parent_id WHERE (parent_group_id IS NULL OR parent_group_id = '') AND parent_id IS NOT NULL`,
					`ALTER TABLE IF EXISTS channels DROP COLUMN IF EXISTS parent_id`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_id VARCHAR(36)`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_group_id VARCHAR(36)`,
					`UPDATE channels SET parent_id = parent_group_id WHERE parent_id IS NULL AND parent_group_id IS NOT NULL`,
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
					`CREATE TABLE IF NOT EXISTS binding_snapshots (
						config_id       TEXT NOT NULL,
						slot            VARCHAR(256) NOT NULL,
						type            VARCHAR(64) NOT NULL,
						resource_id     TEXT NOT NULL,
						snapshot        JSONB,
						secret_snapshot BYTEA,
						updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
						PRIMARY KEY (config_id, slot)
					)`,
					`CREATE INDEX IF NOT EXISTS idx_binding_snapshots_config_id ON binding_snapshots (config_id)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS binding_snapshots`,
				},
			},
		},
	}

	return bootstrapMigration, nil
}
