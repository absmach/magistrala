// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
)

// Table for SenML messages.
const defTable = "messages"

// Config defines the options that are used when connecting to a PostgreSQL instance.
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
		return nil, err
	}

	return db, nil
}

func migrateDB(db *sqlx.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "messages_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS messages (
            			id            UUID,
            			channel       UUID,
            			subtopic      VARCHAR(254),
            			publisher     UUID,
            			protocol      TEXT,
            			name          TEXT,
            			unit          TEXT,
            			value         FLOAT,
            			string_value  TEXT,
            			bool_value    BOOL,
            			data_value    TEXT,
            			sum           FLOAT,
            			time          FlOAT,
            			update_time   FLOAT,
            			PRIMARY KEY (id)
					)`,
				},
				Down: []string{
					"DROP TABLE messages",
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
