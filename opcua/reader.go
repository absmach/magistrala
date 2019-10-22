// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package opcua

// Reader represents the OPC-UA client.
type Reader interface {
	// Read given OPC-UA Server NodeID (Namespace + ID).
	Read(Config) error
}
