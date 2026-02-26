// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	gpostgres "github.com/absmach/supermq/groups/postgres"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

func Migration() (*migrate.MemoryMigrationSource, error) {
	rolesMigration, err := rolesPostgres.Migration(rolesTableNamePrefix, entityTableName, entityIDColumnName)
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}
	channelsMigration := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "channels_01",
				// VARCHAR(36) for colums with IDs as UUIDS have a maximum of 36 characters
				// STATUS 0 to imply enabled and 1 to imply disabled
				Up: []string{
					`CREATE TABLE IF NOT EXISTS channels (
						id                 VARCHAR(36) PRIMARY KEY,
						name               VARCHAR(1024),
						domain_id          VARCHAR(36) NOT NULL,
						parent_group_id    VARCHAR(36) DEFAULT NULL,
						tags               TEXT[],
						metadata           JSONB,
						created_by         VARCHAR(254),
						created_at         TIMESTAMP,
						updated_at         TIMESTAMP,
						updated_by         VARCHAR(254),
						status             SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE 			   (id, domain_id),
						UNIQUE             (domain_id, name)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    VARCHAR(36),
						domain_id     VARCHAR(36),
						client_id     VARCHAR(36),
						type          SMALLINT NOT NULL CHECK (type IN (1, 2)),
						FOREIGN KEY   (channel_id, domain_id) REFERENCES channels (id, domain_id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY   (channel_id, domain_id, client_id, type)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS channels`,
					`DROP TABLE IF EXISTS connections`,
				},
			},
			{
				Id: "channels_02",
				Up: []string{
					`ALTER TABLE channels DROP CONSTRAINT IF EXISTS channels_domain_id_name_key`,
				},
				Down: []string{
					`ALTER TABLE channels ADD CONSTRAINT channels_domain_id_name_key UNIQUE (domain_id, name)`,
				},
			},
			{
				Id: "channels_03",
				Up: []string{
					`ALTER TABLE channels ADD COLUMN IF NOT EXISTS route VARCHAR(36);`,
					`CREATE UNIQUE INDEX IF NOT EXISTS unique_domain_route_not_null ON channels (domain_id, route) WHERE route IS NOT NULL;`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS unique_domain_route_not_null;`,
					`ALTER TABLE channels DROP COLUMN IF EXISTS route;`,
				},
			},
			{
				Id: "channels_04",
				Up: []string{
					`ALTER TABLE channels ALTER COLUMN created_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE channels ALTER COLUMN updated_at TYPE TIMESTAMPTZ;`,
				},
				Down: []string{
					`ALTER TABLE channels ALTER COLUMN created_at TYPE TIMESTAMP;`,
					`ALTER TABLE channels ALTER COLUMN updated_at TYPE TIMESTAMP;`,
				},
			},
			{
				Id: "channels_05",
				Up: []string{
					`UPDATE channels 
					 SET metadata = COALESCE(metadata, '{}'::jsonb) || COALESCE(metadata->'ui', '{}'::jsonb) - 'ui'
					 WHERE metadata ? 'ui' AND jsonb_typeof(metadata->'ui') = 'object'`,
				},
				Down: []string{
					`SELECT 1`,
				},
			},
		},
	}
	channelsMigration.Migrations = append(channelsMigration.Migrations, rolesMigration.Migrations...)

	groupsMigration, err := gpostgres.Migration()
	if err != nil {
		return &migrate.MemoryMigrationSource{}, err
	}

	channelsMigration.Migrations = append(channelsMigration.Migrations, groupsMigration.Migrations...)

	return channelsMigration, nil
}
