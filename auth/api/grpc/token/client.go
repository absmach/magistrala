// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"
	"time"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/auth"
	grpcapi "github.com/absmach/supermq/auth/api/grpc"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const tokenSvcName = "token.v1.TokenService"

type tokenGrpcClient struct {
	issue                 endpoint.Endpoint
	refresh               endpoint.Endpoint
	revoke                endpoint.Endpoint
	listUserRefreshTokens endpoint.Endpoint
	timeout               time.Duration
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
		revoke: kitgrpc.NewClient(
			conn,
			tokenSvcName,
			"Revoke",
			encodeRevokeRequest,
			decodeRevokeResponse,
			grpcTokenV1.RevokeRes{},
		).Endpoint(),
		listUserRefreshTokens: kitgrpc.NewClient(
			conn,
			tokenSvcName,
			"ListUserRefreshTokens",
			encodeListUserRefreshTokensRequest,
			decodeListUserRefreshTokensResponse,
			grpcTokenV1.ListUserRefreshTokensRes{},
		).Endpoint(),
		timeout: timeout,
	}
}

func (client tokenGrpcClient) Issue(ctx context.Context, req *grpcTokenV1.IssueReq, _ ...grpc.CallOption) (*grpcTokenV1.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.issue(ctx, issueReq{
		userID:      req.GetUserId(),
		userRole:    auth.Role(req.GetUserRole()),
		keyType:     auth.KeyType(req.GetType()),
		verified:    req.GetVerified(),
		description: req.GetDescription(),
	})
	if err != nil {
		return &grpcTokenV1.Token{}, grpcapi.DecodeError(err)
	}
	return res.(*grpcTokenV1.Token), nil
}

func encodeIssueRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(issueReq)
	return &grpcTokenV1.IssueReq{
		UserId:      req.userID,
		UserRole:    uint32(req.userRole),
		Type:        uint32(req.keyType),
		Verified:    req.verified,
		Description: req.description,
	}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes any) (any, error) {
	return grpcRes, nil
}

func (client tokenGrpcClient) Refresh(ctx context.Context, req *grpcTokenV1.RefreshReq, _ ...grpc.CallOption) (*grpcTokenV1.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.refresh(ctx, refreshReq{refreshToken: req.GetRefreshToken(), verified: req.GetVerified()})
	if err != nil {
		return &grpcTokenV1.Token{}, grpcapi.DecodeError(err)
	}
	return res.(*grpcTokenV1.Token), nil
}

func encodeRefreshRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(refreshReq)
	return &grpcTokenV1.RefreshReq{RefreshToken: req.refreshToken, Verified: req.verified}, nil
}

func decodeRefreshResponse(_ context.Context, grpcRes any) (any, error) {
	return grpcRes, nil
}

func (client tokenGrpcClient) Revoke(ctx context.Context, req *grpcTokenV1.RevokeReq, _ ...grpc.CallOption) (*grpcTokenV1.RevokeRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.revoke(ctx, revokeReq{tokenID: req.GetTokenId()})
	if err != nil {
		return &grpcTokenV1.RevokeRes{}, grpcapi.DecodeError(err)
	}
	return res.(*grpcTokenV1.RevokeRes), nil
}

func encodeRevokeRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(revokeReq)
	return &grpcTokenV1.RevokeReq{TokenId: req.tokenID}, nil
}

func decodeRevokeResponse(_ context.Context, grpcRes any) (any, error) {
	return grpcRes, nil
}

func (client tokenGrpcClient) ListUserRefreshTokens(ctx context.Context, req *grpcTokenV1.ListUserRefreshTokensReq, _ ...grpc.CallOption) (*grpcTokenV1.ListUserRefreshTokensRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.listUserRefreshTokens(ctx, listUserRefreshTokensReq{userID: req.GetUserId()})
	if err != nil {
		return &grpcTokenV1.ListUserRefreshTokensRes{}, grpcapi.DecodeError(err)
	}
	return res.(*grpcTokenV1.ListUserRefreshTokensRes), nil
}

func encodeListUserRefreshTokensRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(listUserRefreshTokensReq)
	return &grpcTokenV1.ListUserRefreshTokensReq{UserId: req.userID}, nil
}

func decodeListUserRefreshTokensResponse(_ context.Context, grpcRes any) (any, error) {
	return grpcRes, nil
}
