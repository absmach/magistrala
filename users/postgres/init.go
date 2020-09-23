// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq" // required for SQL access
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
// unapplied database migrations. A non-nil error is returned to indicate
// failure.
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
				Id: "users_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS users (
					 email    VARCHAR(254) PRIMARY KEY,
					 password CHAR(60)     NOT  NULL
					)`,
				},
				Down: []string{"DROP TABLE users"},
			},
			{
				Id: "users_2",
				Up: []string{
					`ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS metadata JSONB`,
				},
			},
			{
				Id: "users_3",
				Up: []string{
					`CREATE EXTENSION IF NOT EXISTS "pgcrypto";
					 ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS
					 id UUID NOT NULL DEFAULT gen_random_uuid()`,
				},
			},
			{
				Id: "users_4",
				Up: []string{
					`ALTER TABLE IF EXISTS users DROP CONSTRAINT users_pkey`,
					`ALTER TABLE IF EXISTS users ADD CONSTRAINT users_email_key UNIQUE (email)`,
					`ALTER TABLE IF EXISTS users ADD PRIMARY KEY (id)`,
					`CREATE TABLE IF NOT EXISTS groups ( 
					 id          UUID NOT NULL,
					 parent_id   UUID, 
					 owner_id    UUID,
					 name        VARCHAR(254) UNIQUE NOT NULL,
					 description VARCHAR(1024),
					 metadata    JSONB,
					 PRIMARY KEY (id),
					 FOREIGN KEY (parent_id) REFERENCES groups (id)  ON DELETE CASCADE ON UPDATE CASCADE,
					 FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
				)`,
					`CREATE TABLE IF NOT EXISTS group_relations (
					 user_id UUID NOT NULL,
					 group_id UUID NOT NULL,
					 FOREIGN KEY (user_id)  REFERENCES users  (id) ON DELETE CASCADE ON UPDATE CASCADE,
					 FOREIGN KEY (group_id) REFERENCES groups (id) ON DELETE CASCADE ON UPDATE CASCADE,
					 PRIMARY KEY (user_id, group_id)
				)`,
					`ALTER TABLE IF EXISTS users ADD COLUMN IF NOT EXISTS owner_id UUID`,
					`ALTER TABLE IF EXISTS users ADD FOREIGN KEY (owner_id) REFERENCES groups(id)`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
