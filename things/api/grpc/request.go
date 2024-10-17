// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type authorizeReq struct {
	ThingID    string
	ThingKey   string
	ChannelID  string
	Permission string
}
