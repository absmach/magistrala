//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package grpc

import (
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ mainflux.ThingsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	canAccess kitgrpc.Handler
	identify  kitgrpc.Handler
}

// NewServer returns new ThingsServiceServer instance.
func NewServer(svc things.Service) mainflux.ThingsServiceServer {
	return &grpcServer{
		canAccess: kitgrpc.NewServer(
			canAccessEndpoint(svc),
			decodeCanAccessRequest,
			encodeIdentityResponse,
		),
		identify: kitgrpc.NewServer(
			identifyEndpoint(svc),
			decodeIdentifyRequest,
			encodeIdentityResponse,
		),
	}
}

func (gs *grpcServer) CanAccess(ctx context.Context, req *mainflux.AccessReq) (*mainflux.ThingID, error) {
	_, res, err := gs.canAccess.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*mainflux.ThingID), nil
}

func (gs *grpcServer) Identify(ctx context.Context, req *mainflux.Token) (*mainflux.ThingID, error) {
	_, res, err := gs.identify.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*mainflux.ThingID), nil
}

func decodeCanAccessRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.AccessReq)
	return accessReq{thingKey: req.GetToken(), chanID: req.GetChanID()}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.Token)
	return identifyReq{key: req.GetValue()}, nil
}

func encodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &mainflux.ThingID{Value: res.id}, encodeError(res.err)
}

func encodeError(err error) error {
	switch err {
	case nil:
		return nil
	case things.ErrMalformedEntity:
		return status.Error(codes.InvalidArgument, "received invalid can access request")
	case things.ErrUnauthorizedAccess:
		return status.Error(codes.PermissionDenied, "missing or invalid credentials provided")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
