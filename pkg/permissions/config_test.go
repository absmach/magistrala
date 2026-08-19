// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package permissions_test

import (
	"testing"

	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/stretchr/testify/require"
)

// permissionsFilePath points at the real deployed config, not a fixture —
// this is what actually ships, so it is what must validate.
const permissionsFilePath = "../../docker/permission.yaml"

func TestParsePermissionsFileValidates(t *testing.T) {
	_, err := permissions.ParsePermissionsFile(permissionsFilePath)
	require.NoError(t, err)
}

func TestDevicesEntityReplacesClients(t *testing.T) {
	cfg, err := permissions.ParsePermissionsFile(permissionsFilePath)
	require.NoError(t, err)

	// "clients" must be gone (acceptance criterion 9).
	_, _, err = cfg.GetEntityPermissions("clients")
	require.Error(t, err)

	ops, roleOps, err := cfg.GetEntityPermissions("devices")
	require.NoError(t, err)
	require.NotEmpty(t, roleOps)

	perm, ok := ops["set_gateways"]
	require.True(t, ok, "devices block must declare set_gateways")
	require.Equal(t, permissions.Permission("set_gateways_permission"), perm)

	// A representative sample of the ordinary CRUD ops carried over unchanged.
	for _, op := range []string{"view", "update", "enable", "disable", "delete"} {
		_, ok := ops[op]
		require.True(t, ok, "devices block must still declare %s", op)
	}
}

func TestNoGatewaysEntityBlock(t *testing.T) {
	cfg, err := permissions.ParsePermissionsFile(permissionsFilePath)
	require.NoError(t, err)

	// A gateway is a device (spec §8 A12) — there must be no separate block.
	_, _, err = cfg.GetEntityPermissions("gateways")
	require.Error(t, err)
}
