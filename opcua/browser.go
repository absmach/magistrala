// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package opcua

// Browser represents the OPC-UA Server Nodes browser.
type Browser interface {
	// Browse availlable Nodes for a given URI.
	Browse(string, string) ([]string, error)
}
