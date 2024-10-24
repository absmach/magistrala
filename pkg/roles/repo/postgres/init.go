// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

// Migration of Auth service.
func Migration(rolesTableNamePrefix, entityTableName, entityIDColumnName string) (*migrate.MemoryMigrationSource, error) {

	// ToDo: need to add check in database to check table exists and column exits as primary key. For this Migration function need database.
	// ToDo: Add entity name in all table prefix. This helps when all entities uses same database
	// ToDo: Add table name prefix option in service and repo. So each entity can have its own tables in same database
	if entityTableName == "" || entityIDColumnName == "" {
		return nil, fmt.Errorf("invalid entity Table Name or column name")
	}

	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: fmt.Sprintf("%s_roles_1", rolesTableNamePrefix),
				Up: []string{
					fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s_roles (
                        id          VARCHAR(254) NOT NULL PRIMARY KEY,
                        name        varchar(200) NOT NULL,
                        entity_id   VARCHAR(36)  NOT NULL,
						created_at  TIMESTAMP,
						updated_at  TIMESTAMP,
						updated_by  VARCHAR(254),
						created_by  VARCHAR(254),
                        CONSTRAINT  unique_role_name_entity_id_constraint UNIQUE ( name, entity_id),
						CONSTRAINT  fk_entity_id FOREIGN KEY(entity_id) REFERENCES %s(%s) ON DELETE CASCADE
                    );`, rolesTableNamePrefix, entityTableName, entityIDColumnName),

					fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s_role_actions (
                        role_id     VARCHAR(254) NOT NULL,
                        action   VARCHAR(254) NOT NULL,
                        CONSTRAINT  unique_domain_role_action_constraint UNIQUE ( role_id, action),
                        CONSTRAINT  fk_%s_roles_id FOREIGN KEY(role_id) REFERENCES %s_roles(id) ON DELETE CASCADE

                    );`, rolesTableNamePrefix, rolesTableNamePrefix, rolesTableNamePrefix),

					fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s_role_members (
                        role_id     VARCHAR(254) NOT NULL,
                        member_id   VARCHAR(254) NOT NULL,
                        CONSTRAINT  unique_role_member_constraint UNIQUE (role_id, member_id),
                        CONSTRAINT  fk_%s_roles_id FOREIGN KEY(role_id) REFERENCES %s_roles(id) ON DELETE CASCADE
                    );`, rolesTableNamePrefix, rolesTableNamePrefix, rolesTableNamePrefix),
				},
				Down: []string{
					fmt.Sprintf(`DROP TABLE IF EXISTS %s_roles`, rolesTableNamePrefix),
					fmt.Sprintf(`DROP TABLE IF EXISTS %s_roles_actions`, rolesTableNamePrefix),
					fmt.Sprintf(`DROP TABLE IF EXISTS %s_roles_members`, rolesTableNamePrefix),
				},
			},
		},
	}, nil
}
