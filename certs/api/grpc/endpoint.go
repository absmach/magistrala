// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	grpcCertsV1 "github.com/absmach/magistrala/api/grpc/certs/v1"
	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/pkg/authn"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
	"google.golang.org/protobuf/types/known/emptypb"
)

func getEntityEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(*grpcCertsV1.EntityReq)

		entityID, err := svc.GetEntityID(ctx, req.SerialNumber)
		if err != nil {
			return nil, err
		}

		return &grpcCertsV1.EntityRes{EntityId: entityID}, nil
	}
}

func revokeCertsEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(*grpcCertsV1.RevokeReq)

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
