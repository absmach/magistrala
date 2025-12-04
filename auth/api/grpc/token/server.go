// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/auth"
	grpcapi "github.com/absmach/supermq/auth/api/grpc"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
)

var _ grpcTokenV1.TokenServiceServer = (*tokenGrpcServer)(nil)

type tokenGrpcServer struct {
	grpcTokenV1.UnimplementedTokenServiceServer
	issue   kitgrpc.Handler
	refresh kitgrpc.Handler
	revoke  kitgrpc.Handler
}

// NewAuthServer returns new AuthnServiceServer instance.
func NewTokenServer(svc auth.Service) grpcTokenV1.TokenServiceServer {
	return &tokenGrpcServer{
		issue: kitgrpc.NewServer(
			(issueEndpoint(svc)),
			decodeIssueRequest,
			encodeIssueResponse,
		),
		refresh: kitgrpc.NewServer(
			(refreshEndpoint(svc)),
			decodeRefreshRequest,
			encodeIssueResponse,
		),
		revoke: kitgrpc.NewServer(
			(revokeEndpoint(svc)),
			decodeRevokeRequest,
			encodeRevokeResponse,
		),
	}
}

func (s *tokenGrpcServer) Issue(ctx context.Context, req *grpcTokenV1.IssueReq) (*grpcTokenV1.Token, error) {
	_, res, err := s.issue.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}
	return res.(*grpcTokenV1.Token), nil
}

func (s *tokenGrpcServer) Refresh(ctx context.Context, req *grpcTokenV1.RefreshReq) (*grpcTokenV1.Token, error) {
	_, res, err := s.refresh.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}
	return res.(*grpcTokenV1.Token), nil
}

func decodeIssueRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcTokenV1.IssueReq)
	return issueReq{
		userID:   req.GetUserId(),
		userRole: auth.Role(req.GetUserRole()),
		keyType:  auth.KeyType(req.GetType()),
		verified: req.Verified,
	}, nil
}

func decodeRefreshRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcTokenV1.RefreshReq)
	return refreshReq{refreshToken: req.GetRefreshToken(), verified: req.Verified}, nil
}

func encodeIssueResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(issueRes)

	return &grpcTokenV1.Token{
		AccessToken:  res.accessToken,
		RefreshToken: &res.refreshToken,
		AccessType:   res.accessType,
	}, nil
}

func (s *tokenGrpcServer) Revoke(ctx context.Context, req *grpcTokenV1.RevokeReq) (*grpcTokenV1.RevokeRes, error) {
	_, res, err := s.revoke.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}
	return res.(*grpcTokenV1.RevokeRes), nil
}

func decodeRevokeRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcTokenV1.RevokeReq)
	return revokeReq{token: req.GetToken()}, nil
}

func encodeRevokeResponse(_ context.Context, grpcRes any) (any, error) {
	return &grpcTokenV1.RevokeRes{}, nil
}
