// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	groups "github.com/absmach/magistrala/groups/private"
	"github.com/go-kit/kit/endpoint"
)

func retrieveEntityEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveEntityReq)
		group, err := svc.RetrieveById(ctx, req.Id)
		if err != nil {
			return retrieveEntityRes{}, err
		}

		return retrieveEntityRes{id: group.ID, domain: group.Domain, parentGroup: group.Parent, status: uint8(group.Status)}, nil
	}
}
