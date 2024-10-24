// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	"github.com/absmach/magistrala/auth"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	grpcAuthV1 "github.com/absmach/magistrala/internal/grpc/auth/v1"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
)

var _ grpcAuthV1.AuthServiceServer = (*authGrpcServer)(nil)

type authGrpcServer struct {
	grpcAuthV1.UnimplementedAuthServiceServer
	authorize    kitgrpc.Handler
	authenticate kitgrpc.Handler
}

// NewAuthServer returns new AuthnServiceServer instance.
func NewAuthServer(svc auth.Service) grpcAuthV1.AuthServiceServer {
	return &authGrpcServer{
		authorize: kitgrpc.NewServer(
			(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),

		authenticate: kitgrpc.NewServer(
			(authenticateEndpoint(svc)),
			decodeAuthenticateRequest,
			encodeAuthenticateResponse,
		),
	}
}

func (s *authGrpcServer) Authenticate(ctx context.Context, req *grpcAuthV1.AuthNReq) (*grpcAuthV1.AuthNRes, error) {
	_, res, err := s.authenticate.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}
	return res.(*grpcAuthV1.AuthNRes), nil
}

func (s *authGrpcServer) Authorize(ctx context.Context, req *grpcAuthV1.AuthZReq) (*grpcAuthV1.AuthZRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}
	return res.(*grpcAuthV1.AuthZRes), nil
}

func decodeAuthenticateRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcAuthV1.AuthNReq)
	return authenticateReq{token: req.GetToken()}, nil
}

func encodeAuthenticateResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authenticateRes)
	return &grpcAuthV1.AuthNRes{Id: res.id, UserId: res.userID, DomainId: res.domainID}, nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcAuthV1.AuthZReq)
	return authReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		SubjectKind: req.GetSubjectKind(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(),
	}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &grpcAuthV1.AuthZRes{Authorized: res.authorized, Id: res.id}, nil
}
