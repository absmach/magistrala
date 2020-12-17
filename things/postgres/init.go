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
				Id: "things_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS things (
						id       UUID,
						owner    VARCHAR(254),
						key      VARCHAR(4096) UNIQUE NOT NULL,
						name     VARCHAR(1024),
						metadata JSON,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS channels (
						id       UUID,
						owner    VARCHAR(254),
						name     VARCHAR(1024),
						metadata JSON,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    UUID,
						channel_owner VARCHAR(254),
						thing_id      UUID,
						thing_owner   VARCHAR(254),
						FOREIGN KEY (channel_id, channel_owner) REFERENCES channels (id, owner) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (thing_id, thing_owner) REFERENCES things (id, owner) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (channel_id, channel_owner, thing_id, thing_owner)
					)`,
				},
				Down: []string{
					"DROP TABLE connections",
					"DROP TABLE things",
					"DROP TABLE channels",
				},
			},
			{
				Id: "things_2",
				Up: []string{
					`ALTER TABLE IF EXISTS things ALTER COLUMN
					 metadata TYPE JSONB using metadata::text::jsonb`,
				},
			},
			{
				Id: "things_3",
				Up: []string{
					`ALTER TABLE IF EXISTS channels ALTER COLUMN
					 metadata TYPE JSONB using metadata::text::jsonb`,
				},
			},
			{
				Id: "things_4",
				Up: []string{
					`ALTER TABLE IF EXISTS things ADD CONSTRAINT things_id_key UNIQUE (id)`,
					`CREATE extension LTREE`,
					`CREATE TABLE IF NOT EXISTS thing_groups ( 
						id          VARCHAR(254) UNIQUE NOT NULL,
						parent_id   VARCHAR(254), 
						owner_id    UUID,
						name        VARCHAR(254) NOT NULL,
						description VARCHAR(1024),
						metadata    JSONB,
						path        LTREE, 
						created_at  TIMESTAMPTZ,
						updated_at  TIMESTAMPTZ,
						PRIMARY KEY (owner_id, path),
						FOREIGN KEY (parent_id) REFERENCES thing_groups (id) ON DELETE CASCADE ON UPDATE CASCADE
				   )`,
					`CREATE TABLE IF NOT EXISTS thing_group_relations (
						thing_id UUID NOT NULL,
						group_id VARCHAR(254) NOT NULL,
						FOREIGN KEY (thing_id) REFERENCES things (id) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (group_id) REFERENCES thing_groups (id),
						PRIMARY KEY (thing_id, group_id)
				   )`,
					`CREATE INDEX path_gist_idx ON thing_groups USING GIST (path);`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
