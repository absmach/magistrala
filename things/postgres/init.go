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

// Connect creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate
// failure.
func Connect(host, port, name, user, pass, sslMode string) (*sql.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s", host, port, user, name, pass, sslMode)

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
				Id: "things_1",
				Up: []string{
					`CREATE TABLE IF NOT EXISTS things (
						id      BIGSERIAL,
						owner   VARCHAR(254),
						type    VARCHAR(10) NOT NULL,
						key     CHAR(36) UNIQUE NOT NULL,
						name    TEXT,
						metadata TEXT,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS channels (
						id    BIGSERIAL,
						owner VARCHAR(254),
						name  TEXT,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    BIGINT,
						channel_owner VARCHAR(254),
						thing_id     BIGINT,
						thing_owner  VARCHAR(254),
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
		},
	}

	_, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	return err
}
