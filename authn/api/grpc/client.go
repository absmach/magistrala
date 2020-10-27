// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

var _ mainflux.AuthNServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	issue    endpoint.Endpoint
	identify endpoint.Endpoint
	timeout  time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(tracer opentracing.Tracer, conn *grpc.ClientConn, timeout time.Duration) mainflux.AuthNServiceClient {
	return &grpcClient{
		issue: kitot.TraceClient(tracer, "issue")(kitgrpc.NewClient(
			conn,
			"mainflux.AuthNService",
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			mainflux.UserIdentity{},
		).Endpoint()),
		identify: kitot.TraceClient(tracer, "identify")(kitgrpc.NewClient(
			conn,
			"mainflux.AuthNService",
			"Identify",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			mainflux.UserIdentity{},
		).Endpoint()),
		timeout: timeout,
	}
}

func (client grpcClient) Issue(ctx context.Context, req *mainflux.IssueReq, _ ...grpc.CallOption) (*mainflux.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.issue(ctx, issueReq{id: req.GetId(), email: req.GetEmail(), keyType: req.Type})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.Token{Value: ir.id}, ir.err
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &mainflux.IssueReq{Id: req.id, Email: req.email, Type: req.keyType}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.UserIdentity)
	return identityRes{id: res.GetId(), email: res.GetEmail(), err: nil}, nil
}

func (client grpcClient) Identify(ctx context.Context, token *mainflux.Token, _ ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.identify(ctx, identityReq{token: token.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.UserIdentity{Id: ir.id, Email: ir.email}, ir.err
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identityReq)
	return &mainflux.Token{Value: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.UserIdentity)
	return identityRes{id: res.GetId(), email: res.GetEmail(), err: nil}, nil
}
