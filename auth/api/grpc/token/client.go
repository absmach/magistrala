// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const tokenSvcName = "magistrala.TokenService"

type tokenGrpcClient struct {
	issue   endpoint.Endpoint
	refresh endpoint.Endpoint
	timeout time.Duration
}

var _ magistrala.TokenServiceClient = (*tokenGrpcClient)(nil)

// NewAuthClient returns new auth gRPC client instance.
func NewTokenClient(conn *grpc.ClientConn, timeout time.Duration) magistrala.TokenServiceClient {
	return &tokenGrpcClient{
		issue: kitgrpc.NewClient(
			conn,
			tokenSvcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			magistrala.Token{},
		).Endpoint(),
		refresh: kitgrpc.NewClient(
			conn,
			tokenSvcName,
			"Refresh",
			encodeRefreshRequest,
			decodeRefreshResponse,
			magistrala.Token{},
		).Endpoint(),
		timeout: timeout,
	}
}

func (client tokenGrpcClient) Issue(ctx context.Context, req *magistrala.IssueReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.issue(ctx, issueReq{
		userID:   req.GetUserId(),
		domainID: req.GetDomainId(),
		keyType:  auth.KeyType(req.GetType()),
	})
	if err != nil {
		return &magistrala.Token{}, grpcapi.DecodeError(err)
	}
	return res.(*magistrala.Token), nil
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &magistrala.IssueReq{
		UserId:   req.userID,
		DomainId: &req.domainID,
		Type:     uint32(req.keyType),
	}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client tokenGrpcClient) Refresh(ctx context.Context, req *magistrala.RefreshReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.refresh(ctx, refreshReq{refreshToken: req.GetRefreshToken(), domainID: req.GetDomainId()})
	if err != nil {
		return &magistrala.Token{}, grpcapi.DecodeError(err)
	}
	return res.(*magistrala.Token), nil
}

func encodeRefreshRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(refreshReq)
	return &magistrala.RefreshReq{RefreshToken: req.refreshToken, DomainId: &req.domainID}, nil
}

func decodeRefreshResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}
