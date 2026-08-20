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

const deviceIDMigrationID = "messages_3"

// TestDeviceIDMigrationOnPopulatedTable runs the real migration set in two
// halves against a real database: everything that existed before device_id,
// then rows, then the device_id migration on top. Adding the column must not
// disturb what is already stored, and "no device" has to stay a first-class
// value rather than a NULL that breaks the reader's plain-string scan.
func TestDeviceIDMigrationOnPopulatedTable(t *testing.T) {
	legacy := freshDatabase(t, "device_id_migration_test")
	defer func() {
		_ = legacy.Close()
	}()

	before := migrationsBeforeDeviceID(t)

	_, err := migrate.Exec(legacy.DB, "postgres", &before, migrate.Up)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))

	// Rows written before the column existed, one of each shape the old schema
	// allowed.
	seed := `INSERT INTO messages (id, channel, subtopic, publisher, protocol, name, unit, value, time, update_time)
		VALUES (gen_random_uuid(), gen_random_uuid(), 'sub', gen_random_uuid(), 'mqtt', $1, 'V', 1.5, $2, 0)`
	for i := 0; i < 3; i++ {
		_, err = legacy.Exec(seed, fmt.Sprintf("legacy-%d", i), float64(1700000000+i))
		require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	}

	applied, err := migrate.Exec(legacy.DB, "postgres", postgres.Migration(), migrate.Up)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))
	assert.Equal(t, 2, applied, "expected the device_id migration and its index migration to be outstanding")

	var total int
	require.Nil(t, legacy.Get(&total, `SELECT COUNT(*) FROM messages`))
	assert.Equal(t, 3, total, "migration lost pre-existing rows")

	// Scanning into a plain string is what the readers do; a NULL here would
	// fail at runtime instead of at migration time.
	var deviceIDs []string
	require.Nil(t, legacy.Select(&deviceIDs, `SELECT device_id FROM messages ORDER BY name`))
	assert.Equal(t, []string{"", "", ""}, deviceIDs)

	var empties int
	require.Nil(t, legacy.Get(&empties, `SELECT COUNT(*) FROM messages WHERE device_id = ''`))
	assert.Equal(t, 3, empties, "pre-existing rows are not queryable as device-less")

	// The migrated table accepts device-attributed writes and filters on them.
	_, err = legacy.Exec(`INSERT INTO messages (id, channel, subtopic, publisher, protocol, name, time, device_id)
		VALUES (gen_random_uuid(), gen_random_uuid(), 'sub', gen_random_uuid(), 'mqtt', 'new', 1700000009, 'Meter.A-01:X')`)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s", err))

	var matched int
	require.Nil(t, legacy.Get(&matched, `SELECT COUNT(*) FROM messages WHERE device_id = ANY($1)`, []string{"Meter.A-01:X"}))
	assert.Equal(t, 1, matched)
}

// migrationsBeforeDeviceID is the migration set as it stood before device_id, so
// the device_id migration afterwards runs against a populated table rather than
// an empty one. Anything at-or-after messages_3 belongs to the device_id era:
// messages_4's indexes reference the column messages_3 adds, so it cannot run
// before it.
func migrationsBeforeDeviceID(t *testing.T) migrate.MemoryMigrationSource {
	t.Helper()

	var before migrate.MemoryMigrationSource
	found := false
	for _, m := range postgres.Migration().Migrations {
		if m.Id == deviceIDMigrationID {
			found = true
			continue
		}
		if m.Id > deviceIDMigrationID {
			continue
		}
		before.Migrations = append(before.Migrations, m)
	}
	require.True(t, found, "device_id migration %q not found", deviceIDMigrationID)
	require.NotEmpty(t, before.Migrations)

	return before
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
