// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type groupBasic struct {
	id     string
	domain string
	status uint8
}

type retrieveEntityRes groupBasic
