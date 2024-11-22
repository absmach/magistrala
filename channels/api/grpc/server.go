// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	mgauth "github.com/absmach/magistrala/auth"
	channels "github.com/absmach/magistrala/channels/private"
	grpcChannelsV1 "github.com/absmach/magistrala/internal/grpc/channels/v1"
	grpcCommonV1 "github.com/absmach/magistrala/internal/grpc/common/v1"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ grpcChannelsV1.ChannelsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	grpcChannelsV1.UnimplementedChannelsServiceServer

	authorize                    kitgrpc.Handler
	removeClientConnections       kitgrpc.Handler
	unsetParentGroupFromChannels kitgrpc.Handler
	retrieveEntity               kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(svc channels.Service) grpcChannelsV1.ChannelsServiceServer {
	return &grpcServer{
		authorize: kitgrpc.NewServer(
			authorizeEndpoint(svc),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		removeClientConnections: kitgrpc.NewServer(
			removeClientConnectionsEndpoint(svc),
			decodeRemoveClientConnectionsRequest,
			encodeRemoveClientConnectionsResponse,
		),
		unsetParentGroupFromChannels: kitgrpc.NewServer(
			unsetParentGroupFromChannelsEndpoint(svc),
			decodeUnsetParentGroupFromChannelsRequest,
			encodeUnsetParentGroupFromChannelsResponse,
		),
		retrieveEntity: kitgrpc.NewServer(
			retrieveEntityEndpoint(svc),
			decodeRetrieveEntityRequest,
			encodeRetrieveEntityResponse,
		),
	}
}

func (s *grpcServer) Authorize(ctx context.Context, req *grpcChannelsV1.AuthzReq) (*grpcChannelsV1.AuthzRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcChannelsV1.AuthzRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcChannelsV1.AuthzReq)

	connType := connections.ConnType(req.GetType())
	if err := connections.CheckConnType(connType); err != nil {
		return nil, err
	}
	return authorizeReq{
		domainID:   req.GetDomainId(),
		clientID:   req.GetClientId(),
		clientType: req.GetClientType(),
		channelID:  req.GetChannelId(),
		connType:   connType,
	}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &grpcChannelsV1.AuthzRes{Authorized: res.authorized}, nil
}

func (s *grpcServer) RemoveClientConnections(ctx context.Context, req *grpcChannelsV1.RemoveClientConnectionsReq) (*grpcChannelsV1.RemoveClientConnectionsRes, error) {
	_, res, err := s.removeClientConnections.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcChannelsV1.RemoveClientConnectionsRes), nil
}

func decodeRemoveClientConnectionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcChannelsV1.RemoveClientConnectionsReq)

	return removeClientConnectionsReq{
		clientID: req.GetClientId(),
	}, nil
}

func encodeRemoveClientConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	_ = grpcRes.(removeClientConnectionsRes)
	return &grpcChannelsV1.RemoveClientConnectionsRes{}, nil
}

func (s *grpcServer) UnsetParentGroupFromChannels(ctx context.Context, req *grpcChannelsV1.UnsetParentGroupFromChannelsReq) (*grpcChannelsV1.UnsetParentGroupFromChannelsRes, error) {
	_, res, err := s.unsetParentGroupFromChannels.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcChannelsV1.UnsetParentGroupFromChannelsRes), nil
}

func decodeUnsetParentGroupFromChannelsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcChannelsV1.UnsetParentGroupFromChannelsReq)

	return unsetParentGroupFromChannelsReq{
		parentGroupID: req.GetParentGroupId(),
	}, nil
}

func encodeUnsetParentGroupFromChannelsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	_ = grpcRes.(unsetParentGroupFromChannelsRes)
	return &grpcChannelsV1.UnsetParentGroupFromChannelsRes{}, nil
}

func (s *grpcServer) RetrieveEntity(ctx context.Context, req *grpcCommonV1.RetrieveEntityReq) (*grpcCommonV1.RetrieveEntityRes, error) {
	_, res, err := s.retrieveEntity.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcCommonV1.RetrieveEntityRes), nil
}

func decodeRetrieveEntityRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcCommonV1.RetrieveEntityReq)
	return retrieveEntityReq{
		Id: req.GetId(),
	}, nil
}

func encodeRetrieveEntityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(retrieveEntityRes)

	return &grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{
			Id:            res.id,
			DomainId:      res.domain,
			ParentGroupId: res.parentGroup,
			Status:        uint32(res.status),
		},
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
