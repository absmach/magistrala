// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of Certs service.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "certs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS certs (
						thing_id     TEXT NOT NULL,
						owner_id     TEXT NOT NULL,
						expire       TIMESTAMPTZ NOT NULL,
						serial       TEXT NOT NULL,
						PRIMARY KEY  (thing_id, owner_id, serial)
					);`,
				},
				Down: []string{
					"DROP TABLE IF EXISTS certs;",
				},
			},
		},
	}
}
