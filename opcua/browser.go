// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package opcua

// BrowsedNode represents the details of a browsed OPC-UA node.
type BrowsedNode struct {
	NodeID      string
	DataType    string
	Description string
	Unit        string
	Scale       string
	BrowseName  string
}

// Browser represents the OPC-UA Server Nodes browser.
type Browser interface {
	// Browse availlable Nodes for a given URI.
	Browse(string, string) ([]BrowsedNode, error)
}
