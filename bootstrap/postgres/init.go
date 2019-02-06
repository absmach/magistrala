//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres

import (
	"database/sql"
	"fmt"

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
func Connect(cfg Config) (*sql.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s", cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert)

	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	if err := migrateDB(db); err != nil {
		return nil, err
	}

	return db, nil
}

func migrateDB(db *sql.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "configs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS configs (
						mainflux_thing    TEXT UNIQUE NOT NULL,
						owner             VARCHAR(254),
						name 			  TEXT,
						mainflux_key      CHAR(36) UNIQUE NOT NULL,
						mainflux_channels jsonb,
						external_id       TEXT UNIQUE NOT NULL,
						external_key 	  TEXT NOT NULL,
						content  		  TEXT,
						state             BIGINT NOT NULL,
						PRIMARY KEY (mainflux_thing, external_id)
					)`,
					`CREATE TABLE IF NOT EXISTS unknown_configs (
						external_id       TEXT UNIQUE NOT NULL,
						external_key 	  TEXT NOT NULL,
						PRIMARY KEY (external_id, external_key)
					)`,
				},
				Down: []string{
					"DROP TABLE configs",
					"DROP TABLE unknown_configs",
				},
			},
		},
	}

	_, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	return err
}
