// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/pkg/authn"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
	"google.golang.org/protobuf/types/known/emptypb"
)

func getEntityEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(*certs.EntityReq)

		entityID, err := svc.GetEntityID(ctx, req.SerialNumber)
		if err != nil {
			return nil, err
		}

		return &certs.EntityRes{EntityId: entityID}, nil
	}
}

func revokeCertsEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(*certs.RevokeReq)

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		err := svc.RevokeAll(ctx, session, req.EntityId)
		if err != nil {
			return nil, err
		}

		return &emptypb.Empty{}, nil
	}
}
