// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type authorizeRes struct {
	id         string
	authorized bool
}

type verifyConnectionsRes struct {
	Status      string             `json:"status"`
	Connections []connectionStatus `json:"connections_status"`
}

type connectionStatus struct {
	ThingId   string `json:"thing_id"`
	ChannelId string `json:"channel_id"`
	Status    string `json:"status"`
}
