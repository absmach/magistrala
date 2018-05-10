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
func Connect(host, port, name, user, pass string) (*sql.DB, error) {
	t := "host=%s port=%s user=%s dbname=%s password=%s sslmode=disable"
	url := fmt.Sprintf(t, host, port, user, name, pass)

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
			&migrate.Migration{
				Id: "clients_1",
				Up: []string{
					`CREATE TABLE clients (
						id      CHAR(36),
						owner   VARCHAR(254),
						type    VARCHAR(10) NOT NULL,
						name    TEXT,
						key     TEXT,
						payload TEXT,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE channels (
						id    CHAR(36),
						owner VARCHAR(254),
						name  TEXT,
						PRIMARY KEY (id, owner)
					)`,
					`CREATE TABLE connections (
						channel_id    CHAR(36),
						channel_owner VARCHAR(254),
						client_id     CHAR(36),
						client_owner  VARCHAR(254),
						FOREIGN KEY (channel_id, channel_owner) REFERENCES channels (id, owner) ON DELETE CASCADE ON UPDATE CASCADE,
						FOREIGN KEY (client_id, client_owner) REFERENCES clients (id, owner) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (channel_id, channel_owner, client_id, client_owner)
					)`,
				},
				Down: []string{
					"DROP TABLE connections",
					"DROP TABLE clients",
					"DROP TABLE channels",
				},
			},
		},
	}

	_, err := migrate.Exec(db, "postgres", migrations, migrate.Up)
	return err
}
