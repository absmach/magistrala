// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	gpostgres "github.com/absmach/magistrala/groups/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	rolesPostgres "github.com/absmach/magistrala/pkg/roles/repo/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

func Migration() (*migrate.MemoryMigrationSource, error) {
	clientsRolesMigration, err := rolesPostgres.Migration(rolesTableNamePrefix, entityTableName, entityIDColumnName)
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}

	clientsMigration := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "clients_01",
				// VARCHAR(36) for columns with IDs as UUIDS have a maximum of 36 characters
				// STATUS 0 to imply enabled and 1 to imply disabled
				Up: []string{
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
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    VARCHAR(36),
						domain_id 	  VARCHAR(36),
						client_id     VARCHAR(36),
						type          SMALLINT NOT NULL CHECK (type IN (1, 2)),
						FOREIGN KEY   (client_id, domain_id) REFERENCES clients (id, domain_id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY   (channel_id, domain_id, client_id, type)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS clients`,
					`DROP TABLE IF EXISTS connections`,
				},
			},
			{
				Id: "clients_02",
				Up: []string{
					`ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_domain_id_name_key`,
				},
				Down: []string{
					`ALTER TABLE clients ADD CONSTRAINT clients_domain_id_name_key UNIQUE (domain_id, name)`,
				},
			},
			{
				Id: "clients_03",
				Up: []string{
					`ALTER TABLE clients ALTER COLUMN created_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE clients ALTER COLUMN updated_at TYPE TIMESTAMPTZ;`,
				},
				Down: []string{
					`ALTER TABLE clients ALTER COLUMN created_at TYPE TIMESTAMP;`,
					`ALTER TABLE clients ALTER COLUMN updated_at TYPE TIMESTAMP;`,
				},
			},
			{
				Id: "clients_04",
				Up: []string{
					`ALTER TABLE clients ADD COLUMN private_metadata JSONB;`,
				},
				Down: []string{
					`ALTER TABLE clients DROP COLUMN private_metadata;`,
				},
			},
			{
				Id: "clients_05",
				Up: []string{
					`UPDATE clients 
					 SET metadata = (COALESCE(metadata, '{}'::jsonb) || COALESCE(metadata->'ui', '{}'::jsonb)) - 'ui'
					 WHERE metadata ? 'ui' AND jsonb_typeof(metadata->'ui') = 'object'`,
					`UPDATE clients 
					 SET private_metadata = (COALESCE(private_metadata, '{}'::jsonb) || COALESCE(private_metadata->'ui', '{}'::jsonb)) - 'ui'
					 WHERE private_metadata ? 'ui' AND jsonb_typeof(private_metadata->'ui') = 'object'`,
				},
				Down: []string{
					`SELECT 1`,
				},
			},
			{
				Id: "clients_06",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_clients_domain_id_status ON clients(domain_id, status);`,
					`CREATE INDEX IF NOT EXISTS idx_clients_parent_group_id ON clients(parent_group_id);`,
					`CREATE INDEX IF NOT EXISTS idx_connections_client_id ON connections(client_id);`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_clients_domain_id_status;`,
					`DROP INDEX IF EXISTS idx_clients_parent_group_id;`,
					`DROP INDEX IF EXISTS idx_connections_client_id;`,
				},
			},
		},
	}

	clientsMigration.Migrations = append(clientsMigration.Migrations, clientsRolesMigration.Migrations...)

	groupsMigration, err := gpostgres.Migration()
	if err != nil {
		return &migrate.MemoryMigrationSource{}, err
	}

	clientsMigration.Migrations = append(clientsMigration.Migrations, groupsMigration.Migrations...)

	return clientsMigration, nil
}
