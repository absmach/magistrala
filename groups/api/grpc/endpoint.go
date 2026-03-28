// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	groups "github.com/absmach/supermq/groups/private"
	"github.com/go-kit/kit/endpoint"
)

func retrieveEntityEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(retrieveEntityReq)
		group, err := svc.RetrieveById(ctx, req.Id)
		if err != nil {
			return retrieveEntityRes{}, err
		}

		return retrieveEntityRes{id: group.ID, domain: group.Domain, parentGroup: group.Parent, status: uint8(group.Status)}, nil
	}
}

func deleteDomainGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deleteDomainGroupsReq)
		if err := req.validate(); err != nil {
			return deleteDomainGroupsRes{}, err
		}

		err := svc.DeleteDomainGroups(ctx, req.domainID)
		if err != nil {
			return deleteDomainGroupsRes{}, err
		}

		return deleteDomainGroupsRes{deleted: true}, nil
	}
}
