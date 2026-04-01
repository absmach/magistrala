// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package messaging

// ClientIdentity returns the transport client identifier carried by the message.
// It falls back to Publisher for backward compatibility with older messages.
func (m *Message) ClientIdentity() string {
	if m == nil {
		return ""
	}
	if clientID := m.GetClientId(); clientID != "" {
		return clientID
	}
	return m.GetPublisher()
}
