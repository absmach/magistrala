// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/consumers/writers/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeviceIDColumnIsUsableAfterMigration runs the real migration set against
// a real database and writes through the resulting table. "no device" has to
// stay a first-class value rather than a NULL that breaks the reader's
// plain-string scan, and device-attributed rows have to be filterable.
func TestDeviceIDColumnIsUsableAfterMigration(t *testing.T) {
	fresh := freshDatabase(t, "device_id_migration_test")
	defer func() {
		_ = fresh.Close()
	}()

	_, err := migrate.Exec(fresh.DB, "postgres", postgres.Migration(), migrate.Up)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))

	// Rows written without naming device_id at all, one of each shape a
	// device-less writer produces.
	seed := `INSERT INTO messages (id, channel, subtopic, publisher, protocol, name, unit, value, time, update_time)
		VALUES (gen_random_uuid(), gen_random_uuid(), 'sub', gen_random_uuid(), 'mqtt', $1, 'V', 1.5, $2, 0)`
	for i := 0; i < 3; i++ {
		_, err = fresh.Exec(seed, fmt.Sprintf("legacy-%d", i), float64(1700000000+i))
		require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	}

	// Scanning into a plain string is what the readers do; a NULL here would
	// fail at runtime instead of at migration time.
	var deviceIDs []string
	require.Nil(t, fresh.Select(&deviceIDs, `SELECT device_id FROM messages ORDER BY name`))
	assert.Equal(t, []string{"", "", ""}, deviceIDs)

	var empties int
	require.Nil(t, fresh.Get(&empties, `SELECT COUNT(*) FROM messages WHERE device_id = ''`))
	assert.Equal(t, 3, empties, "device-less rows are not queryable as device-less")

	// The table accepts device-attributed writes and filters on them.
	_, err = fresh.Exec(`INSERT INTO messages (id, channel, subtopic, publisher, protocol, name, time, device_id)
		VALUES (gen_random_uuid(), gen_random_uuid(), 'sub', gen_random_uuid(), 'mqtt', 'new', 1700000009, 'Meter.A-01:X')`)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))

	var matched int
	require.Nil(t, fresh.Get(&matched, `SELECT COUNT(*) FROM messages WHERE device_id = ANY($1)`, []string{"Meter.A-01:X"}))
	assert.Equal(t, 1, matched)
}

func freshDatabase(t *testing.T, name string) *sqlx.DB {
	t.Helper()

	_, err := db.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, name))
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE %s`, name))
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))

	url := fmt.Sprintf("host=localhost port=%s user=test dbname=%s password=test sslmode=disable", dbPort, name)
	fresh, err := sqlx.Open("pgx", url)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	require.Nil(t, fresh.Ping())

	return fresh
}
