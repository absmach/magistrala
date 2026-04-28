// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	rolesPostgres "github.com/absmach/magistrala/pkg/roles/repo/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of Domains service.
func Migration() (*migrate.MemoryMigrationSource, error) {
	rolesMigration, err := rolesPostgres.Migration(rolesTableNamePrefix, entityTableName, entityIDColumnName)
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}

	domainMigrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "domain_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS domains (
                        id          VARCHAR(36) PRIMARY KEY,
                        name        VARCHAR(254),
                        tags        TEXT[],
                        metadata    JSONB,
					    alias       VARCHAR(254) NOT NULL UNIQUE,
                        created_at  TIMESTAMP,
                        updated_at  TIMESTAMP,
                        updated_by  VARCHAR(254),
                        created_by  VARCHAR(254),
                        status      SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0)
                    );`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS domains`,
				},
			},
			{
				Id: "domain_2",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS invitations (
						invited_by       VARCHAR(36) NOT NULL,
						invitee_user_id  VARCHAR(36) NOT NULL,
						domain_id        VARCHAR(36) NOT NULL,
						role_id          VARCHAR(36) NOT NULL,
						created_at       TIMESTAMP NOT NULL,
						updated_at       TIMESTAMP,
						confirmed_at     TIMESTAMP,
						rejected_at      TIMESTAMP,
						UNIQUE (invitee_user_id, domain_id),
						PRIMARY KEY (invitee_user_id, domain_id),
						FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
					);`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS invitations`,
				},
			},
			{
				Id: "domain_3",
				Up: []string{
					`DO $$
						BEGIN
							IF EXISTS (
								SELECT 1
								FROM information_schema.columns
								WHERE table_schema = current_schema()
								AND table_name = 'domains'
								AND column_name = 'alias'
							)
							AND NOT EXISTS (
								SELECT 1
								FROM information_schema.columns
								WHERE table_schema = current_schema()
								AND table_name = 'domains'
								AND column_name = 'route'
							) THEN
								EXECUTE 'ALTER TABLE domains RENAME COLUMN alias TO route;';
							END IF;
						END $$;`,
				},
				Down: []string{
					`DO $$
						BEGIN
							IF EXISTS (
								SELECT 1
								FROM information_schema.columns
								WHERE table_schema = current_schema()
								AND table_name = 'domains'
								AND column_name = 'route'
							)
							AND NOT EXISTS (
								SELECT 1
								FROM information_schema.columns
								WHERE table_schema = current_schema()
								AND table_name = 'domains'
								AND column_name = 'alias'
							) THEN
								EXECUTE 'ALTER TABLE domains RENAME COLUMN route TO alias;';
							END IF;
						END $$;`,
				},
			},
			{
				Id: "domain_4",
				Up: []string{
					`ALTER TABLE domains ALTER COLUMN created_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE domains ALTER COLUMN updated_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE invitations ALTER COLUMN created_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE invitations ALTER COLUMN updated_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE invitations ALTER COLUMN confirmed_at TYPE TIMESTAMPTZ;`,
					`ALTER TABLE invitations ALTER COLUMN rejected_at TYPE TIMESTAMPTZ;`,
				},
				Down: []string{
					`ALTER TABLE domains ALTER COLUMN created_at TYPE TIMESTAMP;`,
					`ALTER TABLE domains ALTER COLUMN updated_at TYPE TIMESTAMP;`,
					`ALTER TABLE invitations ALTER COLUMN created_at TYPE TIMESTAMP;`,
					`ALTER TABLE invitations ALTER COLUMN updated_at TYPE TIMESTAMP;`,
					`ALTER TABLE invitations ALTER COLUMN confirmed_at TYPE TIMESTAMP;`,
					`ALTER TABLE invitations ALTER COLUMN rejected_at TYPE TIMESTAMP;`,
				},
			},
			{
				Id: "domain_5",
				Up: []string{
					`DO $$
					BEGIN
						IF EXISTS (
							SELECT 1
							FROM pg_constraint c
							JOIN pg_class t ON c.conrelid = t.oid
							JOIN pg_namespace n ON n.oid = t.relnamespace
							WHERE t.relname = 'domains'
							AND n.nspname = current_schema()
							AND c.conname = 'domains_alias_key'
						) THEN
							EXECUTE 'ALTER TABLE domains RENAME CONSTRAINT domains_alias_key TO domains_route_key;';
						END IF;
					END $$;`,
				},
				Down: []string{
					`DO $$
					BEGIN
						IF EXISTS (
							SELECT 1
							FROM pg_constraint c
							JOIN pg_class t ON c.conrelid = t.oid
							JOIN pg_namespace n ON n.oid = t.relnamespace
							WHERE t.relname = 'domains'
							AND n.nspname = current_schema()
							AND c.conname = 'domains_route_key'
						) THEN
							EXECUTE 'ALTER TABLE domains RENAME CONSTRAINT domains_route_key TO domains_alias_key;';
						END IF;
					END $$;`,
				},
			},
			{
				Id: "domain_6",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_invitations_invited_by ON invitations(invited_by);`,
					`CREATE INDEX IF NOT EXISTS idx_invitations_role_id ON invitations(role_id);`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_invitations_invited_by;`,
					`DROP INDEX IF EXISTS idx_invitations_role_id;`,
				},
			},
			{
				Id: "domain_7",
				Up: []string{
					`UPDATE domains
					 SET metadata = (COALESCE(metadata, '{}'::jsonb) || COALESCE(metadata->'ui', '{}'::jsonb)) - 'ui'
					 WHERE metadata ? 'ui' AND jsonb_typeof(metadata->'ui') = 'object'`,
				},
				Down: []string{
					`SELECT 1`,
				},
			},
			{
				Id: "domains_roles_4",
				Up: []string{
					`INSERT INTO domains_role_actions (role_id, action)
					 SELECT dr.id, a.action
					 FROM domains_roles dr
					 CROSS JOIN (VALUES
						('rule_create'),
						('rule_read'),
						('rule_update'),
						('rule_delete'),
						('rule_manage_role'),
						('rule_add_role_users'),
						('rule_remove_role_users'),
						('rule_view_role_users'),
						('alarm_update'),
						('alarm_read'),
						('alarm_delete'),
						('alarm_assign'),
						('alarm_acknowledge'),
						('alarm_resolve'),
						('report_create'),
						('report_read'),
						('report_update'),
						('report_delete'),
						('report_manage_role'),
						('report_add_role_users'),
						('report_remove_role_users'),
						('report_view_role_users')
					 ) AS a(action)
					 WHERE dr.name = 'admin'
					 ON CONFLICT DO NOTHING;`,
				},
				Down: []string{
					`DELETE FROM domains_role_actions
					 WHERE action IN (
						'rule_create',
						'rule_read',
						'rule_update',
						'rule_delete',
						'rule_manage_role',
						'rule_add_role_users',
						'rule_remove_role_users',
						'rule_view_role_users',
						'alarm_update',
						'alarm_read',
						'alarm_delete',
						'alarm_assign',
						'alarm_acknowledge',
						'alarm_resolve',
						'report_create',
						'report_read',
						'report_update',
						'report_delete',
						'report_manage_role',
						'report_add_role_users',
						'report_remove_role_users',
						'report_view_role_users'
					 )
					 AND role_id IN (SELECT id FROM domains_roles WHERE name = 'admin');`,
				},
			},
		},
	}

	domainMigrations.Migrations = append(domainMigrations.Migrations, rolesMigration.Migrations...)

	return domainMigrations, nil
}
