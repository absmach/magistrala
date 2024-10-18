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
	entityForeignKeyTableName  = "channels"
	entityForeignKeyColumnName = "id"
)

func Migration() (*migrate.MemoryMigrationSource, error) {
	rolesMigration, err := entityRolesRepo.Migration(rolesTableNamePrefix, entityForeignKeyTableName, entityForeignKeyColumnName)
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
						id			VARCHAR(36) PRIMARY KEY,
						name		VARCHAR(1024),
						domain_id	VARCHAR(36) NOT NULL,
						tags		TEXT[],
						metadata	JSONB,
						created_by  VARCHAR(254),
						created_at	TIMESTAMP,
						updated_at	TIMESTAMP,
						updated_by  VARCHAR(254),
						status		SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE		(domain_id, name),
						UNIQUE		(domain_id, id)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    VARCHAR(36),
						domain_id 	  VARCHAR(36),
						thing_id      VARCHAR(36),
						FOREIGN KEY (channel_id, domain_id) REFERENCES channels (id, domain_id) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (thing_id, domain_id) REFERENCES clients (id, domain_id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (channel_id, domain_id, thing_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS channels`,
					`DROP TABLE IF EXISTS connections`,
				},
			},
		},
	}
	channelsMigration.Migrations = append(channelsMigration.Migrations, rolesMigration.Migrations...)
	return channelsMigration, nil
}
