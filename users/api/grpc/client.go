//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package grpc

import (
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var _ mainflux.UsersServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	identify endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn) mainflux.UsersServiceClient {
	endpoint := kitgrpc.NewClient(
		conn,
		"mainflux.UsersService",
		"Identify",
		encodeIdentifyRequest,
		decodeIdentifyResponse,
		mainflux.UserID{},
	).Endpoint()

	return &grpcClient{endpoint}
}

func (client grpcClient) Identify(ctx context.Context, token *mainflux.Token, _ ...grpc.CallOption) (*mainflux.UserID, error) {
	res, err := client.identify(ctx, identityReq{token.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.UserID{Value: ir.id}, ir.err
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identityReq)
	return &mainflux.Token{Value: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.UserID)
	return identityRes{res.GetValue(), nil}, nil
}
