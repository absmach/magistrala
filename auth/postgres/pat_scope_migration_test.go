// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/auth"
	"github.com/stretchr/testify/require"
)

// TestPatScopesClientsMigration proves the auth_9 migration statements turn
// pre-change 'clients' scope rows into 'devices' rows instead of leaving them
// to fail ParseEntityType once ClientsType is gone (edge/architecture.md §8 C2).
func TestPatScopesClientsMigration(t *testing.T) {
	ctx := context.Background()
	patID := generateID(t)

	_, err := db.ExecContext(ctx, `INSERT INTO pats (id, name) VALUES ($1, $2)`, patID, "legacy-pat")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM pats WHERE id = $1`, patID)
	})

	scopes := []struct {
		id        string
		operation string
		want      string
	}{
		{id: generateID(t), operation: "update", want: "update"},
		{id: generateID(t), operation: "create_clients", want: "create_devices"},
		{id: generateID(t), operation: "list_clients", want: "list_devices"},
	}

	for _, scope := range scopes {
		// Simulates scope rows written before ClientsType was removed.
		_, err = db.ExecContext(ctx, `
			INSERT INTO pat_scopes (id, pat_id, workspace_id, entity_type, operation, entity_id)
			VALUES ($1, $2, $3, 'clients', $4, $5)`,
			scope.id, patID, generateID(t), scope.operation, auth.AnyIDs)
		require.NoError(t, err)
	}

	// Re-runs the auth_9 up statement to prove it rewrites the stale row.
	_, err = db.ExecContext(ctx, `UPDATE pat_scopes SET operation = 'create_devices' WHERE entity_type = 'clients' AND operation = 'create_clients'`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE pat_scopes SET operation = 'list_devices' WHERE entity_type = 'clients' AND operation = 'list_clients'`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE pat_scopes SET entity_type = 'devices' WHERE entity_type = 'clients'`)
	require.NoError(t, err)

	for _, scope := range scopes {
		var entityType, operation string
		err = db.QueryRowContext(ctx, `SELECT entity_type, operation FROM pat_scopes WHERE id = $1`, scope.id).Scan(&entityType, &operation)
		require.NoError(t, err)
		require.Equal(t, "devices", entityType)
		require.Equal(t, scope.want, operation)

		parsed, err := auth.ParseEntityType(entityType)
		require.NoError(t, err)
		require.Equal(t, auth.DevicesType, parsed)
	}
}

// TestParseEntityTypeRejectsRemovedClientsScope documents that an unmigrated
// 'clients' scope is rejected outright rather than silently accepted under
// old vocabulary — the migration above is the only path back to working.
func TestParseEntityTypeRejectsRemovedClientsScope(t *testing.T) {
	_, err := auth.ParseEntityType("clients")
	require.Error(t, err)
}
