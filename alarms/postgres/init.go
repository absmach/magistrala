// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	dpostgres "github.com/absmach/supermq/domains/postgres"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of Users service.
func Migration() (*migrate.MemoryMigrationSource, error) {
	rolesMigration, err := rolesPostgres.Migration(rolesTableNamePrefix, entityTableName, entityIDColumnName)
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}
	alarmsMigration := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "alarms_01",
				// VARCHAR(36) for columns with IDs as UUIDS have a maximum of 36 characters
				Up: []string{
					`CREATE TABLE IF NOT EXISTS alarms (
						id         	    VARCHAR(36) PRIMARY KEY,
						rule_id		    VARCHAR(36) NOT NULL CHECK (length(rule_id) > 0),
						domain_id	    VARCHAR(36) NOT NULL,
						channel_id	    VARCHAR(36) NOT NULL,
						subtopic        TEXT NOT NULL,
						client_id	    VARCHAR(36) NOT NULL,
						measurement	    TEXT NOT NULL,
						value		    TEXT NOT NULL,
						unit		    TEXT NOT NULL,
						threshold	    TEXT NOT NULL,
						cause	    	TEXT NOT NULL,
						status          SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						severity        SMALLINT NOT NULL DEFAULT 0 CHECK (severity >= 0),
						assignee_id	    VARCHAR(36),
						created_at	    TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at	    TIMESTAMPTZ NULL,
						updated_by	    VARCHAR(36) NULL,
						assigned_at	    TIMESTAMPTZ NULL,
						assigned_by	    VARCHAR(36) NULL,
						acknowledged_at	TIMESTAMPTZ NULL,
						acknowledged_by	VARCHAR(36) NULL,
						resolved_at	    TIMESTAMPTZ NULL,
						resolved_by	    VARCHAR(36) NULL,
						metadata	    JSONB
					);`,
					"CREATE INDEX IF NOT EXISTS idx_alarms_state ON alarms (domain_id, rule_id, channel_id, subtopic, client_id, measurement, created_at DESC);",
				},
				Down: []string{
					`DROP TABLE IF EXISTS alarms`,
				},
			},
		},
	}

	alarmsMigration.Migrations = append(alarmsMigration.Migrations, rolesMigration.Migrations...)

	domainsMigration, err := dpostgres.Migration()
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}
	alarmsMigration.Migrations = append(alarmsMigration.Migrations, domainsMigration.Migrations...)

	return alarmsMigration, nil
}
