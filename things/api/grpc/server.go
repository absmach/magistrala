// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/things"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ magistrala.AuthzServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	magistrala.UnimplementedAuthzServiceServer
	authorize kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(svc things.Service) magistrala.AuthzServiceServer {
	return &grpcServer{
		authorize: kitgrpc.NewServer(
			(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
	}
}

func (s *grpcServer) Authorize(ctx context.Context, req *magistrala.AuthorizeReq) (*magistrala.AuthorizeRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.AuthorizeRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.AuthorizeReq)
	return req, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &magistrala.AuthorizeRes{Authorized: res.authorized, Id: res.id}, nil
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingMemberType,
		err == apiutil.ErrMissingPolicySub,
		err == apiutil.ErrMissingPolicyObj,
		err == apiutil.ErrMalformedPolicyAct:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, auth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
