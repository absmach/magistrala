// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type authenticateReq struct {
	clientID string
	username string
	password string
}

type authorizeReq struct {
	externalID string
	topic      string
	action     uint8
}
