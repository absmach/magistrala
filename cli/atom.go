// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import "github.com/absmach/magistrala/pkg/atom"

// Keep the Atom client handle in a package-level var, mirroring sdk.go's
// smqsdk.SDK. Whoever restores cmd/cli/main.go wires this up; unused for now.
var atomClient *atom.Client

// SetAtomClient sets the Atom client instance used by devices/gateways commands.
func SetAtomClient(c *atom.Client) {
	atomClient = c
}
