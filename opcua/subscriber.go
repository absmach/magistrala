// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package opcua

import "context"

// Subscriber represents the OPC-UA Server client.
type Subscriber interface {
	// Subscribes to given NodeID and receives events.
	Subscribe(context.Context, Config) error
}
