// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"

	"github.com/absmach/magistrala/pkg/atom"
	"github.com/spf13/cobra"
)

// Keep the Atom client handle in a package-level var, mirroring sdk.go's
// smqsdk.SDK. Whoever restores cmd/cli/main.go wires this up; unused for now.
var atomClient *atom.Client

// SetAtomClient sets the Atom client instance used by devices/gateways commands.
func SetAtomClient(c *atom.Client) {
	atomClient = c
}

// requireAtomClient reports a wiring error through logErrorCmd rather than
// letting a nil atomClient reach pkg/atom and panic inside (*Client).graphQL.
// That is what happens today if cmd/cli/main.go (currently missing) is
// restored and registers NewDevicesCmd/NewGatewaysCmd/NewDeviceTypesCmd
// without also calling SetAtomClient first.
func requireAtomClient(cmd *cobra.Command) bool {
	if atomClient != nil {
		return true
	}
	logErrorCmd(*cmd, errors.New("atom client is not configured: call cli.SetAtomClient before running this command"))
	return false
}
