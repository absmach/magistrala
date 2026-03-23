// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	authv1 "github.com/absmach/fluxmq/pkg/proto/auth/v1"
	smqauth "github.com/absmach/supermq/auth"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ authv1.AuthServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	authv1.UnimplementedAuthServiceServer
	authenticate kitgrpc.Handler
	authorize    kitgrpc.Handler
}

// NewServer creates a FluxMQ AuthService gRPC server that bridges to
// SuperMQ's Clients (authn) and Channels (authz) services.
func NewServer(
	clients grpcClientsV1.ClientsServiceClient,
	channels grpcChannelsV1.ChannelsServiceClient,
	parser messaging.TopicParser,
) authv1.AuthServiceServer {
	return &grpcServer{
		authenticate: kitgrpc.NewServer(
			authenticateEndpoint(clients),
			decodeAuthenticateRequest,
			encodeAuthenticateResponse,
		),
		authorize: kitgrpc.NewServer(
			authorizeEndpoint(channels, parser),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
	}
}

func (s *grpcServer) Authenticate(ctx context.Context, req *authv1.AuthnReq) (*authv1.AuthnRes, error) {
	_, res, err := s.authenticate.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*authv1.AuthnRes), nil
}

func decodeAuthenticateRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*authv1.AuthnReq)
	return authenticateReq{
		clientID: req.GetClientId(),
		username: req.GetUsername(),
		password: req.GetPassword(),
	}, nil
}

func encodeAuthenticateResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(authenticateRes)
	return &authv1.AuthnRes{
		Authenticated: res.authenticated,
		Id:            res.id,
	}, nil
}

func (s *grpcServer) Authorize(ctx context.Context, req *authv1.AuthzReq) (*authv1.AuthzRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*authv1.AuthzRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*authv1.AuthzReq)
	return authorizeReq{
		externalID: req.GetExternalId(),
		topic:      req.GetTopic(),
		action:     uint8(req.GetAction()),
	}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(authorizeRes)
	return &authv1.AuthzRes{
		Authorized: res.authorized,
	}, nil
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrMissingID:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, smqauth.ErrKeyExpired):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, svcerr.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, messaging.ErrMalformedTopic):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
