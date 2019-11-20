// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	kitot "github.com/go-kit/kit/tracing/opentracing"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	mainflux "github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/users"

	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errGrpc = errors.New("GRPC error")
var _ mainflux.UsersServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	handler kitgrpc.Handler
}

// NewServer returns new UsersServiceServer instance.
func NewServer(tracer opentracing.Tracer, svc users.Service) mainflux.UsersServiceServer {
	handler := kitgrpc.NewServer(
		kitot.TraceServer(tracer, "identify")(identifyEndpoint(svc)),
		decodeIdentifyRequest,
		encodeIdentifyResponse,
	)
	return &grpcServer{handler}
}

func (s *grpcServer) Identify(ctx context.Context, token *mainflux.Token) (*mainflux.UserID, error) {
	_, res, err := s.handler.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(errors.Wrap(errGrpc, err))
	}
	return res.(*mainflux.UserID), nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*mainflux.Token)
	return identityReq{req.GetValue()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	if res.err != nil {
		return &mainflux.UserID{Value: res.id}, encodeError(errors.Wrap(errGrpc, res.err))
	}
	return &mainflux.UserID{Value: res.id}, nil

}

func encodeError(err errors.Error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Contains(err, users.ErrMalformedEntity):
		return status.Error(codes.InvalidArgument, "received invalid token request")
	case errors.Contains(err, users.ErrUnauthorizedAccess):
		return status.Error(codes.Unauthenticated, "failed to identify user from token")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
