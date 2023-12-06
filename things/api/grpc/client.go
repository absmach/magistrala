// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const svcName = "magistrala.AuthzService"

var _ magistrala.AuthzServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout   time.Duration
	authorize endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, timeout time.Duration) magistrala.AuthzServiceClient {
	return &grpcClient{
		authorize: kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			magistrala.AuthorizeRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client grpcClient) Authorize(ctx context.Context, req *magistrala.AuthorizeReq, _ ...grpc.CallOption) (r *magistrala.AuthorizeRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.authorize(ctx, req)
	if err != nil {
		return &magistrala.AuthorizeRes{}, err
	}

	ar := res.(authorizeRes)
	return &magistrala.AuthorizeRes{Authorized: ar.authorized, Id: ar.id}, err
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.AuthorizeRes)
	return authorizeRes{authorized: res.Authorized, id: res.Id}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.AuthorizeReq)
	return &magistrala.AuthorizeReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		SubjectKind: req.GetSubjectKind(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(),
	}, nil
}
