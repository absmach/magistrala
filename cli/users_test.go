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

func TestUsersCreateCmd(t *testing.T) {
	fa := newFakeAtom(t)
	rootCmd := setFlags(cli.NewUsersCmd())

	userJSON := `{"name":"Jane Doe"}`
	out := executeCommand(t, rootCmd, createCmd, userJSON, "workspace-1")

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "human", got.Kind)
	assert.Equal(t, "Jane Doe", got.Name)
	assert.Equal(t, "workspace-1", got.TenantID)

	require.Len(t, fa.requests, 1)
	input, _ := fa.requests[0].Variables["input"].(map[string]any)
	assert.Equal(t, "human", input["kind"])
}

// TestUsersCreateCmdNoWorkspace pins that a user can be created with no
// workspace membership at all -- unlike a device, a human need not belong
// to a tenant (a self-registered account starts this way too).
func TestUsersCreateCmdNoWorkspace(t *testing.T) {
	newFakeAtom(t)
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, createCmd, `{"name":"Jane Doe"}`)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "human", got.Kind)
	assert.Equal(t, "", got.TenantID)
}

func TestUsersGetCmd(t *testing.T) {
	newFakeAtom(t, atom.Entity{ID: "user-1", Kind: "human", Name: "Jane Doe", TenantID: "workspace-1"})
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, "user-1", getCmd)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "user-1", got.ID)
	assert.Equal(t, "Jane Doe", got.Name)
}

func TestUsersGetAllCmd(t *testing.T) {
	newFakeAtom(t,
		atom.Entity{ID: "user-1", Kind: "human", TenantID: "workspace-1"},
		atom.Entity{ID: "user-2", Kind: "human", TenantID: "workspace-1"},
		atom.Entity{ID: "user-3", Kind: "human", TenantID: "workspace-2"},
		atom.Entity{ID: "device-1", Kind: "device", TenantID: "workspace-1"},
	)
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, allCmd, getCmd, "workspace-1")

	var got atom.EntityList
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, uint64(2), got.Total)
}

// TestUsersGetAllCmdNoWorkspace pins that "all get" with no workspace_id
// lists across every tenant, rather than requiring one the way devices does.
func TestUsersGetAllCmdNoWorkspace(t *testing.T) {
	newFakeAtom(t,
		atom.Entity{ID: "user-1", Kind: "human", TenantID: "workspace-1"},
		atom.Entity{ID: "user-2", Kind: "human", TenantID: "workspace-2"},
	)
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, allCmd, getCmd)

	var got atom.EntityList
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, uint64(2), got.Total)
}

func TestUsersUpdateCmd(t *testing.T) {
	newFakeAtom(t, atom.Entity{ID: "user-1", Kind: "human", Name: "old-name", TenantID: "workspace-1"})
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, "user-1", updateCmd, `{"name":"new-name"}`)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "new-name", got.Name)
}

func TestUsersDeleteCmd(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "user-1", Kind: "human"})
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, "user-1", deleteCmd)

	assert.Contains(t, out, "ok")
	_, exists := fa.entities["user-1"]
	assert.False(t, exists)
}

// TestUsersEnableCmd pins the same trap as devices.go's equivalent test:
// UpdateEntity's status wire value is Atom's vocabulary (active/inactive).
func TestUsersEnableCmd(t *testing.T) {
	newFakeAtom(t, atom.Entity{ID: "user-1", Kind: "human", Status: "inactive"})
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, "user-1", enableCmd)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "active", got.Status)
}

func TestUsersDisableCmd(t *testing.T) {
	newFakeAtom(t, atom.Entity{ID: "user-1", Kind: "human", Status: "active"})
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, "user-1", disableCmd)

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "inactive", got.Status)
}

func TestUsersPasswordSetCmd(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "user-1", Kind: "human"})
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, "user-1", "password", "set", "new-secret")

	assert.Contains(t, out, "ok")
	assert.Equal(t, "new-secret", fa.passwords["user-1"])
}

func TestUsersPasswordSetCmdUnknownUser(t *testing.T) {
	newFakeAtom(t)
	rootCmd := setFlags(cli.NewUsersCmd())

	out := executeCommand(t, rootCmd, "missing-user", "password", "set", "new-secret")

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "entity not found")
}
