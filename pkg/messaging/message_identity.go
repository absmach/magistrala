// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package messaging

// ClientIdentity returns the authenticated application identity carried by the
// message. FluxMQ stores the protocol connection identifier in client_id and the
// Atom/Magistrala entity identifier in publisher/external_id, so publisher wins
// when both are present.
func (m *Message) ClientIdentity() string {
	if m == nil {
		return ""
	}
	if publisher := m.GetPublisher(); publisher != "" {
		return publisher
	}
	return m.GetClientId()
}
