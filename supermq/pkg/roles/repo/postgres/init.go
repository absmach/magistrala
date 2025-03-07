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
                        CONSTRAINT  %s_roles_unique_role_name_entity_id_constraint UNIQUE (name, entity_id),
						CONSTRAINT  %s_roles_fk_entity_id FOREIGN KEY(entity_id) REFERENCES %s(%s) ON DELETE CASCADE
                    );`, rolesTableNamePrefix, rolesTableNamePrefix, rolesTableNamePrefix, entityTableName, entityIDColumnName),

					fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s_role_actions (
                        role_id     VARCHAR(254) NOT NULL,
                        action      VARCHAR(254) NOT NULL,
                        CONSTRAINT  %s_role_actions_unique_role_action_constraint UNIQUE (role_id, action),
                        CONSTRAINT  %s_role_actions_fk_roles_id FOREIGN KEY(role_id) REFERENCES %s_roles(id) ON DELETE CASCADE
                    );`, rolesTableNamePrefix, rolesTableNamePrefix, rolesTableNamePrefix, rolesTableNamePrefix),

					fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s_role_members (
                        role_id     VARCHAR(254) NOT NULL,
                        member_id   VARCHAR(254) NOT NULL,
                        entity_id   VARCHAR(36)  NOT NULL,
                        CONSTRAINT  %s_role_members_unique_role_member_constraint UNIQUE (role_id, member_id),
                        CONSTRAINT  %s_role_members_unique_entity_member_constraint UNIQUE (member_id, entity_id),
                        CONSTRAINT  %s_role_members_fk_roles_id FOREIGN KEY(role_id) REFERENCES %s_roles(id) ON DELETE CASCADE
                    );`, rolesTableNamePrefix, rolesTableNamePrefix, rolesTableNamePrefix, rolesTableNamePrefix, rolesTableNamePrefix),
				},
				Down: []string{
					fmt.Sprintf(`DROP TABLE IF EXISTS %s_roles`, rolesTableNamePrefix),
					fmt.Sprintf(`DROP TABLE IF EXISTS %s_role_actions`, rolesTableNamePrefix),
					fmt.Sprintf(`DROP TABLE IF EXISTS %s_role_members`, rolesTableNamePrefix),
				},
			},
		},
	}, nil
}
