// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	rolesPostgres "github.com/absmach/magistrala/pkg/roles/repo/postgres"
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
						PRIMARY KEY   (channel_id, domain_id, client_id, type),
						UNIQUE        (channel_id, client_id)
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
