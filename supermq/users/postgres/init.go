// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of Users service.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "clients_01",
				// VARCHAR(36) for colums with IDs as UUIDS have a maximum of 36 characters
				// STATUS 0 to imply enabled and 1 to imply disabled
				// Role 0 to imply user role and 1 to imply admin role
				Up: []string{
					`CREATE TABLE IF NOT EXISTS clients (
						id          VARCHAR(36) PRIMARY KEY,
						name        VARCHAR(254) NOT NULL UNIQUE,
						domain_id   VARCHAR(36),
						identity    VARCHAR(254) NOT NULL UNIQUE,
						secret      TEXT NOT NULL,
						tags        TEXT[],
						metadata    JSONB,
						created_at  TIMESTAMP,
						updated_at  TIMESTAMP,
						updated_by  VARCHAR(254),
						status      SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						role        SMALLINT DEFAULT 0 CHECK (status >= 0)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS clients`,
				},
			},
			{
				// To support creation of clients from Oauth2 provider
				Id: "clients_02",
				Up: []string{
					`ALTER TABLE clients ALTER COLUMN secret DROP NOT NULL`,
				},
				Down: []string{},
			},
			{
				Id: "clients_03",
				Up: []string{
					`ALTER TABLE clients
                        ADD COLUMN username VARCHAR(254) UNIQUE,
                        ADD COLUMN first_name VARCHAR(254) NOT NULL DEFAULT '', 
                        ADD COLUMN last_name VARCHAR(254) NOT NULL DEFAULT '', 
                        ADD COLUMN profile_picture TEXT`,
					`ALTER TABLE clients RENAME COLUMN identity TO email`,
					`ALTER TABLE clients DROP COLUMN name`,
				},
				Down: []string{
					`ALTER TABLE clients
                        DROP COLUMN username,
                        DROP COLUMN first_name,
                        DROP COLUMN last_name,
                        DROP COLUMN profile_picture`,
					`ALTER TABLE clients RENAME COLUMN email TO identity`,
					`ALTER TABLE clients ADD COLUMN name VARCHAR(254) NOT NULL UNIQUE`,
				},
			},
			{
				Id: "clients_04",
				Up: []string{
					`ALTER TABLE IF EXISTS clients RENAME TO users`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS users RENAME TO clients`,
				},
			},
			{
				Id: "clients_05",
				Up: []string{
					`ALTER TABLE users ALTER COLUMN first_name DROP DEFAULT`,
					`ALTER TABLE users ALTER COLUMN last_name DROP DEFAULT`,
				},
				Down: []string{
					`ALTER TABLE users ALTER COLUMN first_name SET DEFAULT ''`,
					`ALTER TABLE users ALTER COLUMN last_name SET DEFAULT ''`,
				},
			},
		},
	}
}
