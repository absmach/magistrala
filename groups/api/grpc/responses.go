// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type groupBasic struct {
	id          string
	domain      string
	parentGroup string
	status      uint8
}

type retrieveEntityRes groupBasic
