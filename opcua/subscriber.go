// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package opcua

// Subscriber represents the OPC-UA Server client.
type Subscriber interface {
	// Subscribes to given NodeID and receives events.
	Subscribe(Config) error
}
