// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

const primaryKey = "primary_key"

// ErrMigrate indicates error during database migrations.
var ErrMigrate = errors.New("error executing database migrations")

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

	db, err := sqlx.Open("pgx", url)
	if err != nil {
		return nil, err
	}

	if err := migrateDB(db); err != nil {
		mErr, ok := err.(*migrate.TxError)
		if ok && mErr.Migration.Id == primaryKey {
			return db, ErrMigrate
		}
		return nil, err
	}

	return db, nil
}

func migrateDB(db *sqlx.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "certs_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS certs (
						thing_id     TEXT NOT NULL,
						owner_id     TEXT NOT NULL,
						expire       TIMESTAMPTZ NOT NULL,
						serial       TEXT NOT NULL,
						PRIMARY KEY  (thing_id, owner_id, serial)
					);`,
				},
				Down: []string{
					"DROP TABLE IF EXISTS certs;",
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
