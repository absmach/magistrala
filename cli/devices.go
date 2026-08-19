// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/pkg/atom"
	"github.com/spf13/cobra"
)

// atomKindDevice is the literal Atom wire kind for CreateEntity/ListEntities.
// atom.KindDevice and atom.KindClient are both the Magistrala-internal label
// "client" — entityCreateInput passes Kind through unmodified, so using
// either constant here would send the wrong value to Atom.
const atomKindDevice = "device"

// UpdateEntity passes Status straight to Atom's GraphQL EntityStatus enum
// without going through pkg/atom's enabled/disabled translator, so the wire
// vocabulary here is Atom's (active/inactive), not Magistrala's.
const (
	atomEntityStatusActive   = "active"
	atomEntityStatusInactive = "inactive"
)

const (
	usageDeviceCreate  = "cli devices create <JSON_device> <domain_id>"
	usageDeviceGet     = "cli devices <device_id|all> get <domain_id>"
	usageDeviceUpdate  = "cli devices <device_id> update <JSON_string>"
	usageDeviceDelete  = "cli devices <device_id> delete"
	usageDeviceEnable  = "cli devices <device_id> enable"
	usageDeviceDisable = "cli devices <device_id> disable"
)

func NewDevicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devices <device_id|all|create> [operation] [args...]",
		Short: "Devices management",
		Long: `Format:
  devices create <JSON_device> <domain_id>
  devices <device_id|all> <operation> [args...]

Operations (require device_id/all): get, update, delete, enable, disable

A gateway is a device with attributes.is_gateway set — create or update one
with this command, then use "gateways" to manage its reachability relation.

Examples:
  devices create <JSON_device> <domain_id>
  devices all get <domain_id>
  devices <device_id> get
  devices <device_id> update <JSON_string>
  devices <device_id> delete
  devices <device_id> enable
  devices <device_id> disable`,

		Run: func(cmd *cobra.Command, args []string) {
			if !requireAtomClient(cmd) {
				return
			}
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if args[0] == create {
				handleDeviceCreate(cmd, args[1:])
				return
			}

			if len(args) < 2 {
				logUsageCmd(*cmd, "devices <device_id|all> <get|update|delete|enable|disable> [args...]")
				return
			}

			deviceParams := args[0]
			operation := args[1]
			opArgs := args[2:]

			switch operation {
			case get:
				handleDeviceGet(cmd, deviceParams, opArgs)
			case update:
				handleDeviceUpdate(cmd, deviceParams, opArgs)
			case delete:
				handleDeviceDelete(cmd, deviceParams, opArgs)
			case enable:
				handleDeviceEnable(cmd, deviceParams, opArgs)
			case disable:
				handleDeviceDisable(cmd, deviceParams, opArgs)
			default:
				logErrorCmd(*cmd, fmt.Errorf("unknown operation: %s", operation))
			}
		},
	}

	return cmd
}

func handleDeviceCreate(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDeviceCreate)
		return
	}

	var device atom.Entity
	if err := json.Unmarshal([]byte(args[0]), &device); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	device.Kind = atomKindDevice
	device.TenantID = args[1]

	created, err := atomClient.CreateEntity(cmd.Context(), device)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	markDeclaredGateways(cmd, created)

	logJSONCmd(*cmd, created)
}

// handleDeviceGet's "all" branch lists devices in a domain, mirroring
// channels.go's get-with-all-sentinel shape; a single device ID needs no
// domain argument since GetEntity takes only the entity ID.
func handleDeviceGet(cmd *cobra.Command, deviceID string, args []string) {
	if deviceID == all {
		if len(args) != 1 {
			logUsageCmd(*cmd, usageDeviceGet)
			return
		}

		q := atom.Query{
			Kind:     atomKindDevice,
			TenantID: args[0],
			Q:        Name,
			Limit:    Limit,
			Offset:   Offset,
		}
		l, err := atomClient.ListEntities(cmd.Context(), q)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}

		logJSONCmd(*cmd, l)
		return
	}

	if len(args) != 0 {
		logUsageCmd(*cmd, usageDeviceGet)
		return
	}

	d, err := atomClient.GetEntity(cmd.Context(), deviceID)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, d)
}

func handleDeviceUpdate(cmd *cobra.Command, deviceID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDeviceUpdate)
		return
	}

	var device atom.Entity
	if err := json.Unmarshal([]byte(args[0]), &device); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	updated, err := atomClient.UpdateEntity(cmd.Context(), deviceID, device)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	markDeclaredGateways(cmd, updated)

	logJSONCmd(*cmd, updated)
}

// markDeclaredGateways flags is_gateway on every id the just-written
// entity's gateways attribute names, the same way "gateways set" does via
// SetDeviceGateways -- but this command bypasses SetDeviceGateways
// entirely, unmarshaling arbitrary Entity JSON and writing it straight
// through. Left unpatched, devices <id> update '{"attributes":
// {"gateways":["gw-1"]}}' creates a real relay with gw-1 unflagged: the
// help text above points operators at exactly this command for gateways,
// so it is a reachable path back to Q1's cross-tenant leak, not a
// theoretical one (P3).
//
// Runs after the entity write, against the entity the API just returned --
// not the caller-unmarshaled one -- rather than before it: a device that
// names itself as one of its own gateways then has its own just-written
// attributes marked directly, instead of a snapshot taken before this
// write landed racing it the way SetDeviceGateways's own pre-P2 bug did.
// A marking failure is reported but does not undo the entity write that
// already succeeded -- there is nothing to roll back to, and the operator
// needs to know marking is what failed, not the write.
func markDeclaredGateways(cmd *cobra.Command, entity atom.Entity) {
	gatewayIDs := atom.GatewaysDeclared(entity)
	if len(gatewayIDs) == 0 {
		return
	}
	if err := atomClient.MarkGateways(cmd.Context(), gatewayIDs); err != nil {
		logErrorCmd(*cmd, fmt.Errorf("device saved, but marking its declared gateways failed: %w", err))
	}
}

func handleDeviceDelete(cmd *cobra.Command, deviceID string, args []string) {
	if len(args) != 0 {
		logUsageCmd(*cmd, usageDeviceDelete)
		return
	}

	if err := atomClient.DeleteEntity(cmd.Context(), deviceID); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logOKCmd(*cmd)
}

func handleDeviceEnable(cmd *cobra.Command, deviceID string, args []string) {
	if len(args) != 0 {
		logUsageCmd(*cmd, usageDeviceEnable)
		return
	}

	d, err := atomClient.UpdateEntity(cmd.Context(), deviceID, atom.Entity{Status: atomEntityStatusActive})
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, d)
}

func handleDeviceDisable(cmd *cobra.Command, deviceID string, args []string) {
	if len(args) != 0 {
		logUsageCmd(*cmd, usageDeviceDisable)
		return
	}

	d, err := atomClient.UpdateEntity(cmd.Context(), deviceID, atom.Entity{Status: atomEntityStatusInactive})
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, d)
}
