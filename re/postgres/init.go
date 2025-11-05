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
				Id: "rules_01",
				// VARCHAR(36) for colums with IDs as UUIDS have a maximum of 36 characters
				// STATUS 0 to imply enabled and 1 to imply disabled
				Up: []string{
					`CREATE TABLE IF NOT EXISTS rules (
						id                VARCHAR(36) PRIMARY KEY,
						name              VARCHAR(1024),
						domain_id         VARCHAR(36) NOT NULL,
						metadata          JSONB,
						created_by        VARCHAR(254),
						created_at        TIMESTAMP,
						updated_at        TIMESTAMP,
						updated_by        VARCHAR(254),
						input_channel     VARCHAR(36),
						input_topic       TEXT,
						outputs           JSONB,
						status            SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						logic_type        SMALLINT NOT NULL DEFAULT 0 CHECK (logic_type >= 0),
						logic_value       BYTEA,
						time              TIMESTAMP,
						recurring         SMALLINT,
						recurring_period  SMALLINT,
						start_datetime    TIMESTAMP
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS rules`,
				},
			},
			{
				Id: "rules_02",
				Up: []string{
					`ALTER TABLE rules ADD COLUMN tags TEXT[];`,
				},
				Down: []string{
					`ALTER TABLE rules DROP COLUMN tags;`,
				},
			},
			{
				Id: "rules_03",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS rule_executions (
						id            VARCHAR(36) PRIMARY KEY,
						rule_id       VARCHAR(36) NOT NULL,
						level         VARCHAR(10) NOT NULL,
						message       TEXT NOT NULL,
						error         TEXT,
						exec_time     TIMESTAMP NOT NULL,
						created_at    TIMESTAMP NOT NULL,
						FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE
					)`,
					`CREATE INDEX idx_rule_executions_rule_id ON rule_executions(rule_id)`,
					`CREATE INDEX idx_rule_executions_created_at ON rule_executions(created_at DESC)`,
					`CREATE INDEX idx_rule_executions_exec_time ON rule_executions(exec_time DESC)`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_rule_executions_exec_time`,
					`DROP INDEX IF EXISTS idx_rule_executions_created_at`,
					`DROP INDEX IF EXISTS idx_rule_executions_rule_id`,
					`DROP TABLE IF EXISTS rule_executions`,
				},
			},
		},
	}
}
