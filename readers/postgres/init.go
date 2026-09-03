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

// migrateDB applies the reader's own migrations.
//
// sql-migrate orders migrations lexicographically unless the id starts with a
// number, so the sequence is zero padded: without the padding
// "messages_reader_10" would sort before "messages_reader_2". Keep the same
// width when adding migrations, and never renumber an id that has already
// shipped.
func migrateDB(db *sqlx.DB) error {
	migrations := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				// The Id must not collide with any Id in
				// consumers/writers/postgres's migration set: both packages
				// record into the same gorp_migrations table, so a
				// reader-applied Id would shadow the writer's same-Id migration
				// and the writer would silently skip it. The "messages_reader_"
				// prefix keeps this sequence out of the writer's namespace for
				// good, whichever side connects first.
				//
				// Everything here is IF NOT EXISTS because the writer creates
				// the same table and the same pair of indexes: whichever side
				// applies first, the other's statements are no-ops.
				Id: "messages_reader_0001",
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
					`CREATE INDEX IF NOT EXISTS idx_channel_publisher_device_id ON messages (channel, publisher, device_id)`,
					`CREATE INDEX IF NOT EXISTS idx_channel_device_id_publisher ON messages (channel, device_id, publisher)`,
				},
				Down: []string{
					`DROP INDEX IF EXISTS idx_channel_publisher_device_id`,
					`DROP INDEX IF EXISTS idx_channel_device_id_publisher`,
					"DROP TABLE messages",
				},
			},
		},
	}

	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	return err
}
