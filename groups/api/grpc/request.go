// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import apiutil "github.com/absmach/supermq/api/http/util"

type retrieveEntityReq struct {
	Id string
}

type deleteDomainGroupsReq struct {
	domainID string
}

func (req deleteDomainGroupsReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}
	return nil
}
