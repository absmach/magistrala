// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of Auth service.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "auth_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS keys (
						id          VARCHAR(254) NOT NULL,
						type        SMALLINT,
						subject     VARCHAR(254) NOT NULL,
						issuer_id   VARCHAR(254) NOT NULL,
						issued_at   TIMESTAMP NOT NULL,
						expires_at  TIMESTAMP,
						PRIMARY KEY (id, issuer_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS keys`,
				},
			},
		},
	}
}
