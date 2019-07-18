//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package grpc

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	opentracing "github.com/opentracing/opentracing-go"

	"github.com/mainflux/mainflux"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var _ mainflux.UsersServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	identify endpoint.Endpoint
	timeout  time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(tracer opentracing.Tracer, conn *grpc.ClientConn, timeout time.Duration) mainflux.UsersServiceClient {
	endpoint := kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
		conn,
		"mainflux.UsersService",
		"Identify",
		encodeIdentifyRequest,
		decodeIdentifyResponse,
		mainflux.UserID{},
	).Endpoint())

	return &grpcClient{
		identify: endpoint,
		timeout:  timeout,
	}
}

func (client grpcClient) Identify(ctx context.Context, token *mainflux.Token, _ ...grpc.CallOption) (*mainflux.UserID, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

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
