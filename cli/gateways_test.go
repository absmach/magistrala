// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"testing"

	"github.com/absmach/magistrala/cli"
	"github.com/absmach/magistrala/pkg/atom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewaysSetCmd(t *testing.T) {
	fa := newFakeAtom(t,
		atom.Entity{ID: "device-1", Kind: "device", Attributes: atom.Attributes{"gateways": []string{"gw-old"}}},
		atom.Entity{ID: "gw-old", Kind: "device", Attributes: atom.Attributes{"is_gateway": true}},
	)
	rootCmd := setFlags(cli.NewGatewaysCmd())

	out := executeCommand(t, rootCmd, "set", "device-1", "gw-1,gw-2")

	assert.Contains(t, out, "ok")
	gws := toStringSlice(fa.entities["device-1"].Attributes["gateways"])
	assert.ElementsMatch(t, []string{"gw-1", "gw-2"}, gws)
}

func TestGatewaysSetCmdClearsList(t *testing.T) {
	// gw-old must resolve as a live entity: DeviceGateways silently drops
	// stale references (pkg/atom/devices.go), and an unseeded ID here would
	// make the CLI's "current" disagree with the raw stored value, always
	// conflicting for the wrong reason.
	fa := newFakeAtom(t,
		atom.Entity{ID: "device-1", Kind: "device", Attributes: atom.Attributes{"gateways": []string{"gw-old"}}},
		atom.Entity{ID: "gw-old", Kind: "device", Attributes: atom.Attributes{"is_gateway": true}},
	)
	rootCmd := setFlags(cli.NewGatewaysCmd())

	out := executeCommand(t, rootCmd, "set", "device-1", "")

	assert.Contains(t, out, "ok")
	gws := toStringSlice(fa.entities["device-1"].Attributes["gateways"])
	assert.Empty(t, gws)
}

// TestGatewaysSetCmdConflict simulates a concurrent write landing between
// the CLI's read and write, and asserts it surfaces as a clear error rather
// than silently clobbering (pkg/atom/devices.go's concurrency contract).
func TestGatewaysSetCmdConflict(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", Attributes: atom.Attributes{"gateways": []string{"gw-old"}}})
	fa.mutateOnNthGet("device-1", 2, atom.Attributes{"gateways": []string{"gw-changed"}})

	rootCmd := setFlags(cli.NewGatewaysCmd())
	out := executeCommand(t, rootCmd, "set", "device-1", "gw-1,gw-2")

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "gateways changed")

	// The write must not have happened.
	gws := toStringSlice(fa.entities["device-1"].Attributes["gateways"])
	assert.Equal(t, []string{"gw-changed"}, gws)
}

func TestGatewaysDevicesCmd(t *testing.T) {
	newFakeAtom(t,
		atom.Entity{ID: "device-1", Kind: "device", TenantID: "domain-1", Attributes: atom.Attributes{"gateways": []string{"gw-1"}}},
		atom.Entity{ID: "device-2", Kind: "device", TenantID: "domain-1", Attributes: atom.Attributes{"gateways": []string{"gw-2"}}},
		atom.Entity{ID: "device-3", Kind: "device", TenantID: "domain-1", Attributes: atom.Attributes{"gateways": []string{"gw-1", "gw-2"}}},
	)
	rootCmd := setFlags(cli.NewGatewaysCmd())

	out := executeCommand(t, rootCmd, "gw-1", "devices", "domain-1")

	var got atom.EntityList
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, uint64(2), got.Total)

	ids := map[string]bool{}
	for _, item := range got.Items {
		ids[item.ID] = true
	}
	assert.True(t, ids["device-1"])
	assert.True(t, ids["device-3"])
	assert.False(t, ids["device-2"])
}
