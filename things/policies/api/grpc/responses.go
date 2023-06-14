// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

type identityRes struct {
	id string
}

type authorizeRes struct {
	thingID    string
	authorized bool
}
