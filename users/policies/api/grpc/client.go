// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux/users/policies"
	"google.golang.org/grpc"
)

const svcName = "mainflux.users.policies.AuthService"

var _ policies.AuthServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	authorize endpoint.Endpoint
	identify  endpoint.Endpoint
	timeout   time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, timeout time.Duration) policies.AuthServiceClient {
	return &grpcClient{
		authorize: kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			policies.AuthorizeRes{},
		).Endpoint(),
		identify: kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			policies.IdentifyRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client grpcClient) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()
	areq := authReq{subject: req.GetSubject(), object: req.GetObject(), action: req.GetAction(), entityType: req.GetEntityType()}
	res, err := client.authorize(ctx, areq)
	if err != nil {
		return &policies.AuthorizeRes{}, err
	}

	ares := res.(authorizeRes)
	return &policies.AuthorizeRes{Authorized: ares.authorized}, err
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.AuthorizeRes)
	return authorizeRes{authorized: res.GetAuthorized()}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authReq)
	return &policies.AuthorizeReq{
		Subject:    req.subject,
		Object:     req.object,
		Action:     req.action,
		EntityType: req.entityType,
	}, nil
}

func (client grpcClient) Identify(ctx context.Context, req *policies.IdentifyReq, _ ...grpc.CallOption) (*policies.IdentifyRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	ireq, err := client.identify(ctx, identifyReq{token: req.GetToken()})
	if err != nil {
		return nil, err
	}

	ires := ireq.(identifyRes)
	return &policies.IdentifyRes{Id: ires.id}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identifyReq)
	return &policies.IdentifyReq{Token: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.IdentifyRes)
	return identifyRes{id: res.GetId()}, nil
}
