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

func TestDevicesCmdUsageOnMissingArgs(t *testing.T) {
	newFakeAtom(t)
	rootCmd := setFlags(cli.NewDevicesCmd())

	out := executeCommand(t, rootCmd, "device-1", updateCmd)

	assert.Contains(t, out, "usage")
}
