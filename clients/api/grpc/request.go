// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type authenticateReq struct {
	Token string
}

type retrieveEntitiesReq struct {
	Ids []string
}

type retrieveEntityReq struct {
	Id string
}

type removeChannelConnectionsReq struct {
	channelID string
}

type UnsetParentGroupFromClientReq struct {
	parentGroupID string
}

type deleteDomainClientsReq struct {
	domainID string
}
