// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

// Config defines the options that are used when connecting to a PostgreSQL instance
type Config struct {
	Host        string
	Port        string
	User        string
	Pass        string
	Name        string
	SSLMode     string
	SSLCert     string
	SSLKey      string
	SSLRootCert string
}

// Connect creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate failure.
func Connect(cfg Config) (*sqlx.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s", cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert)

	db, err := sqlx.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	if err := migrateDB(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrateDB(db *sqlx.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "auth_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS keys (
						id          VARCHAR(254) NOT NULL,
						type        SMALLINT,
						subject     VARCHAR(254) NOT NULL,
						issuer_id   UUID NOT NULL,
						issued_at   TIMESTAMP NOT NULL,
						expires_at  TIMESTAMP,
						PRIMARY KEY (id, issuer_id)
					)`,
					`CREATE extension LTREE`,
					`CREATE TABLE IF NOT EXISTS group_type (
						id INTEGER UNIQUE NOT NULL,
						name VARCHAR(254) UNIQUE NOT NULL,
						PRIMARY KEY (id)
					)`,
					`CREATE TABLE IF NOT EXISTS groups ( 
						id          VARCHAR(254) UNIQUE NOT NULL,
						parent_id   VARCHAR(254), 
						owner_id    VARCHAR(254),
						name        VARCHAR(254) NOT NULL,
						description VARCHAR(1024),
						metadata    JSONB,
						path        LTREE, 
						type        INTEGER NOT NULL,
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						PRIMARY KEY (owner_id, path),
						FOREIGN KEY (parent_id) REFERENCES groups (id) ON DELETE CASCADE,
						FOREIGN KEY (type) REFERENCES group_type (id)
				   )`,
					`CREATE TABLE IF NOT EXISTS group_relations (
						member_id VARCHAR(254) NOT NULL,
						group_id VARCHAR(254) NOT NULL,
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE,
						PRIMARY KEY (member_id, group_id)
				   )`,
					`CREATE INDEX path_gist_idx ON groups USING GIST (path);`,
					`INSERT INTO group_type (id, name) VALUES (1, 'things')`,
					`INSERT INTO group_type (id, name) VALUES (2, 'users')`,
					`CREATE OR REPLACE FUNCTION inherit_type()
					 RETURNS trigger 
					 LANGUAGE PLPGSQL
					 AS
					 $$
					 BEGIN
					 IF NEW.parent_id IS NULL OR NEW.parent_id = '' THEN
						RETURN NEW;
					 END IF;
					 SELECT type INTO NEW.type FROM groups WHERE id = NEW.parent_id;
					 RETURN NEW;
					 END;
					 $$`,
					`CREATE TRIGGER inherit_type_tr
					 BEFORE INSERT
					 ON groups
					 FOR EACH ROW
					 EXECUTE PROCEDURE inherit_type();`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS keys`,
					`DROP EXTENSION IF EXISTS LTREE`,
					`DROP TABLE IF EXISTS groups`,
					`DROP TABLE IF EXISTS group_type`,
					`DROP TABLE IF EXISTS group_relations`,
					`DROP FUNCTION IF EXISTS inherit_type`,
					`DROP TRIGGER IF EXISTS inherit_type_tr ON groups`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
