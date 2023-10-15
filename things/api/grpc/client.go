// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux"
	"google.golang.org/grpc"
)

const svcName = "mainflux.AuthzService"

var _ mainflux.AuthzServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout   time.Duration
	authorize endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, timeout time.Duration) mainflux.AuthzServiceClient {
	return &grpcClient{
		authorize: kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			mainflux.AuthorizeRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client grpcClient) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *mainflux.AuthorizeRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.authorize(ctx, req)
	if err != nil {
		return &mainflux.AuthorizeRes{}, err
	}

	ar := res.(authorizeRes)
	return &mainflux.AuthorizeRes{Authorized: ar.authorized, Id: ar.id}, err
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.AuthorizeRes)
	return authorizeRes{authorized: res.Authorized, id: res.Id}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AuthorizeReq)
	return &mainflux.AuthorizeReq{
		Namespace:   req.GetNamespace(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		SubjectKind: req.GetSubjectKind(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(),
	}, nil
}
