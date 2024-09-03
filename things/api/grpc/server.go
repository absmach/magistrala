// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/things"
	"github.com/absmach/magistrala/things/api/http"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ magistrala.AuthzServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	magistrala.UnimplementedAuthzServiceServer
	authorize         kitgrpc.Handler
	verifyConnections kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(svc things.Service) magistrala.AuthzServiceServer {
	return &grpcServer{
		authorize: kitgrpc.NewServer(
			(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		verifyConnections: kitgrpc.NewServer(
			(verifyConnectionsEndpoint(svc)),
			decodeVerifyConnectionsRequest,
			encodeVerifyConnectionsResponse,
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

func (s *grpcServer) VerifyConnections(ctx context.Context, req *magistrala.VerifyConnectionsReq) (*magistrala.VerifyConnectionsRes, error) {
	_, res, err := s.verifyConnections.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.VerifyConnectionsRes), nil
}

func decodeVerifyConnectionsRequest(_ context.Context, grpcreq interface{}) (interface{}, error) {
	req := grpcreq.(*magistrala.VerifyConnectionsReq)
	uniqueThings := http.GetUniqueValues(req.ThingIds)
	uniqueChannels := http.GetUniqueValues(req.ChannelIds)
	req.ThingIds = uniqueThings
	req.ChannelIds = uniqueChannels
	return req, nil
}

func encodeVerifyConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(verifyConnectionsRes)
	connections := []*magistrala.ConnStatus{}
	for _, conn := range res.Connections {
		connections = append(connections, &magistrala.ConnStatus{
			ThingId:   conn.ThingId,
			ChannelId: conn.ChannelId,
			Status:    conn.Status,
		})
	}
	return &magistrala.VerifyConnectionsRes{
		Status:            res.Status,
		ConnectionsStatus: connections,
	}, nil
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
		err == apiutil.ErrMalformedPolicyAct,
		err == apiutil.ErrMissingThingIDs,
		err == apiutil.ErrMissingChannelIDs:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, auth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, svcerr.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
