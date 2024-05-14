// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

func Migration() *migrate.MemoryMigrationSource {
	return &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "groups_01",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS groups (
						id			VARCHAR(36) PRIMARY KEY,
						parent_id	VARCHAR(36),
						domain_id	VARCHAR(36) NOT NULL,
						name		VARCHAR(1024) NOT NULL,
						description	VARCHAR(1024),
						metadata	JSONB,
						created_at	TIMESTAMP,
						updated_at	TIMESTAMP,
						updated_by  VARCHAR(254),
						status		SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE		(domain_id, name),
						FOREIGN KEY (parent_id) REFERENCES groups (id) ON DELETE SET NULL
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS groups`,
				},
			},
			{
				Id: "groups_02",
				Up: []string{
					`CREATE EXTENSION IF NOT EXISTS LTREE`,
					`ALTER TABLE groups ADD COLUMN path LTREE`,
					`CREATE INDEX path_gist_idx ON groups USING GIST (path);`,
					`CREATE OR REPLACE FUNCTION inherit_group()
					 RETURNS trigger 
					 LANGUAGE PLPGSQL
					 AS
					 $$
					 BEGIN
					 IF NEW.parent_id IS NULL OR NEW.parent_id = '' THEN
						RETURN NEW;
					 END IF;
					 IF NOT EXISTS (SELECT id FROM groups WHERE id = NEW.parent_id) THEN
						RAISE EXCEPTION 'wrong parent id';
					 END IF;
					 SELECT text2ltree(ltree2text(path) || '.' || NEW.id) INTO NEW.path FROM groups WHERE id = NEW.parent_id;
					 RETURN NEW;
					 END;
					 $$`,
					`CREATE TRIGGER inherit_group_tr
					 BEFORE INSERT
					 ON groups
					 FOR EACH ROW
					 EXECUTE PROCEDURE inherit_group();`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS groups`,
					`DROP EXTENSION IF EXISTS LTREE`,
					`DROP FUNCTION IF EXISTS inherit_group`,
					`DROP TRIGGER IF EXISTS inherit_group_tr ON groups`,
				},
			},
		},
	}
}
