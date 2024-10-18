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

					`CREATE TABLE IF NOT EXISTS domains (
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
					`ALTER TABLE domains ALTER COLUMN alias SET NOT NULL`,
				},
			},
			{
				Id: "auth_3",
				Up: []string{
					`DROP TABLE IF EXISTS policies;
                     DROP TABLE IF EXISTS domains;
                    `,
				},
			},
		},
	}
}
