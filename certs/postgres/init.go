// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	_ "github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"
)

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "certs_1",
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
