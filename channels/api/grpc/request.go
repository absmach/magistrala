// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/absmach/magistrala/pkg/connections"

type authorizeReq struct {
	domainID   string
	channelID  string
	clientID   string
	clientType string
	connType   connections.ConnType
}
type removeClientConnectionsReq struct {
	clientID string
}

type unsetParentGroupFromChannelsReq struct {
	parentGroupID string
}

type retrieveEntityReq struct {
	Id string
}
