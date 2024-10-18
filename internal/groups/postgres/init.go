// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	entityRolesRepo "github.com/absmach/magistrala/pkg/entityroles/postrgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

const (
	entityForeignKeyTableName  = "groups"
	entityForeignKeyColumnName = "id"
)

func Migration() (*migrate.MemoryMigrationSource, error) {
	rolesMigration, err := entityRolesRepo.Migration(rolesTableNamePrefix, entityForeignKeyTableName, entityForeignKeyColumnName)
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
					`ALTER TABLE groups ADD COLUMN path LTREE`,
					`CREATE INDEX path_gist_idx ON groups USING GIST (path);`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS groups`,
					`DROP EXTENSION IF EXISTS LTREE`,
				},
			},
		},
	}

	groupsMigration.Migrations = append(groupsMigration.Migrations, rolesMigration.Migrations...)

	return groupsMigration, nil
}
