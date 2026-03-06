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
				Id: "journal_01",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS journal (
						id          VARCHAR(36) PRIMARY KEY,
						operation	VARCHAR NOT NULL,
						domain		VARCHAR,
						occurred_at	TIMESTAMP NOT NULL,
						attributes	JSONB NOT NULL,
						metadata	JSONB,
						UNIQUE(operation, occurred_at, attributes)
					)`,
					`CREATE INDEX idx_journal_default_user_filter ON journal(operation, (attributes->>'id'), (attributes->>'user_id'), occurred_at DESC);`,
					`CREATE INDEX idx_journal_default_group_filter ON journal(operation, (attributes->>'id'), (attributes->>'group_id'), occurred_at DESC);`,
					`CREATE INDEX idx_journal_default_client_filter ON journal(operation, (attributes->>'id'), (attributes->>'client_id'), occurred_at DESC);`,
					`CREATE INDEX idx_journal_default_channel_filter ON journal(operation, (attributes->>'id'), (attributes->>'channel_id'), occurred_at DESC);`,
					`CREATE TABLE IF NOT EXISTS clients_telemetry (
						client_id         VARCHAR(36) PRIMARY KEY,
						domain_id         VARCHAR(36) NOT NULL,
						inbound_messages  BIGINT DEFAULT 0,
						outbound_messages BIGINT DEFAULT 0,
						first_seen        TIMESTAMP,
						last_seen         TIMESTAMP
					)`,
					`CREATE TABLE IF NOT EXISTS subscriptions (
						id              VARCHAR(36) PRIMARY KEY,
						subscriber_id   VARCHAR(1024) NOT NULL,
						channel_id      VARCHAR(36) NOT NULL,
						subtopic        VARCHAR(1024),
						client_id       VARCHAR(36),
						FOREIGN KEY (client_id) REFERENCES clients_telemetry(client_id) ON DELETE CASCADE ON UPDATE CASCADE
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS clients_telemetry`,
					`DROP TABLE IF EXISTS subscriptions`,
					`DROP TABLE IF EXISTS journal`,
				},
			},
			{
				Id: "journal_02",
				Up: []string{
					`ALTER TABLE journal ALTER COLUMN occurred_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE clients_telemetry ALTER COLUMN first_seen TYPE TIMESTAMPTZ;`,
					`ALTER TABLE clients_telemetry ALTER COLUMN last_seen TYPE TIMESTAMPTZ;`,
				},
				Down: []string{
					`ALTER TABLE journal ALTER COLUMN occurred_at TYPE TIMESTAMP;`,
					`ALTER TABLE clients_telemetry ALTER COLUMN first_seen TYPE TIMESTAMP;`,
					`ALTER TABLE clients_telemetry ALTER COLUMN last_seen TYPE TIMESTAMP;`,
				},
			},
			{
				Id: "journal_03",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_journal_domain ON journal(domain);`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_journal_domain;`,
				},
			},
		},
	}
}
