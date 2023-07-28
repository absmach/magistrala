// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/clients"
	"github.com/mainflux/mainflux/things/policies"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ policies.AuthServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	authorize kitgrpc.Handler
	identify  kitgrpc.Handler
	policies.UnimplementedAuthServiceServer
}

// NewServer returns new ThingsServiceServer instance.
func NewServer(csvc clients.Service, psvc policies.Service) policies.AuthServiceServer {
	return &grpcServer{
		authorize: kitgrpc.NewServer(
			otelkit.EndpointMiddleware(otelkit.WithOperation("authorize"))(authorizeEndpoint(psvc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		identify: kitgrpc.NewServer(
			otelkit.EndpointMiddleware(otelkit.WithOperation("identify"))(identifyEndpoint(csvc)),
			decodeIdentifyRequest,
			encodeIdentityResponse,
		),
	}
}

func (gs *grpcServer) Authorize(ctx context.Context, req *policies.AuthorizeReq) (*policies.AuthorizeRes, error) {
	_, res, err := gs.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*policies.AuthorizeRes), nil
}

func (gs *grpcServer) Identify(ctx context.Context, req *policies.IdentifyReq) (*policies.IdentifyRes, error) {
	_, res, err := gs.identify.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*policies.IdentifyRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.AuthorizeReq)
	return authorizeReq{subject: req.GetSubject(), object: req.GetObject(), action: req.GetAction(), entityType: req.GetEntityType()}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.IdentifyReq)
	return identifyReq{secret: req.GetSecret()}, nil
}

func encodeIdentityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &policies.IdentifyRes{Id: res.id}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &policies.AuthorizeRes{ThingID: res.thingID, Authorized: res.authorized}, nil
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingPolicySub,
		err == apiutil.ErrMissingPolicyObj,
		err == apiutil.ErrMalformedPolicyAct,
		err == apiutil.ErrMalformedPolicy,
		err == apiutil.ErrMissingPolicyOwner,
		err == apiutil.ErrBearerKey:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, errors.ErrNotFound):
		return status.Error(codes.NotFound, "entity does not exist")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
