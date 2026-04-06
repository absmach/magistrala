// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	domains "github.com/absmach/magistrala/domains/private"
	"github.com/go-kit/kit/endpoint"
)

func deleteUserFromDomainsEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
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

func retrieveStatusEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(retrieveStatusReq)
		if err := req.validate(); err != nil {
			return retrieveStatusRes{}, err
		}

		status, err := svc.RetrieveStatus(ctx, req.ID)
		if err != nil {
			return retrieveStatusRes{}, err
		}

		return retrieveStatusRes{
			status: uint8(status),
		}, nil
	}
}

func retrieveIDByRouteEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(retrieveIDByRouteReq)
		if err := req.validate(); err != nil {
			return retrieveIDByRouteRes{}, err
		}

		id, err := svc.RetrieveIDByRoute(ctx, req.Route)
		if err != nil {
			return retrieveIDByRouteRes{}, err
		}

		return retrieveIDByRouteRes{
			id: id,
		}, nil
	}
}
