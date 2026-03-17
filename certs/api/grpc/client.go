// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/certs/api"
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

func NewClient(conn *grpc.ClientConn, timeout time.Duration) certs.CertsServiceClient {
	return &grpcClient{
		getEntityID: kitgrpc.NewClient(
			conn,
			svcName,
			"GetEntityID",
			encodeGetEntityIDRequest,
			decodeGetEntityIDResponse,
			certs.EntityRes{},
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

func (c *grpcClient) GetEntityID(ctx context.Context, req *certs.EntityReq, _ ...grpc.CallOption) (*certs.EntityRes, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	res, err := c.getEntityID(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.(*certs.EntityRes), nil
}

func (c *grpcClient) RevokeCerts(ctx context.Context, req *certs.RevokeReq, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	res, err := c.revokeCerts(ctx, req)
	if err != nil {
		return nil, err
	}
	return res.(*emptypb.Empty), nil
}

func encodeGetEntityIDRequest(_ context.Context, request any) (any, error) {
	req := request.(*certs.EntityReq)
	return &certs.EntityReq{
		SerialNumber: api.NormalizeSerialNumber(req.GetSerialNumber()),
	}, nil
}

func decodeGetEntityIDResponse(_ context.Context, response any) (any, error) {
	res := response.(*certs.EntityRes)
	return &certs.EntityRes{
		EntityId: res.EntityId,
	}, nil
}

func encodeRevokeCertsRequest(_ context.Context, request any) (any, error) {
	req := request.(*certs.RevokeReq)
	return &certs.RevokeReq{
		EntityId: req.GetEntityId(),
	}, nil
}

func decodeRevokeCertsResponse(_ context.Context, response any) (any, error) {
	return &emptypb.Empty{}, nil
}
