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
				Id: "clients_01",
				// VARCHAR(36) for column with IDs as UUIDS have a maximum of 36 characters
				// STATUS 0 to imply enabled and 1 to imply disabled
				// Role 0 to imply user role and 1 to imply admin role
				Up: []string{
					`CREATE TABLE IF NOT EXISTS clients (
						id          VARCHAR(36) PRIMARY KEY,
						name        VARCHAR(254) NOT NULL UNIQUE,
						domain_id   VARCHAR(36),
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
				},
				Down: []string{
					`DROP TABLE IF EXISTS clients`,
				},
			},
			{
				// To support creation of clients from Oauth2 provider
				Id: "clients_02",
				Up: []string{
					`ALTER TABLE clients ALTER COLUMN secret DROP NOT NULL`,
				},
				Down: []string{},
			},
			{
				Id: "clients_03",
				Up: []string{
					`ALTER TABLE clients
                        ADD COLUMN username VARCHAR(254) UNIQUE,
                        ADD COLUMN first_name VARCHAR(254) NOT NULL DEFAULT '',
                        ADD COLUMN last_name VARCHAR(254) NOT NULL DEFAULT '',
                        ADD COLUMN profile_picture TEXT`,
					`ALTER TABLE clients RENAME COLUMN identity TO email`,
					`ALTER TABLE clients DROP COLUMN name`,
				},
				Down: []string{
					`ALTER TABLE clients
                        DROP COLUMN username,
                        DROP COLUMN first_name,
                        DROP COLUMN last_name,
                        DROP COLUMN profile_picture`,
					`ALTER TABLE clients RENAME COLUMN email TO identity`,
					`ALTER TABLE clients ADD COLUMN name VARCHAR(254) NOT NULL UNIQUE`,
				},
			},
			{
				Id: "clients_04",
				Up: []string{
					`ALTER TABLE IF EXISTS clients RENAME TO users`,
				},
				Down: []string{
					`ALTER TABLE IF EXISTS users RENAME TO clients`,
				},
			},
			{
				Id: "clients_05",
				Up: []string{
					`ALTER TABLE users ALTER COLUMN first_name DROP DEFAULT`,
					`ALTER TABLE users ALTER COLUMN last_name DROP DEFAULT`,
				},
				Down: []string{
					`ALTER TABLE users ALTER COLUMN first_name SET DEFAULT ''`,
					`ALTER TABLE users ALTER COLUMN last_name SET DEFAULT ''`,
				},
			},
			{
				Id: "clients_06",
				Up: []string{
					`ALTER TABLE users ALTER COLUMN created_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE users ALTER COLUMN updated_at TYPE TIMESTAMPTZ;`,
				},
				Down: []string{
					`ALTER TABLE users ALTER COLUMN created_at TYPE TIMESTAMP;`,
					`ALTER TABLE users ALTER COLUMN updated_at TYPE TIMESTAMP;`,
				},
			},
			{
				Id: "clients_07",
				Up: []string{
					`ALTER TABLE users ADD COLUMN verified_at TIMESTAMPTZ DEFAULT NULL;`,
					`CREATE TABLE users_verifications (
						user_id VARCHAR(36) NOT NULL,
						email VARCHAR(254) NOT NULL,
						otp VARCHAR(255),
						created_at TIMESTAMPTZ,
						expires_at  TIMESTAMPTZ,
						used_at TIMESTAMPTZ,
						FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
					);
					CREATE INDEX idx_users_verifications_lookup ON users_verifications (user_id, email, created_at DESC);
					`,
				},
				Down: []string{
					`ALTER TABLE users DROP COLUMN verified_at;`,
					`DROP TABLE users_verifications;`,
				},
			},
			{
				Id: "clients_08",
				Up: []string{
					`ALTER TABLE users RENAME CONSTRAINT clients_identity_key TO clients_email_key;`,
				},
				Down: []string{
					`ALTER TABLE users RENAME CONSTRAINT clients_email_key TO clients_identity_key;`,
				},
			},
			{
				Id: "clients_09",
				Up: []string{
					`ALTER TABLE users ADD COLUMN auth_provider VARCHAR(254);`,
				},
				Down: []string{
					`ALTER TABLE users DROP COLUMN auth_provider`,
				},
			},
			{
				Id: "clients_10",
				Up: []string{
					`ALTER TABLE users ADD COLUMN private_metadata JSONB;`,
				},
				Down: []string{
					`ALTER TABLE users DROP COLUMN private_metadata;`,
				},
			},
			{
				Id: "clients_11",
				Up: []string{
					`UPDATE users 
					 SET metadata = (COALESCE(metadata, '{}'::jsonb) || COALESCE(metadata->'ui', '{}'::jsonb)) - 'ui'
					 WHERE metadata ? 'ui' AND jsonb_typeof(metadata->'ui') = 'object'`,
					`UPDATE users 
					 SET private_metadata = (COALESCE(private_metadata, '{}'::jsonb) || COALESCE(private_metadata->'ui', '{}'::jsonb)) - 'ui'
					 WHERE private_metadata ? 'ui' AND jsonb_typeof(private_metadata->'ui') = 'object'`,
				},
				Down: []string{
					`SELECT 1`,
				},
			},
		},
	}
}
