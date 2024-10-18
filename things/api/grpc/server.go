// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/things"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ magistrala.ThingsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	magistrala.UnimplementedThingsServiceServer
	authorize kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(svc things.Service) magistrala.ThingsServiceServer {
	return &grpcServer{
		authorize: kitgrpc.NewServer(
			(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
	}
}

func (s *grpcServer) Authorize(ctx context.Context, req *magistrala.ThingsAuthzReq) (*magistrala.ThingsAuthzRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.ThingsAuthzRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.ThingsAuthzReq)
	return authorizeReq{
		ThingID:    req.GetThingID(),
		ThingKey:   req.GetThingKey(),
		ChannelID:  req.GetChannelID(),
		Permission: req.GetPermission(),
	}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &magistrala.ThingsAuthzRes{Authorized: res.authorized, Id: res.id}, nil
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
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, mgauth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, svcerr.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
