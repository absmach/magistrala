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
				Id: "groups_01",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS groups (
						id			VARCHAR(36) PRIMARY KEY,
						parent_id	VARCHAR(36),
						owner_id	VARCHAR(36) NOT NULL,
						name		VARCHAR(1024) NOT NULL,
						description	VARCHAR(1024),
						metadata	JSONB,
						created_at	TIMESTAMP,
						updated_at	TIMESTAMP,
						updated_by  VARCHAR(254),
						status		SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE		(owner_id, name),
						FOREIGN KEY (parent_id) REFERENCES groups (id) ON DELETE SET NULL
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS groups`,
				},
			},
		},
	}
}
