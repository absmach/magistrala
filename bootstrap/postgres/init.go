// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	channelspg "github.com/absmach/magistrala/channels/postgres"
	clientspg "github.com/absmach/magistrala/clients/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
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
					`ALTER TABLE IF EXISTS channels ALTER COLUMN metadata TYPE JSONB USING metadata::jsonb`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS id VARCHAR(36)`,
					`UPDATE channels SET id = magistrala_channel WHERE id IS NULL`,
					`ALTER TABLE IF EXISTS channels ALTER COLUMN id SET NOT NULL`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_group_id VARCHAR(36)`,
					`UPDATE channels SET parent_group_id = parent_id WHERE parent_group_id IS NULL`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS tags TEXT[]`,
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS created_by VARCHAR(254)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS channels_id_key ON channels (id)`,
					`CREATE UNIQUE INDEX IF NOT EXISTS channels_id_domain_id_key ON channels (id, domain_id)`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS client_id TEXT`,
					`ALTER TABLE IF EXISTS configs ADD COLUMN IF NOT EXISTS client_secret CHAR(36)`,
					`UPDATE configs SET client_id = magistrala_client WHERE client_id IS NULL`,
					`UPDATE configs SET client_secret = magistrala_secret WHERE client_secret IS NULL`,
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
					`DO $$
					BEGIN
						IF EXISTS (
							SELECT 1
							FROM information_schema.tables
							WHERE table_schema = 'public' AND table_name = 'connections'
						) THEN
							INSERT INTO config_channels (channel_id, config_id, domain_id)
							SELECT channel_id, config_id, COALESCE(config_owner, channel_owner, '')
							FROM connections
							ON CONFLICT (channel_id, config_id, domain_id) DO NOTHING;
						END IF;
					END $$`,
					`DROP TABLE IF EXISTS connections`,
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
					`DROP TABLE IF EXISTS clients`,
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
					`ALTER TABLE IF EXISTS channels DROP CONSTRAINT IF EXISTS channels_pkey`,
					`ALTER TABLE IF EXISTS channels ALTER COLUMN magistrala_channel DROP NOT NULL`,
				},
				Down: []string{
					`UPDATE channels SET magistrala_channel = id WHERE magistrala_channel IS NULL`,
					`ALTER TABLE IF EXISTS channels ALTER COLUMN magistrala_channel SET NOT NULL`,
					`ALTER TABLE IF EXISTS channels ADD PRIMARY KEY (magistrala_channel, domain_id)`,
				},
			},
			{
				Id: "configs_9",
				Up: []string{
					`UPDATE configs SET client_id = magistrala_client WHERE (client_id IS NULL OR client_id = '') AND magistrala_client IS NOT NULL`,
					`UPDATE configs SET client_secret = magistrala_secret WHERE (client_secret IS NULL OR client_secret = '') AND magistrala_secret IS NOT NULL`,
					`UPDATE channels SET id = magistrala_channel WHERE (id IS NULL OR id = '') AND magistrala_channel IS NOT NULL`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS magistrala_client`,
					`ALTER TABLE IF EXISTS configs DROP COLUMN IF EXISTS magistrala_secret`,
					`ALTER TABLE IF EXISTS channels DROP COLUMN IF EXISTS magistrala_channel`,
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
				Id: "configs_10",
				Up: []string{
					`UPDATE channels SET parent_group_id = parent_id WHERE (parent_group_id IS NULL OR parent_group_id = '') AND parent_id IS NOT NULL`,
					`ALTER TABLE IF EXISTS channels DROP COLUMN IF EXISTS parent_id`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS channels ADD COLUMN IF NOT EXISTS parent_id VARCHAR(36)`,
					`UPDATE channels SET parent_id = parent_group_id WHERE parent_id IS NULL`,
				},
			},
		},
	}

	channelsMigration, err := channelspg.Migration()
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}

	seen := map[string]struct{}{}
	bootstrapMigration.Migrations = append(bootstrapMigration.Migrations, withMigrationPrefix(channelsMigration.Migrations, "zz01_", seen)...)

	clientsMigration, err := clientspg.Migration()
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}

	bootstrapMigration.Migrations = append(bootstrapMigration.Migrations, withMigrationPrefix(clientsMigration.Migrations, "zz02_", seen)...)
	bootstrapMigration.Migrations = append(bootstrapMigration.Migrations, &migrate.Migration{
		Id: "zz03_bootstrap_connections_client_fk",
		Up: []string{
			`CREATE INDEX IF NOT EXISTS idx_connections_client_domain
				ON connections (client_id, domain_id)`,
			`DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1
					FROM pg_constraint
					WHERE conname = 'bootstrap_connections_client_fk'
					  AND conrelid = 'connections'::regclass
				) THEN
					ALTER TABLE connections
					ADD CONSTRAINT bootstrap_connections_client_fk
					FOREIGN KEY (client_id, domain_id)
					REFERENCES clients (id, domain_id)
					ON DELETE CASCADE ON UPDATE CASCADE;
				END IF;
			END $$`,
		},
		Down: []string{
			`ALTER TABLE IF EXISTS connections DROP CONSTRAINT IF EXISTS bootstrap_connections_client_fk`,
			`DROP INDEX IF EXISTS idx_connections_client_domain`,
		},
	})

	return bootstrapMigration, nil
}

func withMigrationPrefix(migrations []*migrate.Migration, prefix string, seen map[string]struct{}) []*migrate.Migration {
	prefixed := make([]*migrate.Migration, 0, len(migrations))
	for _, migration := range migrations {
		if _, ok := seen[migration.Id]; ok {
			continue
		}
		seen[migration.Id] = struct{}{}

		cloned := *migration
		cloned.Id = prefix + migration.Id
		prefixed = append(prefixed, &cloned)
	}
	return prefixed
}
