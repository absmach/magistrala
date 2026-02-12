// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

func Migration() (*migrate.MemoryMigrationSource, error) {
	rolesMigration, err := rolesPostgres.Migration(rolesTableNamePrefix, entityTableName, entityIDColumnName)
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}
	rulesMigration := &migrate.MemoryMigrationSource{
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
		},
	}

	rulesMigration.Migrations = append(rulesMigration.Migrations, rolesMigration.Migrations...)
	return rulesMigration, nil
}
