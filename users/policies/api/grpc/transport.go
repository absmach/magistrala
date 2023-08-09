// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/clients"
	"github.com/mainflux/mainflux/users/policies"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ policies.AuthServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	authorize kitgrpc.Handler
	identify  kitgrpc.Handler
	policies.UnimplementedAuthServiceServer
}

// NewServer returns new AuthServiceServer instance.
func NewServer(csvc clients.Service, psvc policies.Service) policies.AuthServiceServer {
	return &grpcServer{
		authorize: kitgrpc.NewServer(
			authorizeEndpoint(psvc),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		identify: kitgrpc.NewServer(
			identifyEndpoint(csvc),
			decodeIdentifyRequest,
			encodeIdentifyResponse,
		),
	}
}

func (s *grpcServer) Authorize(ctx context.Context, req *policies.AuthorizeReq) (*policies.AuthorizeRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*policies.AuthorizeRes), nil
}

func (s *grpcServer) Identify(ctx context.Context, token *policies.IdentifyReq) (*policies.IdentifyRes, error) {
	_, res, err := s.identify.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*policies.IdentifyRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.AuthorizeReq)
	return authReq{subject: req.GetSubject(), object: req.GetObject(), action: req.GetAction(), entityType: req.GetEntityType()}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &policies.AuthorizeRes{Authorized: res.authorized}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.IdentifyReq)
	return identifyReq{token: req.GetToken()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identifyRes)
	return &policies.IdentifyRes{Id: res.id}, nil
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrInvalidAuthKey),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrMissingPolicySub),
		errors.Contains(err, apiutil.ErrMissingPolicyObj),
		errors.Contains(err, apiutil.ErrMalformedPolicyAct),
		errors.Contains(err, apiutil.ErrMalformedPolicy),
		errors.Contains(err, apiutil.ErrMissingPolicyOwner):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, apiutil.ErrBearerToken):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
