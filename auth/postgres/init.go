// Copyright (c) Abstract Machines
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

					`CREATE TABLE IF NOT EXISTS workspaces (
                        id          VARCHAR(36) PRIMARY KEY,
                        name        VARCHAR(254),
                        tags        TEXT[],
                        metadata    JSONB,
                        alias       VARCHAR(254) NULL UNIQUE,
                        created_at  TIMESTAMP,
                        updated_at  TIMESTAMP,
                        updated_by  VARCHAR(254),
                        created_by  VARCHAR(254),
                        status      SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0)
                    );`,
					`CREATE TABLE IF NOT EXISTS policies (
                        subject_type        VARCHAR(254) NOT NULL,
                        subject_id          VARCHAR(254) NOT NULL,
                        subject_relation    VARCHAR(254) NOT NULL,
                        relation            VARCHAR(254) NOT NULL,
                        object_type         VARCHAR(254) NOT NULL,
                        object_id           VARCHAR(254) NOT NULL,
                        CONSTRAINT unique_policy_constraint UNIQUE (subject_type, subject_id, subject_relation, relation, object_type, object_id)
                    );`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS keys`,
				},
			},
			{
				Id: "auth_2",
				Up: []string{
					`ALTER TABLE workspaces ALTER COLUMN alias SET NOT NULL`,
				},
			},
			{
				Id: "auth_3",
				Up: []string{
					`DROP TABLE IF EXISTS policies;
                     DROP TABLE IF EXISTS workspaces;
                    `,
				},
			},
			{
				Id: "auth_4",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS pats (
						id 					VARCHAR(36) PRIMARY KEY,
                        name        		VARCHAR(254) NOT NULL,
						user_id	 			VARCHAR(36),
						description			TEXT,
						secret				TEXT,
						issued_at			TIMESTAMP,
						expires_at 			TIMESTAMP,
						updated_at 			TIMESTAMP,
						revoked 			BOOLEAN,
						revoked_at 			TIMESTAMP,
						last_used_at		TIMESTAMP,
						UNIQUE 				(id, name, secret)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS pats`,
				},
			},
			{
				Id: "auth_5",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS pat_scopes (
						id              	VARCHAR(36) PRIMARY KEY,
						pat_id          	VARCHAR(36) REFERENCES pats(id) ON DELETE CASCADE,
						optional_workspace_id	VARCHAR(36),
						entity_type     	VARCHAR(50) NOT NULL,
						operation 			VARCHAR(50) NOT NULL,
						entity_id			VARCHAR(50) NOT NULL,
						UNIQUE (pat_id, optional_workspace_id, entity_type, operation, entity_id)
					);`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS pat_scopes;`,
				},
			},
			{
				Id: "auth_6",
				Up: []string{
					`ALTER TABLE keys ALTER COLUMN issued_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE keys ALTER COLUMN expires_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE pats ALTER COLUMN issued_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE pats ALTER COLUMN expires_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE pats ALTER COLUMN updated_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE pats ALTER COLUMN revoked_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE pats ALTER COLUMN last_used_at TYPE TIMESTAMPTZ;`,
				},
				Down: []string{
					`ALTER TABLE keys ALTER COLUMN issued_at TYPE TIMESTAMP;`,
					`ALTER TABLE keys ALTER COLUMN expires_at TYPE TIMESTAMP;`,
					`ALTER TABLE pats ALTER COLUMN issued_at TYPE TIMESTAMP;`,
					`ALTER TABLE pats ALTER COLUMN expires_at TYPE TIMESTAMP;`,
					`ALTER TABLE pats ALTER COLUMN updated_at TYPE TIMESTAMP;`,
					`ALTER TABLE pats ALTER COLUMN revoked_at TYPE TIMESTAMP;`,
					`ALTER TABLE pats ALTER COLUMN last_used_at TYPE TIMESTAMP;`,
				},
			},
			{
				Id: "auth_7",
				Up: []string{
					`ALTER TABLE pat_scopes RENAME COLUMN optional_workspace_id TO workspace_id;`,
				},
				Down: []string{
					`ALTER TABLE pat_scopes RENAME COLUMN workspace_id TO optional_workspace_id;`,
				},
			},
			{
				Id: "auth_8",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_pats_user_id ON pats(user_id);`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_pats_user_id;`,
				},
			},
			{
				// ClientsType was removed; rows keep working under the new name
				// rather than failing ParseEntityType (edge/architecture.md §8 C2).
				Id: "auth_9",
				Up: []string{
					`UPDATE pat_scopes SET operation = 'create_devices' WHERE entity_type = 'clients' AND operation = 'create_clients';`,
					`UPDATE pat_scopes SET operation = 'list_devices' WHERE entity_type = 'clients' AND operation = 'list_clients';`,
					`UPDATE pat_scopes SET entity_type = 'devices' WHERE entity_type = 'clients';`,
				},
				Down: []string{
					`UPDATE pat_scopes SET operation = 'create_clients' WHERE entity_type = 'devices' AND operation = 'create_devices';`,
					`UPDATE pat_scopes SET operation = 'list_clients' WHERE entity_type = 'devices' AND operation = 'list_devices';`,
					`UPDATE pat_scopes SET entity_type = 'clients' WHERE entity_type = 'devices';`,
				},
			},
		},
	}
}
