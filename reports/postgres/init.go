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

func Migration() (*migrate.MemoryMigrationSource, error) {
	rolesMigration, err := rolesPostgres.Migration(rolesTableNamePrefix, entityTableName, entityIDColumnName)
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}
	reportsMigration := &migrate.MemoryMigrationSource{
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
			{
				Id: "reports_02",
				Up: []string{
					`ALTER TABLE report_config ADD COLUMN report_template TEXT;`,
				},
				Down: []string{
					`ALTER TABLE report_config DROP COLUMN report_template;`,
				},
			},
			{
				Id: "reports_03",
				Up: []string{
					// Canonicalize legacy report metric subtopics from dot/NATS wildcards
					// to slash/MQTT wildcards.
					`UPDATE report_config AS rc
						SET metrics = COALESCE((
							SELECT jsonb_agg(
								CASE
									WHEN metric.elem ? 'subtopic'
										AND jsonb_typeof(metric.elem->'subtopic') = 'string'
									THEN jsonb_set(
										metric.elem,
										'{subtopic}',
										to_jsonb(REPLACE(REPLACE(REPLACE(metric.elem->>'subtopic', '>', '#'), '*', '+'), '.', '/')),
										false
									)
									ELSE metric.elem
								END
								ORDER BY metric.ord
							)
							FROM jsonb_array_elements(rc.metrics) WITH ORDINALITY AS metric(elem, ord)
						), '[]'::jsonb)
						WHERE jsonb_typeof(rc.metrics) = 'array'`,
				},
				Down: []string{
					`UPDATE report_config AS rc
						SET metrics = COALESCE((
							SELECT jsonb_agg(
								CASE
									WHEN metric.elem ? 'subtopic'
										AND jsonb_typeof(metric.elem->'subtopic') = 'string'
									THEN jsonb_set(
										metric.elem,
										'{subtopic}',
										to_jsonb(REPLACE(REPLACE(REPLACE(metric.elem->>'subtopic', '#', '>'), '+', '*'), '/', '.')),
										false
									)
									ELSE metric.elem
								END
								ORDER BY metric.ord
							)
							FROM jsonb_array_elements(rc.metrics) WITH ORDINALITY AS metric(elem, ord)
						), '[]'::jsonb)
						WHERE jsonb_typeof(rc.metrics) = 'array'`,
				},
			},
		},
	}

	reportsMigration.Migrations = append(reportsMigration.Migrations, rolesMigration.Migrations...)

	domainsMigration, err := dpostgres.Migration()
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}
	reportsMigration.Migrations = append(reportsMigration.Migrations, domainsMigration.Migrations...)

	return reportsMigration, nil
}
