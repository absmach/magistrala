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
				// The legacy `state` column and the `status` column that
				// replaces it use opposite encodings: State was
				// Inactive = 0 / Active = 1, while Status is
				// EnabledStatus = 0 / DisabledStatus = 1. Renaming the column
				// without remapping its values would silently invert every
				// stored row — locking out every whitelisted device and
				// letting every deliberately non-whitelisted one bootstrap.
				// The values are therefore flipped as part of the rename.
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
							UPDATE configs SET status = 1 - status WHERE status IN (0, 1);
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
							UPDATE configs SET status = 1 - status WHERE status IN (0, 1);
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
			{
				Id: "configs_16",
				Up: []string{
					`ALTER TABLE IF EXISTS profiles RENAME COLUMN template_format TO content_format`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS profiles RENAME COLUMN content_format TO template_format`,
				},
			},
			{
				Id: "configs_17",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS domain_transport_keys (
						domain_id        VARCHAR(36) NOT NULL,
						key_id           VARCHAR(36) NOT NULL,
						encrypted_secret TEXT NOT NULL,
						wrapping_key_id  VARCHAR(128) NOT NULL,
						status           VARCHAR(16) NOT NULL CHECK (status IN ('active', 'retiring', 'revoked')),
						created_at       TIMESTAMPTZ NOT NULL,
						updated_at       TIMESTAMPTZ NOT NULL,
						retire_at        TIMESTAMPTZ,
						PRIMARY KEY (domain_id, key_id)
					)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS idx_domain_transport_keys_active
						ON domain_transport_keys (domain_id) WHERE status = 'active'`,
					`CREATE TABLE IF NOT EXISTS secure_bootstrap_requests (
						domain_id VARCHAR(36) NOT NULL,
						key_id VARCHAR(36) NOT NULL,
						request_id VARCHAR(128) NOT NULL,
						expires_at TIMESTAMPTZ NOT NULL,
						PRIMARY KEY (domain_id, key_id, request_id),
						FOREIGN KEY (domain_id, key_id) REFERENCES domain_transport_keys (domain_id, key_id) ON DELETE CASCADE
					)`,
					`CREATE INDEX IF NOT EXISTS idx_secure_bootstrap_requests_expires_at
						ON secure_bootstrap_requests (expires_at)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS secure_bootstrap_requests`,
					`DROP TABLE IF EXISTS domain_transport_keys`,
				},
			},
			{
				Id: "configs_18",
				Up: []string{
					`ALTER TABLE IF EXISTS configs
						ADD COLUMN IF NOT EXISTS bootstrap_key_version BIGINT NOT NULL DEFAULT 1`,
					`DROP TABLE IF EXISTS secure_bootstrap_requests`,
					`DROP TABLE IF EXISTS domain_transport_keys`,
					`CREATE TABLE IF NOT EXISTS bootstrap_challenges (
						challenge_id VARCHAR(36) PRIMARY KEY,
						config_id VARCHAR(36) NOT NULL,
						key_version BIGINT NOT NULL,
						server_nonce BYTEA NOT NULL,
						created_at TIMESTAMPTZ NOT NULL,
						expires_at TIMESTAMPTZ NOT NULL,
						consumed_at TIMESTAMPTZ
					)`,
					`CREATE INDEX IF NOT EXISTS idx_bootstrap_challenges_config_id
						ON bootstrap_challenges (config_id)`,
					`CREATE INDEX IF NOT EXISTS idx_bootstrap_challenges_expires_at
						ON bootstrap_challenges (expires_at)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS bootstrap_challenges`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS bootstrap_key_version`,
				},
			},
			{
				Id: "configs_19",
				Up: []string{
					`ALTER TABLE IF EXISTS profiles
						ADD COLUMN IF NOT EXISTS content_type VARCHAR(128) NOT NULL DEFAULT 'text/plain'`,
					`UPDATE profiles SET content_type = CASE content_format
						WHEN 'json' THEN 'application/json'
						WHEN 'yaml' THEN 'application/yaml'
						WHEN 'toml' THEN 'application/toml'
						ELSE 'text/plain'
					END`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS profiles DROP COLUMN IF EXISTS content_type`,
				},
			},
			{
				// Migration IDs in this legacy sequence are sorted
				// lexicographically by sql-migrate: an ID is ordered
				// numerically only when it begins with digits, and
				// "configs_N" does not. The effective order is therefore
				// configs_1, configs_10..configs_19, configs_2..configs_9.
				//
				// configs_9 consequently runs after configs_8 has renamed
				// client_id to id, while also applying safely to databases
				// that already ran configs_18.
				//
				// Because of that ordering, a new migration must NOT be
				// added as "configs_20": it would sort between configs_2 and
				// configs_3 and run before the configs_4..configs_8 column
				// renames it is likely to depend on. New migrations use the
				// "configs_z<NN>" scheme below, which sorts after every
				// "configs_<digit>" ID.
				Id: "configs_9",
				Up: []string{
					`DO $$
					BEGIN
						IF NOT EXISTS (
							SELECT 1 FROM pg_constraint
							WHERE conname = 'bootstrap_challenges_config_id_fkey'
						) THEN
							ALTER TABLE bootstrap_challenges
								ADD CONSTRAINT bootstrap_challenges_config_id_fkey
								FOREIGN KEY (config_id) REFERENCES configs (id) ON DELETE CASCADE;
						END IF;
					END $$`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS bootstrap_challenges
						DROP CONSTRAINT IF EXISTS bootstrap_challenges_config_id_fkey`,
				},
			},
			{
				// First migration of the "configs_z<NN>" sequence, which sorts
				// after every legacy "configs_<digit>" ID (see configs_9).
				//
				// Renames the tenant column to match the platform-wide rename
				// of domains to workspaces. Guarded so it applies both to a
				// fresh database (where configs_5 and configs_10 have just
				// created domain_id) and to a database upgraded from the
				// pre-removal bootstrap service.
				Id: "configs_z01",
				Up: []string{
					renameColumn("configs", "domain_id", "workspace_id"),
					renameColumn("profiles", "domain_id", "workspace_id"),
					renameIndex("idx_profiles_domain_id", "idx_profiles_workspace_id"),
					renameIndex("configs_client_id_domain_id_key", "configs_client_id_workspace_id_key"),
					renameConstraint("configs", "configs_name_domain_id_key", "configs_name_workspace_id_key"),
				},
				Down: []string{
					renameConstraint("configs", "configs_name_workspace_id_key", "configs_name_domain_id_key"),
					renameIndex("configs_client_id_workspace_id_key", "configs_client_id_domain_id_key"),
					renameIndex("idx_profiles_workspace_id", "idx_profiles_domain_id"),
					renameColumn("profiles", "workspace_id", "domain_id"),
					renameColumn("configs", "workspace_id", "domain_id"),
				},
			},
			{
				// configs_8 renamed the configs.client_id column itself to id,
				// but left the indexes that predate that rename still spelling
				// out "client_id" even though they now index the id column.
				// Renames them to match, as part of the platform-wide rename of
				// clients to devices.
				Id: "configs_z02",
				Up: []string{
					renameIndex("configs_client_id_key", "configs_id_key"),
					renameIndex("configs_client_id_workspace_id_key", "configs_id_workspace_id_key"),
				},
				Down: []string{
					renameIndex("configs_id_workspace_id_key", "configs_client_id_workspace_id_key"),
					renameIndex("configs_id_key", "configs_client_id_key"),
				},
			},
		},
	}
}

// renameColumn builds an idempotent column rename: it runs only when the
// source column is present and the target is not, so the migration is safe on
// both freshly created and already-renamed databases.
func renameColumn(table, from, to string) string {
	return `DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = '` + table + `' AND column_name = '` + from + `'
			) AND NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = '` + table + `' AND column_name = '` + to + `'
			) THEN
				ALTER TABLE ` + table + ` RENAME COLUMN ` + from + ` TO ` + to + `;
			END IF;
		END $$`
}

// renameIndex builds an idempotent index rename, guarded the same way as
// renameColumn.
func renameIndex(from, to string) string {
	return `DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM pg_class WHERE relname = '` + from + `'
			) AND NOT EXISTS (
				SELECT 1 FROM pg_class WHERE relname = '` + to + `'
			) THEN
				ALTER INDEX ` + from + ` RENAME TO ` + to + `;
			END IF;
		END $$`
}

// renameConstraint builds an idempotent constraint rename, guarded the same
// way as renameColumn.
func renameConstraint(table, from, to string) string {
	return `DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = '` + from + `'
			) AND NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = '` + to + `'
			) THEN
				ALTER TABLE ` + table + ` RENAME CONSTRAINT ` + from + ` TO ` + to + `;
			END IF;
		END $$`
}
