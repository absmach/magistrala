// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/absmach/magistrala/pkg/atom"
	"github.com/spf13/cobra"
)

// Operations unique to device types. The shared ones (create, get, update,
// all) come from channels.go's block.
const (
	deviceTypeVersions      = "versions"
	deviceTypeCreateVersion = "create-version"
	deviceTypeActiveVersion = "active-version"
	deviceTypeBind          = "bind"
)

const (
	usageDeviceTypeCreate        = "cli devicetypes create <JSON_device_type> <domain_id>"
	usageDeviceTypeGet           = "cli devicetypes <device_type_id|all> get <domain_id>"
	usageDeviceTypeUpdate        = "cli devicetypes <device_type_id> update <JSON_string>"
	usageDeviceTypeVersions      = "cli devicetypes <device_type_id> versions"
	usageDeviceTypeCreateVersion = "cli devicetypes <device_type_id> create-version <JSON_version>"
	usageDeviceTypeActiveVersion = "cli devicetypes <device_type_id> active-version"
	usageDeviceTypeBind          = "cli devicetypes <device_type_id> bind <device_id> [version_id]"
)

// NewDeviceTypesCmd wraps pkg/atom's device type API. There is deliberately no
// delete operation: Atom has no deleteProfile mutation, so a device type is
// retired by setting its status, not removed.
func NewDeviceTypesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devicetypes <device_type_id|all|create> [operation] [args...]",
		Short: "Device types management",
		Long: `Format:
  devicetypes create <JSON_device_type> <domain_id>
  devicetypes <device_type_id|all> <operation> [args...]

Operations (require device_type_id/all): get, update, versions, create-version,
active-version, bind

Listing is always domain-scoped. Atom's device type filter is "this tenant" or
"no filter at all", and no filter at all crosses tenants, so "all get" takes a
domain_id.

There is no delete: status carries retirement instead, and its three values
(active, deprecated, disabled) do not reduce to an enable/disable pair, so it
is set through update:
  devicetypes <device_type_id> update '{"status":"deprecated"}'

A version is immutable once created and its status is fixed at creation —
create it with "status":"draft" to stage one without making it bindable.

"bind" is the checked way to attach a device to a type: it refuses a type or
version that is not active, which writing device_type_id through
"devices <device_id> update" does not.

Examples:
  devicetypes create '{"key":"thermostat","name":"Thermostat"}' <domain_id>
  devicetypes all get <domain_id>
  devicetypes <device_type_id> get
  devicetypes <device_type_id> update '{"name":"Thermostat","status":"deprecated"}'
  devicetypes <device_type_id> versions
  devicetypes <device_type_id> create-version '{"capabilities":{"measurements":[{"name":"temperature","type":"number","unit":"Cel"}]}}'
  devicetypes <device_type_id> active-version
  devicetypes <device_type_id> bind <device_id> <version_id>`,

		Run: func(cmd *cobra.Command, args []string) {
			if !requireAtomClient(cmd) {
				return
			}
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if args[0] == create {
				handleDeviceTypeCreate(cmd, args[1:])
				return
			}

			if len(args) < 2 {
				logUsageCmd(*cmd, "devicetypes <device_type_id|all> <get|update|versions|create-version|active-version|bind> [args...]")
				return
			}

			deviceTypeID := args[0]
			operation := args[1]
			opArgs := args[2:]

			switch operation {
			case get:
				handleDeviceTypeGet(cmd, deviceTypeID, opArgs)
			case update:
				handleDeviceTypeUpdate(cmd, deviceTypeID, opArgs)
			case deviceTypeVersions:
				handleDeviceTypeVersions(cmd, deviceTypeID, opArgs)
			case deviceTypeCreateVersion:
				handleDeviceTypeCreateVersion(cmd, deviceTypeID, opArgs)
			case deviceTypeActiveVersion:
				handleDeviceTypeActiveVersion(cmd, deviceTypeID, opArgs)
			case deviceTypeBind:
				handleDeviceTypeBind(cmd, deviceTypeID, opArgs)
			default:
				logErrorCmd(*cmd, fmt.Errorf("unknown operation: %s", operation))
			}
		},
	}

	return cmd
}

func handleDeviceTypeCreate(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDeviceTypeCreate)
		return
	}

	var deviceType atom.DeviceType
	if err := json.Unmarshal([]byte(args[0]), &deviceType); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	deviceType.TenantID = args[1]

	created, err := atomClient.CreateDeviceType(cmd.Context(), deviceType)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, created)
}

// handleDeviceTypeGet's "all" branch needs the domain argument that a single
// lookup does not: ListDeviceTypes refuses a tenant-less listing, since Atom
// would answer it with every tenant's types.
func handleDeviceTypeGet(cmd *cobra.Command, deviceTypeID string, args []string) {
	if deviceTypeID == all {
		if len(args) != 1 || args[0] == "" {
			logUsageCmd(*cmd, usageDeviceTypeGet)
			return
		}

		q := atom.DeviceTypeQuery{
			TenantID: args[0],
			Status:   Status,
			Limit:    Limit,
			Offset:   Offset,
		}
		l, err := atomClient.ListDeviceTypes(cmd.Context(), q)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}

		logJSONCmd(*cmd, l)
		return
	}

	if len(args) != 0 {
		logUsageCmd(*cmd, usageDeviceTypeGet)
		return
	}

	dt, err := atomClient.GetDeviceType(cmd.Context(), deviceTypeID)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, dt)
}

func handleDeviceTypeUpdate(cmd *cobra.Command, deviceTypeID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDeviceTypeUpdate)
		return
	}

	var deviceType atom.DeviceType
	if err := json.Unmarshal([]byte(args[0]), &deviceType); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	updated, err := atomClient.UpdateDeviceType(cmd.Context(), deviceTypeID, deviceType)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, updated)
}

func handleDeviceTypeVersions(cmd *cobra.Command, deviceTypeID string, args []string) {
	if len(args) != 0 {
		logUsageCmd(*cmd, usageDeviceTypeVersions)
		return
	}

	l, err := atomClient.ListDeviceTypeVersions(cmd.Context(), deviceTypeID)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, l)
}

// handleDeviceTypeCreateVersion takes a whole version rather than a bare
// capability document so the CLI can also stage a draft and pin a version
// number — both are fixed at creation, with no mutation to change them after.
func handleDeviceTypeCreateVersion(cmd *cobra.Command, deviceTypeID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDeviceTypeCreateVersion)
		return
	}

	var version atom.DeviceTypeVersion
	if err := json.Unmarshal([]byte(args[0]), &version); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	created, err := atomClient.CreateDeviceTypeVersion(cmd.Context(), deviceTypeID, version)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, created)
}

// handleDeviceTypeActiveVersion reports the version a bind picks when none is
// named, which is otherwise only visible by reading the whole version list.
func handleDeviceTypeActiveVersion(cmd *cobra.Command, deviceTypeID string, args []string) {
	if len(args) != 0 {
		logUsageCmd(*cmd, usageDeviceTypeActiveVersion)
		return
	}

	version, err := atomClient.ActiveDeviceTypeVersion(cmd.Context(), deviceTypeID)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, version)
}

// handleDeviceTypeBind lives here rather than under "devices" because binding
// is a device type act — pkg/atom keeps BindDeviceType in device_types.go for
// the same reason, and the type and version status checks it performs are what
// separate it from an ordinary device update.
func handleDeviceTypeBind(cmd *cobra.Command, deviceTypeID string, args []string) {
	if len(args) < 1 || len(args) > 2 {
		logUsageCmd(*cmd, usageDeviceTypeBind)
		return
	}

	versionID := ""
	if len(args) == 2 {
		versionID = args[1]
	}

	device, err := atomClient.BindDeviceType(cmd.Context(), args[0], deviceTypeID, versionID)
	if err != nil {
		logDeviceTypeErrorCmd(cmd, err)
		return
	}

	logJSONCmd(*cmd, device)
}

// logDeviceTypeErrorCmd prints err, expanding a device type schema rejection
// into one line per violation. SchemaValidationError's own message lists
// "field: message" pairs but drops the constraint and the expected value,
// which is the part that says what to write instead.
//
// Only bind can raise one: it writes the device through UpdateEntity, and Atom
// validates the attributes already on the device — against the version bound
// before this call, not the one being bound.
func logDeviceTypeErrorCmd(cmd *cobra.Command, err error) {
	schemaErr, ok := atom.AsSchemaValidationError(err)
	if !ok {
		logErrorCmd(*cmd, err)
		return
	}

	var b strings.Builder
	b.WriteString("attributes rejected by the device type schema:")
	for _, violation := range schemaErr.Violations {
		field := violation.Field
		if field == "" {
			field = "<unidentified field>"
		}
		b.WriteString("\n  " + field)
		if violation.Constraint != "" {
			b.WriteString(" (" + violation.Constraint + ")")
		}
		b.WriteString(": " + violation.Message)
		if violation.Expected != "" {
			b.WriteString("; expected " + violation.Expected)
		}
	}

	logErrorCmd(*cmd, errors.New(b.String()))
}
