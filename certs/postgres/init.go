// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of Certs service.
//
// sql-migrate orders migrations lexicographically unless the id starts with a
// number, so the sequence is zero padded: without the padding "certs_10" would
// sort before "certs_2". Keep the same width when adding migrations, and never
// renumber an id that has already shipped.
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "certs_0001",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS cert_entity_mappings (
						serial_number VARCHAR(255) UNIQUE NOT NULL,
						entity_id     VARCHAR(255) NOT NULL,
						created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						PRIMARY KEY (serial_number)
                    )`,
					`CREATE INDEX IF NOT EXISTS idx_cert_entity_mappings_entity_id ON cert_entity_mappings(entity_id)`,
				},
				Down: []string{
					"DROP INDEX IF EXISTS idx_cert_entity_mappings_entity_id",
					"DROP TABLE cert_entity_mappings",
				},
			},
		},
	}
}
