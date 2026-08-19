// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/readers/postgres"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestConnectMigratesSelfSufficientlyOnPreExistingDeployment is a regression
// test for N1: messages_5 created its two indexes on device_id without ever
// adding the column itself, trusting consumers/writers/postgres's messages_3
// to have already run. On a fresh database that assumption always holds --
// messages_1 here already declares device_id, so TestMain's own call to
// Connect never exercised the bug. It only shows up on a deployment that was
// already at messages_1 before this PR: that Id is already recorded in
// gorp_migrations, so migrate.Up treats it as applied and never re-runs its
// (now edited) body, leaving device_id genuinely missing when messages_5's
// CREATE INDEX runs.
//
// This reproduces exactly that shape: a database seeded with the pre-PR
// messages_1 result (table with no device_id column, "messages_1" already
// recorded as applied) and nothing else -- the same state a reader
// connecting before the writer has upgraded would see.
func TestConnectMigratesSelfSufficientlyOnPreExistingDeployment(t *testing.T) {
	const dbName = "n1_regression_test"

	_, err := db.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, dbName))
	require.Nil(t, err, "expected no error dropping any stale test db, got %s", err)
	_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE %s`, dbName))
	require.Nil(t, err, "expected no error creating test db, got %s", err)

	url := fmt.Sprintf("host=localhost port=%s user=test dbname=%s password=test sslmode=disable", dbPort, dbName)
	seed, err := sqlx.Open("pgx", url)
	require.Nil(t, err, "expected no error opening seed connection, got %s", err)
	defer seed.Close()

	// The pre-PR shape of messages_1: no device_id column at all.
	_, err = seed.Exec(`CREATE TABLE messages (
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
	)`)
	require.Nil(t, err, "expected no error seeding pre-PR messages table, got %s", err)

	_, err = seed.Exec(`CREATE TABLE gorp_migrations (id VARCHAR(255) NOT NULL PRIMARY KEY, applied_at TIMESTAMP WITH TIME ZONE)`)
	require.Nil(t, err, "expected no error seeding gorp_migrations, got %s", err)
	_, err = seed.Exec(`INSERT INTO gorp_migrations (id, applied_at) VALUES ('messages_1', $1)`, time.Now())
	require.Nil(t, err, "expected no error recording messages_1 as already applied, got %s", err)

	cfg := postgres.Config{
		Host:    "localhost",
		Port:    dbPort,
		User:    "test",
		Pass:    "test",
		Name:    dbName,
		SSLMode: "disable",
	}
	conn, err := postgres.Connect(cfg)
	require.Nil(t, err, "expected messages_5 to add device_id itself rather than depend on messages_1 having run, got %s", err)
	defer conn.Close()

	var hasColumn bool
	err = seed.Get(&hasColumn, `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_name = 'messages' AND column_name = 'device_id'
	)`)
	require.Nil(t, err, "expected no error checking for device_id column, got %s", err)
	require.True(t, hasColumn, "expected messages_5 to have added device_id")
}
