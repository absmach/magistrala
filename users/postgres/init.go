// Copyright (c) Mainflux
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
						name        VARCHAR(254),
						owner_id    VARCHAR(36),
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
					`CREATE TABLE IF NOT EXISTS groups (
						id          VARCHAR(36) PRIMARY KEY,
						parent_id   VARCHAR(36),
						owner_id    VARCHAR(36) NOT NULL,
						name        VARCHAR(254) NOT NULL,
						description VARCHAR(1024),
						metadata    JSONB,
						created_at  TIMESTAMP,
						updated_at  TIMESTAMP,
						updated_by  VARCHAR(254),
						status      SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE (owner_id, name),
						FOREIGN KEY (parent_id) REFERENCES groups (id) ON DELETE CASCADE
					)`,
					`CREATE TABLE IF NOT EXISTS policies (
						owner_id    VARCHAR(36) NOT NULL,
						subject     VARCHAR(36) NOT NULL,
						object      VARCHAR(36) NOT NULL,
						actions     TEXT[] NOT NULL,
						created_at  TIMESTAMP,
						updated_at  TIMESTAMP,
						updated_by  VARCHAR(254),
						FOREIGN KEY (subject) REFERENCES clients (id) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (object) REFERENCES groups (id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (subject, object)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS clients`,
					`DROP TABLE IF EXISTS groups`,
					`DROP TABLE IF EXISTS policies`,
				},
			},
		},
	}
}
