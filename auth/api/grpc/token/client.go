// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"
	"time"

	"github.com/absmach/magistrala/auth"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const tokenSvcName = "token.v1.TokenService"

type tokenGrpcClient struct {
	issue   endpoint.Endpoint
	refresh endpoint.Endpoint
	timeout time.Duration
}

var _ grpcTokenV1.TokenServiceClient = (*tokenGrpcClient)(nil)

// NewAuthClient returns new auth gRPC client instance.
func NewTokenClient(conn *grpc.ClientConn, timeout time.Duration) grpcTokenV1.TokenServiceClient {
	return &tokenGrpcClient{
		issue: kitgrpc.NewClient(
			conn,
			tokenSvcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			grpcTokenV1.Token{},
		).Endpoint(),
		refresh: kitgrpc.NewClient(
			conn,
			tokenSvcName,
			"Refresh",
			encodeRefreshRequest,
			decodeRefreshResponse,
			grpcTokenV1.Token{},
		).Endpoint(),
		timeout: timeout,
	}
}

func (client tokenGrpcClient) Issue(ctx context.Context, req *grpcTokenV1.IssueReq, _ ...grpc.CallOption) (*grpcTokenV1.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.issue(ctx, issueReq{
		userID:  req.GetUserId(),
		keyType: auth.KeyType(req.GetType()),
	})
	if err != nil {
		return &grpcTokenV1.Token{}, grpcapi.DecodeError(err)
	}
	return res.(*grpcTokenV1.Token), nil
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &grpcTokenV1.IssueReq{
		UserId: req.userID,
		Type:   uint32(req.keyType),
	}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client tokenGrpcClient) Refresh(ctx context.Context, req *grpcTokenV1.RefreshReq, _ ...grpc.CallOption) (*grpcTokenV1.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.refresh(ctx, refreshReq{refreshToken: req.GetRefreshToken()})
	if err != nil {
		return &grpcTokenV1.Token{}, grpcapi.DecodeError(err)
	}
	return res.(*grpcTokenV1.Token), nil
}

func encodeRefreshRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(refreshReq)
	return &grpcTokenV1.RefreshReq{RefreshToken: req.refreshToken}, nil
}

func decodeRefreshResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}
