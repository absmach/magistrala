// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/absmach/magistrala/pkg/atom"
	"github.com/spf13/cobra"
)

const (
	usageGatewaysSet     = "cli gateways set <device_id> <gateway_id1,gateway_id2,...>"
	usageGatewaysDevices = "cli gateways <gateway_id> devices <domain_id>"
)

// NewGatewaysCmd is a thin view over devices filtered by is_gateway, not a
// separate entity surface — creating/editing a gateway is "devices" with
// attributes.is_gateway set (spec §8 A12).
func NewGatewaysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateways <set|gateway_id> [operation] [args...]",
		Short: "Device-gateway reachability",
		Long: `Format:
  gateways set <device_id> <gateway_id1,gateway_id2,...>
  gateways <gateway_id> devices <domain_id>

Examples:
  gateways set device-123 gw-1,gw-2
  gateways gw-1 devices domain-1`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if args[0] == "set" {
				handleSetDeviceGateways(cmd, args[1:])
				return
			}

			if len(args) < 2 {
				logUsageCmd(*cmd, usageGatewaysDevices)
				return
			}

			gatewayID := args[0]
			operation := args[1]
			opArgs := args[2:]

			switch operation {
			case "devices":
				handleGatewayDevices(cmd, gatewayID, opArgs)
			default:
				logErrorCmd(*cmd, fmt.Errorf("unknown operation: %s", operation))
			}
		},
	}

	return cmd
}

// handleSetDeviceGateways reads the device's current gateways to use as
// expectedCurrent, so a conflicting concurrent write is reported rather than
// silently clobbered (pkg/atom/devices.go's optimistic-concurrency contract).
func handleSetDeviceGateways(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageGatewaysSet)
		return
	}

	deviceID := args[0]
	var gatewayIDs []string
	if args[1] != "" {
		gatewayIDs = strings.Split(args[1], ",")
	}

	ctx := cmd.Context()
	current, err := atomClient.DeviceGateways(ctx, deviceID)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	if err := atomClient.SetDeviceGateways(ctx, deviceID, gatewayIDs, current); err != nil {
		if errors.Is(err, atom.ErrGatewaysConflict) {
			logErrorCmd(*cmd, fmt.Errorf("gateways changed since last read, re-run to retry: %w", err))
			return
		}
		logErrorCmd(*cmd, err)
		return
	}

	logOKCmd(*cmd)
}

func handleGatewayDevices(cmd *cobra.Command, gatewayID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageGatewaysDevices)
		return
	}

	q := atom.Query{
		TenantID: args[0],
		Limit:    Limit,
		Offset:   Offset,
	}

	l, err := atomClient.GatewayDevices(cmd.Context(), gatewayID, q)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, l)
}
