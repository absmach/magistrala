// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	dpostgres "github.com/absmach/supermq/domains/postgres"
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

	groupsMigration := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "groups_01",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS groups (
						id			VARCHAR(36) PRIMARY KEY,
						parent_id	VARCHAR(36),
						domain_id	VARCHAR(36) NOT NULL,
						name		VARCHAR(1024) NOT NULL,
						description	VARCHAR(1024),
						metadata	JSONB,
						created_at	TIMESTAMP,
						updated_at	TIMESTAMP,
						updated_by  VARCHAR(254),
						status		SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE		(domain_id, name),
						FOREIGN KEY (parent_id) REFERENCES groups (id) ON DELETE SET NULL,
						CHECK (id != parent_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS groups`,
				},
			},
			{
				Id: "groups_02",
				Up: []string{
					`CREATE EXTENSION IF NOT EXISTS LTREE`,
					`ALTER TABLE groups ADD COLUMN IF NOT EXISTS path LTREE`,
					`CREATE INDEX IF NOT EXISTS path_gist_idx ON groups USING GIST (path);`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS groups`,
					`DROP EXTENSION IF EXISTS LTREE`,
				},
			},
			{
				Id: "groups_03",
				Up: []string{
					`ALTER TABLE groups DROP CONSTRAINT IF EXISTS groups_domain_id_name_key`,
				},
				Down: []string{
					`ALTER TABLE groups ADD CONSTRAINT groups_domain_id_name_key UNIQUE (domain_id, name)`,
				},
			},
			{
				Id: "groups_04",
				Up: []string{
					`ALTER TABLE groups ADD COLUMN IF NOT EXISTS tags TEXT[]`,
				},
				Down: []string{
					`ALTER TABLE groups DROP COLUMN tags`,
				},
			},
			{
				Id: "groups_05",
				Up: []string{
					`ALTER TABLE groups ALTER COLUMN created_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE groups ALTER COLUMN updated_at TYPE TIMESTAMPTZ;`,
				},
				Down: []string{
					`ALTER TABLE groups ALTER COLUMN created_at TYPE TIMESTAMP;`,
					`ALTER TABLE groups ALTER COLUMN updated_at TYPE TIMESTAMP;`,
				},
			},
			{
				Id: "groups_06",
				Up: []string{
					`UPDATE groups 
					 SET metadata = (COALESCE(metadata, '{}'::jsonb) || COALESCE(metadata->'ui', '{}'::jsonb)) - 'ui'
					 WHERE metadata ? 'ui' AND jsonb_typeof(metadata->'ui') = 'object'`,
				},
				Down: []string{
					`SELECT 1`,
				},
			},
		},
	}

	groupsMigration.Migrations = append(groupsMigration.Migrations, rolesMigration.Migrations...)

	domainsMigrations, err := dpostgres.Migration()
	if err != nil {
		return nil, err
	}
	groupsMigration.Migrations = append(groupsMigration.Migrations, domainsMigrations.Migrations...)

	return groupsMigration, nil
}
