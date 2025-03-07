// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	domains "github.com/absmach/supermq/domains/private"
	"github.com/go-kit/kit/endpoint"
)

func deleteUserFromDomainsEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteUserPoliciesReq)
		if err := req.validate(); err != nil {
			return deleteUserRes{}, err
		}

		if err := svc.DeleteUserFromDomains(ctx, req.ID); err != nil {
			return deleteUserRes{}, err
		}

		return deleteUserRes{deleted: true}, nil
	}
}

func retrieveEntityEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveEntityReq)
		if err := req.validate(); err != nil {
			return retrieveEntityRes{}, err
		}

		dom, err := svc.RetrieveEntity(ctx, req.ID)
		if err != nil {
			return retrieveEntityRes{}, err
		}

		return retrieveEntityRes{
			id:     dom.ID,
			status: uint8(dom.Status),
		}, nil
	}
}
