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

func TestDevicesCreateCmd(t *testing.T) {
	fa := newFakeAtom(t)
	rootCmd := setFlags(cli.NewDevicesCmd())

	deviceJSON := `{"name":"pump-1","attributes":{"is_gateway":true}}`
	out := executeCommand(t, rootCmd, createCmd, deviceJSON, "domain-1")

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "device", got.Kind)
	assert.Equal(t, "pump-1", got.Name)
	assert.Equal(t, "domain-1", got.TenantID)
	assert.Equal(t, true, got.Attributes["is_gateway"])

	// The literal wire kind must be "device" — atom.KindDevice/KindClient
	// both equal "client" and would be wrong here (see cli/devices.go).
	require.Len(t, fa.requests, 1)
	input, _ := fa.requests[0].Variables["input"].(map[string]any)
	assert.Equal(t, "device", input["kind"])
}

func TestDevicesGetCmd(t *testing.T) {
	newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", Name: "pump-1", TenantID: "domain-1"})
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", getCmd)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "device-1", got.ID)
	assert.Equal(t, "pump-1", got.Name)
}

func TestDevicesGetAllCmd(t *testing.T) {
	newFakeAtom(t,
		atom.Entity{ID: "device-1", Kind: "device", TenantID: "domain-1"},
		atom.Entity{ID: "device-2", Kind: "device", TenantID: "domain-1"},
		atom.Entity{ID: "device-3", Kind: "device", TenantID: "domain-2"},
	)
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, allCmd, getCmd, "domain-1")

	var got atom.EntityList
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, uint64(2), got.Total)
}

func TestDevicesUpdateCmd(t *testing.T) {
	newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", Name: "old-name", TenantID: "domain-1"})
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", updateCmd, `{"name":"new-name"}`)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "new-name", got.Name)
}

func TestDevicesDeleteCmd(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device"})
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", deleteCmd)

	assert.Contains(t, out, "ok")
	_, exists := fa.entities["device-1"]
	assert.False(t, exists)
}

// TestDevicesEnableCmd pins the trap: UpdateEntity's status wire value is
// Atom's vocabulary (active/inactive), not Magistrala's (enabled/disabled).
func TestDevicesEnableCmd(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", Status: "inactive"})
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", enableCmd)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "active", got.Status)

	require.Len(t, fa.requests, 1)
	input, _ := fa.requests[0].Variables["input"].(map[string]any)
	assert.Equal(t, "active", input["status"], "must send Atom's wire vocabulary, not \"enabled\"")
}

func TestDevicesDisableCmd(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", Status: "active"})
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", disableCmd)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "inactive", got.Status)

	require.Len(t, fa.requests, 1)
	input, _ := fa.requests[0].Variables["input"].(map[string]any)
	assert.Equal(t, "inactive", input["status"], "must send Atom's wire vocabulary, not \"disabled\"")
}

// A regression test for P3: the update command unmarshals arbitrary Entity
// JSON and writes it straight through, bypassing "gateways set" (and its
// SetDeviceGateways-driven flagging) entirely. Declaring a gateways
// attribute through this command must still flag the ids it names.
func TestDevicesUpdateCmdFlagsDeclaredGateways(t *testing.T) {
	fa := newFakeAtom(t,
		atom.Entity{ID: "device-1", Kind: "device", TenantID: "domain-1"},
		atom.Entity{ID: "gw-1", Kind: "device"},
	)
	rootCmd := setFlags(cli.NewDevicesCmd())

	executeCommand(t, rootCmd, "device-1", updateCmd, `{"attributes":{"gateways":["gw-1"]}}`)

	gw := fa.entities["gw-1"]
	assert.Equal(t, true, gw.Attributes["is_gateway"], "expected gw-1 to be flagged is_gateway after being declared, got: %+v", gw.Attributes)
}

// The create command's own path to the same bypass.
func TestDevicesCreateCmdFlagsDeclaredGateways(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "gw-1", Kind: "device"})
	rootCmd := setFlags(cli.NewDevicesCmd())

	executeCommand(t, rootCmd, createCmd, `{"name":"pump-1","attributes":{"gateways":["gw-1"]}}`, "domain-1")

	gw := fa.entities["gw-1"]
	assert.Equal(t, true, gw.Attributes["is_gateway"], "expected gw-1 to be flagged is_gateway after being declared, got: %+v", gw.Attributes)
}

// A regression test for S2: markDeclaredGateways used to only log a
// marking failure and then print the device's success JSON anyway, leaving
// an operator believing a gateway is trusted when it is not. A marking
// failure must now fail the whole command the same way any other error
// does -- reported, with no misleading success output following it.
func TestDevicesCreateCmdFailsWhenMarkingDeclaredGatewaysFails(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "gw-1", Kind: "device"})
	// The device create itself is a mutation CreateEntity, so this only
	// intercepts the mutation UpdateEntity MarkGateways issues afterward
	// to flag gw-1 -- the device write must still succeed.
	fa.failEntityUpdate("boom")
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, createCmd, `{"name":"pump-1","attributes":{"gateways":["gw-1"]}}`, "domain-1")

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "boom")
	// tenant_id only ever appears in the device's own success JSON, never
	// in an error message -- its absence proves logJSONCmd never ran.
	assert.NotContains(t, out, "tenant_id")

	gw := fa.entities["gw-1"]
	_, flagged := gw.Attributes["is_gateway"]
	assert.False(t, flagged, "gateway must not appear flagged when the flagging write itself failed")
}

// A regression test for A15: a device naming itself as one of its own
// gateways is the degenerate case of a gateway chain (self-reference would
// make device-1 both a gateway, via MarkGateways, and a declarer of a
// gateway of its own -- itself). The update must be rejected before it
// reaches Atom, not written through as it was before A15.
func TestDevicesUpdateCmdRejectsSelfReference(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", TenantID: "domain-1"})
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", updateCmd, `{"attributes":{"gateways":["device-1"]}}`)

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "cannot declare its own gateways")
	require.Empty(t, fa.requests, "a rejected write must never reach Atom")
}

// A regression test for A15: the create and update commands unmarshal
// arbitrary Entity JSON and write it straight through, so is_gateway and
// gateways can be set together in one payload -- exactly the combination
// SetDeviceGateways's own ValidateGatewayChain rejects. This command must
// reject it too, before ever calling CreateEntity/UpdateEntity.
func TestDevicesCreateCmdRejectsGatewayDeclaringItsOwnGateways(t *testing.T) {
	fa := newFakeAtom(t)
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, createCmd, `{"name":"pump-1","attributes":{"is_gateway":true,"gateways":["gw-1"]}}`, "domain-1")

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "cannot declare its own gateways")
	require.Empty(t, fa.requests, "a rejected write must never reach Atom")
}

func TestDevicesUpdateCmdRejectsGatewayDeclaringItsOwnGateways(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", Attributes: atom.Attributes{"is_gateway": true}})
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", updateCmd, `{"attributes":{"is_gateway":true,"gateways":["gw-1"]}}`)

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "cannot declare its own gateways")
	require.Empty(t, fa.requests, "a rejected write must never reach Atom")
}

func TestDevicesCmdUsageOnMissingArgs(t *testing.T) {
	newFakeAtom(t)
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", updateCmd)

	assert.Contains(t, out, "usage")
}
