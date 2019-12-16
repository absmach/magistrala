// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package opcua

// Message represent an OPC-UA message
type Message struct {
	ServerURI string
	NodeID    string
	Type      string
	Time      int64
	Data      interface{}
}
