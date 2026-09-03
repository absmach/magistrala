// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of bootstrap service.
//
// sql-migrate orders migrations lexicographically unless the id starts with a
// number, so the sequence is zero padded: without the padding "configs_10"
// would sort before "configs_2". Keep the same width when adding migrations,
// and never renumber an id that has already shipped.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				// profiles is created first: configs.profile_id references it.
				//
				// configs.status uses the Status encoding, EnabledStatus = 0 /
				// DisabledStatus = 1 -- the opposite of the legacy State column
				// it replaced, which was Inactive = 0 / Active = 1.
				Id: "configs_0001",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS profiles (
						id               VARCHAR(36) PRIMARY KEY,
						workspace_id     VARCHAR(36) NOT NULL,
						name             VARCHAR(1024) NOT NULL,
						description      TEXT,
						content_format   VARCHAR(64) NOT NULL DEFAULT 'go-template',
						content_type     VARCHAR(128) NOT NULL DEFAULT 'text/plain',
						content_template TEXT,
						defaults         JSONB,
						binding_slots    JSONB,
						version          INT NOT NULL DEFAULT 1,
						created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
						updated_at       TIMESTAMP NOT NULL DEFAULT NOW(),
						UNIQUE (workspace_id, name)
					)`,
					`CREATE INDEX IF NOT EXISTS idx_profiles_workspace_id ON profiles (workspace_id)`,

					`CREATE TABLE IF NOT EXISTS configs (
						id                    TEXT NOT NULL,
						workspace_id          VARCHAR(254),
						name                  TEXT,
						external_id           TEXT UNIQUE NOT NULL,
						external_key          TEXT NOT NULL,
						content               TEXT,
						client_cert           TEXT,
						client_key            TEXT,
						ca_cert               TEXT,
						status                BIGINT NOT NULL,
						profile_id            VARCHAR(36) REFERENCES profiles (id) ON DELETE SET NULL,
						render_context        JSONB,
						bootstrap_key_version BIGINT NOT NULL DEFAULT 1,
						PRIMARY KEY (id, workspace_id),
						CONSTRAINT configs_name_workspace_id_key UNIQUE (name, workspace_id)
					)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS configs_id_key ON configs (id)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS configs_id_workspace_id_key ON configs (id, workspace_id)`,

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

					`CREATE TABLE IF NOT EXISTS bootstrap_challenges (
						challenge_id VARCHAR(36) PRIMARY KEY,
						config_id    VARCHAR(36) NOT NULL REFERENCES configs (id) ON DELETE CASCADE,
						key_version  BIGINT NOT NULL,
						server_nonce BYTEA NOT NULL,
						created_at   TIMESTAMPTZ NOT NULL,
						expires_at   TIMESTAMPTZ NOT NULL,
						consumed_at  TIMESTAMPTZ
					)`,
					`CREATE INDEX IF NOT EXISTS idx_bootstrap_challenges_config_id
						ON bootstrap_challenges (config_id)`,
					`CREATE INDEX IF NOT EXISTS idx_bootstrap_challenges_expires_at
						ON bootstrap_challenges (expires_at)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS bootstrap_challenges`,
					`DROP TABLE IF EXISTS bindings`,
					`DROP TABLE IF EXISTS configs`,
					`DROP TABLE IF EXISTS profiles`,
				},
			},
		},
	}
}
