// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import migrate "github.com/rubenv/sql-migrate"

// Migration of Users service
func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "users_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS users (
					 email    VARCHAR(254) PRIMARY KEY,
					 password CHAR(60)     NOT  NULL
					)`,
				},
				Down: []string{"DROP TABLE users"},
			},
			{
				Id: "users_2",
				Up: []string{
					`ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS metadata JSONB`,
				},
			},
			{
				Id: "users_3",
				Up: []string{
					`CREATE EXTENSION IF NOT EXISTS "pgcrypto";
					 ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS
					 id UUID NOT NULL DEFAULT gen_random_uuid()`,
				},
			},
			{
				Id: "users_4",
				Up: []string{
					`ALTER TABLE IF EXISTS users DROP CONSTRAINT users_pkey`,
					`ALTER TABLE IF EXISTS users ADD CONSTRAINT users_email_key UNIQUE (email)`,
					`ALTER TABLE IF EXISTS users ADD PRIMARY KEY (id)`,
				},
			},
			{
				Id: "users_5",
				Up: []string{
					`DO $$
					BEGIN
						IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_status') THEN
							CREATE TYPE user_status AS ENUM ('enabled', 'disabled');
						END IF;
					END$$;`,
					`ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS
					status USER_STATUS NOT NULL DEFAULT 'enabled'`,
				},
			},
		},
	}
}
