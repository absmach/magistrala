// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux/things/policies"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
	"google.golang.org/grpc"
)

const svcName = "mainflux.things.policies.ThingsService"

var _ policies.ThingsServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	authorize endpoint.Endpoint
	identify  endpoint.Endpoint
	timeout   time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, timeout time.Duration) policies.ThingsServiceClient {
	return &grpcClient{
		authorize: otelkit.EndpointMiddleware(otelkit.WithOperation("authorize"))(kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			policies.AuthorizeRes{},
		).Endpoint()),
		identify: otelkit.EndpointMiddleware(otelkit.WithOperation("identify"))(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentityResponse,
			policies.ClientID{},
		).Endpoint()),

		timeout: timeout,
	}
}

func (client grpcClient) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (*policies.AuthorizeRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	areq := authorizeReq{
		entityType: req.GetEntityType(),
		clientID:   req.GetSub(),
		groupID:    req.GetObj(),
		action:     req.GetAct(),
	}
	res, err := client.authorize(ctx, areq)
	if err != nil {
		return nil, err
	}

	ares := res.(authorizeRes)
	return &policies.AuthorizeRes{ThingID: ares.thingID, Authorized: ares.authorized}, nil
}

func (client grpcClient) Identify(ctx context.Context, req *policies.Key, _ ...grpc.CallOption) (*policies.ClientID, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.identify(ctx, identifyReq{key: req.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &policies.ClientID{Value: ir.id}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authorizeReq)
	return &policies.AuthorizeReq{Sub: req.clientID, Obj: req.groupID, Act: req.action, EntityType: req.entityType}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identifyReq)
	return &policies.Key{Value: req.key}, nil
}

func decodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.ClientID)
	return identityRes{id: res.GetValue()}, nil
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.AuthorizeRes)
	return authorizeRes{thingID: res.GetThingID(), authorized: res.GetAuthorized()}, nil
}
