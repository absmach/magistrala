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
				Id: "clients_01",
				// VARCHAR(36) for colums with IDs as UUIDS have a maximum of 36 characters
				// STATUS 0 to imply enabled and 1 to imply disabled
				Up: []string{
					`CREATE TABLE IF NOT EXISTS clients (
						id			VARCHAR(36) PRIMARY KEY,
						name		VARCHAR(1024),
						owner_id	VARCHAR(36),
						identity	VARCHAR(254),
						secret		VARCHAR(4096) NOT NULL,
						tags		TEXT[],
						metadata	JSONB,
						created_at	TIMESTAMP,
						updated_at	TIMESTAMP,
						updated_by  VARCHAR(254),
						status		SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE		(owner_id, secret),
						UNIQUE		(owner_id, name)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS clients`,
				},
			},
		},
	}
}
