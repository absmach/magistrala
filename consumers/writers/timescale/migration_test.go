// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package timescale_test

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/consumers/writers/timescale"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeviceIDColumnIsUsableAfterMigration runs the real migration set against
// a real database and writes through the resulting hypertable, across more than
// one chunk. "no device" has to stay a first-class value rather than a NULL
// that breaks the reader's plain-string scan, and device-attributed rows have
// to be filterable.
func TestDeviceIDColumnIsUsableAfterMigration(t *testing.T) {
	fresh := freshDatabase(t, "device_id_migration_test")
	defer func() {
		_ = fresh.Close()
	}()

	_, err := migrate.Exec(fresh.DB, "postgres", timescale.Migration(), migrate.Up)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))

	// Rows written without naming device_id at all, one of each shape a
	// device-less writer produces.
	// Spread across days so the rows land in more than one chunk.
	seed := `INSERT INTO messages (channel, subtopic, publisher, protocol, name, unit, value, time, update_time)
		VALUES (gen_random_uuid(), 'sub', gen_random_uuid(), 'mqtt', $1, 'V', 1.5, $2, 0)`
	for i := 0; i < 3; i++ {
		_, err = fresh.Exec(seed, fmt.Sprintf("legacy-%d", i), int64(1700000000000000000)+int64(i)*86400000000000)
		require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	}

	var chunks int
	require.Nil(t, fresh.Get(&chunks, `SELECT COUNT(*) FROM timescaledb_information.chunks WHERE hypertable_name = 'messages'`))
	require.Greater(t, chunks, 1, "seed did not span multiple chunks")

	// Scanning into a plain string is what the readers do; a NULL here would
	// fail at runtime instead of at migration time.
	var deviceIDs []string
	require.Nil(t, fresh.Select(&deviceIDs, `SELECT device_id FROM messages ORDER BY name`))
	assert.Equal(t, []string{"", "", ""}, deviceIDs)

	var empties int
	require.Nil(t, fresh.Get(&empties, `SELECT COUNT(*) FROM messages WHERE device_id = ''`))
	assert.Equal(t, 3, empties, "device-less rows are not queryable as device-less")

	// The hypertable accepts device-attributed writes and filters on them.
	_, err = fresh.Exec(`INSERT INTO messages (channel, subtopic, publisher, protocol, name, time, device_id)
		VALUES (gen_random_uuid(), 'sub', gen_random_uuid(), 'mqtt', 'new', 1700000009000000000, 'Meter.A-01:X')`)
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

	_, err = fresh.Exec(`CREATE EXTENSION IF NOT EXISTS timescaledb`)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))

	return fresh
}
