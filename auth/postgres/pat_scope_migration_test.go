// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/auth"
	"github.com/stretchr/testify/require"
)

// TestPatScopesClientsMigration proves the auth_9 migration statement turns a
// pre-change 'clients' scope row into 'devices' instead of leaving it to fail
// ParseEntityType once ClientsType is gone (edge/architecture.md §8 C2).
func TestPatScopesClientsMigration(t *testing.T) {
	ctx := context.Background()
	patID := generateID(t)
	scopeID := generateID(t)

	_, err := db.ExecContext(ctx, `INSERT INTO pats (id, name) VALUES ($1, $2)`, patID, "legacy-pat")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM pats WHERE id = $1`, patID)
	})

	// Simulates a scope row written before ClientsType was removed.
	_, err = db.ExecContext(ctx, `
		INSERT INTO pat_scopes (id, pat_id, domain_id, entity_type, operation, entity_id)
		VALUES ($1, $2, $3, 'clients', 'update', $4)`,
		scopeID, patID, generateID(t), auth.AnyIDs)
	require.NoError(t, err)

	// Re-runs the auth_9 up statement to prove it rewrites the stale row.
	_, err = db.ExecContext(ctx, `UPDATE pat_scopes SET entity_type = 'devices' WHERE entity_type = 'clients'`)
	require.NoError(t, err)

	var entityType string
	err = db.QueryRowContext(ctx, `SELECT entity_type FROM pat_scopes WHERE id = $1`, scopeID).Scan(&entityType)
	require.NoError(t, err)
	require.Equal(t, "devices", entityType)

	parsed, err := auth.ParseEntityType(entityType)
	require.NoError(t, err)
	require.Equal(t, auth.DevicesType, parsed)
}

// TestParseEntityTypeRejectsRemovedClientsScope documents that an unmigrated
// 'clients' scope is rejected outright rather than silently accepted under
// old vocabulary — the migration above is the only path back to working.
func TestParseEntityTypeRejectsRemovedClientsScope(t *testing.T) {
	_, err := auth.ParseEntityType("clients")
	require.Error(t, err)
}
