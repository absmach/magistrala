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
            			device_id     TEXT NOT NULL DEFAULT '',
            			PRIMARY KEY (id)
					)`,
				},
				Down: []string{
					"DROP TABLE messages",
				},
			},
			{
				// Mirrors consumers/writers/postgres's own messages_4: in a
				// deployment where the reader connects before the writer
				// ever does, this is what actually creates the index this
				// package's MG-15 aggregation queries need — the writer's
				// own migration of the same name is a same-ID no-op by the
				// time it runs against a database this one already touched.
				Id: "messages_2",
				Up: []string{
					`CREATE INDEX IF NOT EXISTS idx_channel_publisher_device_id ON messages (channel, publisher, device_id)`,
					`CREATE INDEX IF NOT EXISTS idx_channel_device_id_publisher ON messages (channel, device_id, publisher)`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_channel_publisher_device_id`,
					`DROP INDEX IF EXISTS idx_channel_device_id_publisher`,
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
