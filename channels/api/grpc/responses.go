// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type authorizeRes struct {
	authorized bool
}

type removeClientConnectionsRes struct{}

type unsetParentGroupFromChannelsRes struct{}
