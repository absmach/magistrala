// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/policies"
)

var errDomainID = errors.New("domain id required for users")

type authorizeReq struct {
	domainID   string
	channelID  string
	clientID   string
	clientType string
	connType   connections.ConnType
}

func (req authorizeReq) validate() error {
	if req.clientType == policies.UserType && req.domainID == "" {
		return errDomainID
	}
	return nil
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

type retrieveIDByRouteReq struct {
	route    string
	domainID string
}

func (req retrieveIDByRouteReq) validate() error {
	if req.route == "" {
		return apiutil.ErrMissingRoute
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}

type deleteDomainChannelsReq struct {
	domainID string
}

func (req deleteDomainChannelsReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}
	return nil
}
