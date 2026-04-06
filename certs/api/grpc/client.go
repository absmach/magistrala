// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	grpcCertsV1 "github.com/absmach/magistrala/api/grpc/certs/v1"
	"github.com/absmach/magistrala/certs/api"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const svcName = "certs.ClientService"

type grpcClient struct {
	timeout     time.Duration
	getEntityID endpoint.Endpoint
	revokeCerts endpoint.Endpoint
}

func NewClient(conn *grpc.ClientConn, timeout time.Duration) grpcCertsV1.CertsServiceClient {
	return &grpcClient{
		getEntityID: kitgrpc.NewClient(
			conn,
			svcName,
			"GetEntityID",
			encodeGetEntityIDRequest,
			decodeGetEntityIDResponse,
			grpcCertsV1.EntityRes{},
		).Endpoint(),

		revokeCerts: kitgrpc.NewClient(
			conn,
			svcName,
			"RevokeCerts",
			encodeRevokeCertsRequest,
			decodeRevokeCertsResponse,
			emptypb.Empty{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (c *grpcClient) GetEntityID(ctx context.Context, req *grpcCertsV1.EntityReq, _ ...grpc.CallOption) (*grpcCertsV1.EntityRes, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	res, err := c.getEntityID(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.(*grpcCertsV1.EntityRes), nil
}

func (c *grpcClient) RevokeCerts(ctx context.Context, req *grpcCertsV1.RevokeReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	res, err := c.revokeCerts(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.(*emptypb.Empty), nil
}

func encodeGetEntityIDRequest(_ context.Context, request any) (any, error) {
	req := request.(*grpcCertsV1.EntityReq)
	return &grpcCertsV1.EntityReq{
		SerialNumber: api.NormalizeSerialNumber(req.GetSerialNumber()),
	}, nil
}

func decodeGetEntityIDResponse(_ context.Context, response any) (any, error) {
	res := response.(*grpcCertsV1.EntityRes)
	return &grpcCertsV1.EntityRes{
		EntityId: res.EntityId,
	}, nil
}

func encodeRevokeCertsRequest(_ context.Context, request any) (any, error) {
	req := request.(*grpcCertsV1.RevokeReq)
	return &grpcCertsV1.RevokeReq{
		EntityId: req.GetEntityId(),
	}, nil
}

func decodeRevokeCertsResponse(_ context.Context, response any) (any, error) {
	return &emptypb.Empty{}, nil
}
