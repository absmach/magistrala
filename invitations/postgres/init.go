// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "invitations_01",
				// VARCHAR(36) for colums with IDs as UUIDS have a maximum of 36 characters
				Up: []string{
					`CREATE TABLE IF NOT EXISTS invitations (
						invited_by		VARCHAR(36) NOT NULL,
						user_id			VARCHAR(36) NOT NULL,
						domain_id		VARCHAR(36) NOT NULL,
						token			TEXT NOT NULL,
						relation		VARCHAR(254) NOT NULL,
						created_at		TIMESTAMP NOT NULL,
						updated_at		TIMESTAMP,
						confirmed_at	TIMESTAMP,
						UNIQUE (user_id, domain_id),
						PRIMARY KEY (user_id, domain_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS invitations`,
				},
			},
			{
				Id: "invitations_02_add_rejection",
				Up: []string{
					`ALTER TABLE invitations
					 ADD COLUMN rejected_at TIMESTAMP`,
				},
				Down: []string{
					`ALTER TABLE invitations
					 DROP COLUMN rejected_at`,
				},
			},
		},
	}
}
