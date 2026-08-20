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

func TestDeviceTypesCreateCmd(t *testing.T) {
	fa := newFakeAtom(t)
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	deviceTypeJSON := `{"key":"thermostat","name":"Thermostat","description":"wall unit"}`
	out := executeCommand(t, rootCmd, createCmd, deviceTypeJSON, "domain-1")

	var got atom.DeviceType
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "thermostat", got.Key)
	assert.Equal(t, "Thermostat", got.Name)
	assert.Equal(t, "domain-1", got.TenantID)

	// Atom stores device types as Profiles narrowed by object_kind/kind; a
	// type created without that narrowing would not come back from a listing.
	require.Len(t, fa.requests, 1)
	input, _ := fa.requests[0].Variables["input"].(map[string]any)
	assert.Equal(t, "entity", input["objectKind"])
	assert.Equal(t, "device", input["kind"])
	assert.Equal(t, "domain-1", input["tenantId"])
}

func TestDeviceTypesGetCmd(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Name: "Thermostat", Status: "active"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", getCmd)

	var got atom.DeviceType
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "device-type-1", got.ID)
	assert.Equal(t, "thermostat", got.Key)
}

func TestDeviceTypesGetAllCmd(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(
		atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"},
		atom.DeviceType{ID: "device-type-2", TenantID: "domain-1", Key: "valve", Status: "active"},
		atom.DeviceType{ID: "device-type-3", TenantID: "domain-2", Key: "meter", Status: "active"},
	)
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, allCmd, getCmd, "domain-1")

	var got atom.DeviceTypeList
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, uint64(2), got.Total)
}

// TestDeviceTypesGetAllRequiresDomain pins the trap: Atom answers a
// tenant-less device type listing with every tenant's types, so the CLI must
// refuse rather than pass the query through.
func TestDeviceTypesGetAllRequiresDomain(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, allCmd, getCmd)

	assert.Contains(t, out, "usage")
	assert.Empty(t, fa.requests, "no listing may reach Atom without a domain")
}

func TestDeviceTypesUpdateCmd(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Name: "Thermostat", Status: "active"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", updateCmd, `{"name":"Thermostat mk2","status":"deprecated"}`)

	var got atom.DeviceType
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "Thermostat mk2", got.Name)
	// Retirement runs through update: Atom has no delete for device types.
	assert.Equal(t, "deprecated", got.Status)
}

func TestDeviceTypesCreateVersionCmd(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	capabilities := `{"capabilities":{"measurements":[{"name":"temperature","type":"number","unit":"Cel","access":"rw","required":true}],"commands":[{"name":"reboot"}]}}`
	out := executeCommand(t, rootCmd, "device-type-1", createVersionCmd, capabilities)

	var got atom.DeviceTypeVersion
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, 1, got.Version, "first version numbers itself")
	assert.Equal(t, "active", got.Status)

	// The capability document round-trips through the stored schemas.
	require.Len(t, got.Capabilities.Measurements, 1)
	assert.Equal(t, "temperature", got.Capabilities.Measurements[0].Name)
	assert.Equal(t, "Cel", got.Capabilities.Measurements[0].Unit)
	require.Len(t, got.Capabilities.Commands, 1)
	assert.Equal(t, "reboot", got.Capabilities.Commands[0].Name)

	properties, _ := got.JSONSchema["properties"].(map[string]any)
	assert.Contains(t, properties, "temperature")
}

// TestDeviceTypesCreateVersionDraftCmd covers the reason the argument is a
// whole version rather than a bare capability document: a version's status is
// fixed at creation, so staging a draft is only possible here.
func TestDeviceTypesCreateVersionDraftCmd(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	body := `{"status":"draft","capabilities":{"measurements":[{"name":"humidity","type":"number"}]}}`
	out := executeCommand(t, rootCmd, "device-type-1", createVersionCmd, body)

	var got atom.DeviceTypeVersion
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "draft", got.Status)
}

func TestDeviceTypesCreateVersionRejectsBadCapabilityCmd(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	body := `{"capabilities":{"measurements":[{"name":"temperature","type":"tesseract"}]}}`
	out := executeCommand(t, rootCmd, "device-type-1", createVersionCmd, body)

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "unsupported value type")
	assert.Empty(t, fa.requests, "an unrenderable capability document never reaches Atom")
}

func TestDeviceTypesVersionsCmd(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"})
	fa.seedDeviceTypeVersion(
		atom.DeviceTypeVersion{ID: "version-2", DeviceTypeID: "device-type-1", Version: 2, Status: "active"},
		atom.DeviceTypeVersion{ID: "version-1", DeviceTypeID: "device-type-1", Version: 1, Status: "deprecated"},
	)
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", versionsCmd)

	var got []atom.DeviceTypeVersion
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Len(t, got, 2)
	assert.Equal(t, 1, got[0].Version, "versions come back oldest first")
	assert.Equal(t, 2, got[1].Version)
}

func TestDeviceTypesActiveVersionCmd(t *testing.T) {
	fa := newFakeAtom(t)
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"})
	fa.seedDeviceTypeVersion(
		atom.DeviceTypeVersion{ID: "version-1", DeviceTypeID: "device-type-1", Version: 1, Status: "active"},
		atom.DeviceTypeVersion{ID: "version-2", DeviceTypeID: "device-type-1", Version: 2, Status: "active"},
		atom.DeviceTypeVersion{ID: "version-3", DeviceTypeID: "device-type-1", Version: 3, Status: "draft"},
	)
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", activeVersionCmd)

	var got atom.DeviceTypeVersion
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	// The draft is higher-numbered but not bindable.
	assert.Equal(t, "version-2", got.ID)
}

func TestDeviceTypesBindCmd(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", TenantID: "domain-1"})
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"})
	fa.seedDeviceTypeVersion(atom.DeviceTypeVersion{ID: "version-1", DeviceTypeID: "device-type-1", Version: 1, Status: "active"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", bindCmd, "device-1", "version-1")

	var got atom.Entity
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "device-type-1", got.DeviceTypeID)
	assert.Equal(t, "version-1", got.DeviceTypeVersionID)
}

// TestDeviceTypesBindRejectsDeprecatedVersion pins what bind buys over an
// ordinary device update: Atom binds a caller-named version whatever its
// status, so deprecating one would not stop new bindings without this check.
func TestDeviceTypesBindRejectsDeprecatedVersion(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", TenantID: "domain-1"})
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"})
	fa.seedDeviceTypeVersion(atom.DeviceTypeVersion{ID: "version-1", DeviceTypeID: "device-type-1", Version: 1, Status: "deprecated"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", bindCmd, "device-1", "version-1")

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "version is not active")

	device := fa.entities["device-1"]
	assert.Empty(t, device.DeviceTypeID, "a refused bind must not write the device")
}

func TestDeviceTypesBindRejectsDeprecatedType(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", TenantID: "domain-1"})
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "deprecated"})
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", bindCmd, "device-1")

	assert.Contains(t, out, "error")
	assert.Contains(t, out, "device type is not active")
}

// TestDeviceTypesBindSurfacesSchemaViolations checks that a schema rejection
// reaches the operator with the field and the constraint intact, rather than
// as the validator's raw sentence.
func TestDeviceTypesBindSurfacesSchemaViolations(t *testing.T) {
	fa := newFakeAtom(t, atom.Entity{ID: "device-1", Kind: "device", TenantID: "domain-1"})
	fa.seedDeviceType(atom.DeviceType{ID: "device-type-1", TenantID: "domain-1", Key: "thermostat", Status: "active"})
	fa.seedDeviceTypeVersion(atom.DeviceTypeVersion{ID: "version-1", DeviceTypeID: "device-type-1", Version: 1, Status: "active"})
	fa.failEntityUpdate("attributes failed profile schema validation: 'temperature' is a required property; Additional properties are not allowed ('colour' was unexpected)")
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", bindCmd, "device-1")

	assert.Contains(t, out, "rejected by the device type schema")
	assert.Contains(t, out, "temperature (required)")
	assert.Contains(t, out, "colour (additionalProperties)")
}

func TestDeviceTypesCmdUsageOnMissingArgs(t *testing.T) {
	newFakeAtom(t)
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", createVersionCmd)

	assert.Contains(t, out, "usage")
}

func TestDeviceTypesCmdUnknownOperation(t *testing.T) {
	newFakeAtom(t)
	rootCmd := setFlags(cli.NewDeviceTypesCmd())

	out := executeCommand(t, rootCmd, "device-type-1", deleteCmd)

	assert.Contains(t, out, "unknown operation: delete")
}
