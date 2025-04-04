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
				Id: "alarms_01",
				// VARCHAR(36) for columns with IDs as UUIDS have a maximum of 36 characters
				Up: []string{
					`CREATE TABLE IF NOT EXISTS rules (
						id         	VARCHAR(36) PRIMARY KEY,
						name		VARCHAR(254) NOT NULL CHECK (length(name) > 0),
						user_id		VARCHAR(36) NOT NULL,
						domain_id	VARCHAR(36) NOT NULL,
						condition	TEXT NOT NULL,
						channel		VARCHAR(36) NOT NULL,
						created_by	VARCHAR(36) NOT NULL,
						created_at	TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at	TIMESTAMPTZ NULL,
						updated_by	VARCHAR(36) NULL,
						metadata	JSONB,
						CONSTRAINT rules_name_unique UNIQUE (name, user_id, domain_id)
					);`,
					`CREATE TABLE IF NOT EXISTS alarms (
						id         	VARCHAR(36) PRIMARY KEY,
						rule_id		VARCHAR(36) NOT NULL,
						message		TEXT NULL,
						status      SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						user_id		VARCHAR(36) NOT NULL,
						domain_id	VARCHAR(36) NOT NULL,
						assignee_id	VARCHAR(36) NULL,
						created_by	VARCHAR(36) NOT NULL,
						created_at	TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at	TIMESTAMPTZ NULL,
						updated_by	VARCHAR(36) NULL,
						resolved_at	TIMESTAMPTZ NULL,
						resolved_by	VARCHAR(36) NULL,
						metadata	JSONB,
						FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE
					);`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS rules`,
					`DROP TABLE IF EXISTS alarms`,
				},
			},
		},
	}
}
