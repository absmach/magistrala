// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type authenticateRes struct {
	authenticated bool
	id            string
}

type authorizeRes struct {
	authorized bool
}
