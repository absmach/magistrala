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
				Id: "reports_01",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS report_config (
						id          	 	VARCHAR(36) PRIMARY KEY,
						name				VARCHAR(1024),
						description			TEXT,
						domain_id         	VARCHAR(36) NOT NULL,
						status				SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						created_at			TIMESTAMP,
						created_by			VARCHAR(254),
						updated_at			TIMESTAMP,
						updated_by			VARCHAR(254),
						due              	TIMESTAMPTZ,
						recurring         	SMALLINT,
						recurring_period  	SMALLINT,
						start_datetime    	TIMESTAMP,
						config			  	JSONB,
						email				JSONB,
						metrics				JSONB
					);`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS report_config;`,
				},
			},
		},
	}
}
