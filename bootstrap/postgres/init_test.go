// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/absmach/magistrala/bootstrap/postgres"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type byMigrationID []*migrate.Migration

func (b byMigrationID) Len() int           { return len(b) }
func (b byMigrationID) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byMigrationID) Less(i, j int) bool { return b[i].Less(b[j]) }

// appliedOrder returns migration IDs in the order sql-migrate actually
// executes them, which is not the order they are declared in.
func appliedOrder() []string {
	src := postgres.Migration().Migrations
	sorted := make([]*migrate.Migration, len(src))
	copy(sorted, src)
	sort.Sort(byMigrationID(sorted))

	ids := make([]string, len(sorted))
	for i, m := range sorted {
		ids[i] = m.Id
	}
	return ids
}

func position(t *testing.T, ids []string, id string) int {
	t.Helper()
	for i, got := range ids {
		if got == id {
			return i
		}
	}
	require.Failf(t, "migration not found", "no migration with ID %q", id)
	return -1
}

// "configs_N" IDs do not begin with a digit, so sql-migrate orders them as
// strings rather than numerically. Any migration added as "configs_20" would
// therefore run between configs_2 and configs_3 — before the configs_4..8
// column renames. New migrations use the "configs_z<NN>" scheme instead, and
// this test pins that contract.
func TestMigrationOrdering(t *testing.T) {
	ids := appliedOrder()

	t.Run("legacy sequence sorts lexicographically", func(t *testing.T) {
		assert.Less(t, position(t, ids, "configs_10"), position(t, ids, "configs_2"),
			"configs_10 is expected to sort before configs_2 under string ordering")
	})

	t.Run("configs_9 runs after configs_8 renames client_id to id", func(t *testing.T) {
		assert.Less(t, position(t, ids, "configs_8"), position(t, ids, "configs_9"),
			"configs_9 adds a foreign key onto configs (id), created by configs_8")
	})

	t.Run("configs_9 runs after configs_18 creates bootstrap_challenges", func(t *testing.T) {
		assert.Less(t, position(t, ids, "configs_18"), position(t, ids, "configs_9"),
			"configs_9 adds a foreign key onto bootstrap_challenges, created by configs_18")
	})

	t.Run("z-series runs after every legacy migration", func(t *testing.T) {
		for i, id := range ids {
			if !strings.HasPrefix(id, "configs_z") {
				continue
			}
			for _, later := range ids[i+1:] {
				assert.True(t, strings.HasPrefix(later, "configs_z"),
					"legacy migration %q must not run after z-series migration %q", later, id)
			}
			break
		}
	})

	t.Run("no migration uses the unsafe configs_2x scheme", func(t *testing.T) {
		for _, id := range ids {
			for _, unsafe := range []string{"configs_20", "configs_21", "configs_22", "configs_23"} {
				assert.NotEqual(t, unsafe, id,
					"%q sorts before configs_3; use the configs_z<NN> scheme instead", id)
			}
		}
	})

	t.Run("z-series IDs are unique and zero padded", func(t *testing.T) {
		seen := map[string]bool{}
		for _, id := range ids {
			if !strings.HasPrefix(id, "configs_z") {
				continue
			}
			assert.False(t, seen[id], "duplicate migration ID %q", id)
			seen[id] = true
			assert.Regexp(t, `^configs_z\d{2}$`, id,
				"z-series IDs are zero padded so they keep sorting in sequence")
		}
	})
}
