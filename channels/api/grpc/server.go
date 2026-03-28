// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauth "github.com/absmach/supermq/auth"
	channels "github.com/absmach/supermq/channels/private"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ grpcChannelsV1.ChannelsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	grpcChannelsV1.UnimplementedChannelsServiceServer
	authorize                    kitgrpc.Handler
	removeClientConnections      kitgrpc.Handler
	unsetParentGroupFromChannels kitgrpc.Handler
	retrieveEntity               kitgrpc.Handler
	retrieveIDByRoute            kitgrpc.Handler
	deleteDomainChannels         kitgrpc.Handler
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
		retrieveIDByRoute: kitgrpc.NewServer(
			retrieveIDByRouteEndpoint(svc),
			decodeRetrieveIDByRouteRequest,
			encodeRetrieveIDByRouteResponse,
		),
		deleteDomainChannels: kitgrpc.NewServer(
			deleteDomainChannelsEndpoint(svc),
			decodeDeleteDomainChannelsRequest,
			encodeDeleteDomainChannelsResponse,
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

func decodeAuthorizeRequest(_ context.Context, grpcReq any) (any, error) {
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

func encodeAuthorizeResponse(_ context.Context, grpcRes any) (any, error) {
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

func decodeRemoveClientConnectionsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcChannelsV1.RemoveClientConnectionsReq)

	return removeClientConnectionsReq{
		clientID: req.GetClientId(),
	}, nil
}

func encodeRemoveClientConnectionsResponse(_ context.Context, grpcRes any) (any, error) {
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

func decodeUnsetParentGroupFromChannelsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcChannelsV1.UnsetParentGroupFromChannelsReq)

	return unsetParentGroupFromChannelsReq{
		parentGroupID: req.GetParentGroupId(),
	}, nil
}

func encodeUnsetParentGroupFromChannelsResponse(_ context.Context, grpcRes any) (any, error) {
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

func decodeRetrieveEntityRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcCommonV1.RetrieveEntityReq)
	return retrieveEntityReq{
		Id: req.GetId(),
	}, nil
}

func encodeRetrieveEntityResponse(_ context.Context, grpcRes any) (any, error) {
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

func decodeRetrieveIDByRouteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcCommonV1.RetrieveIDByRouteReq)
	return retrieveIDByRouteReq{
		route:    req.GetRoute(),
		domainID: req.GetDomainId(),
	}, nil
}

func encodeRetrieveIDByRouteResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(retrieveIDByRouteRes)

	return &grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{
			Id: res.id,
		},
	}, nil
}

func (s *grpcServer) RetrieveIDByRoute(ctx context.Context, req *grpcCommonV1.RetrieveIDByRouteReq) (*grpcCommonV1.RetrieveEntityRes, error) {
	_, res, err := s.retrieveIDByRoute.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcCommonV1.RetrieveEntityRes), nil
}

func decodeDeleteDomainChannelsRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcCommonV1.DeleteDomainEntitiesReq)

	return deleteDomainChannelsReq{
		domainID: req.GetDomainId(),
	}, nil
}

func encodeDeleteDomainChannelsResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(deleteDomainChannelsRes)

	return &grpcCommonV1.DeleteDomainEntitiesRes{
		Deleted: res.deleted,
	}, nil
}

func (s *grpcServer) DeleteDomainChannels(ctx context.Context, req *grpcCommonV1.DeleteDomainEntitiesReq) (*grpcCommonV1.DeleteDomainEntitiesRes, error) {
	_, res, err := s.deleteDomainChannels.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}

	return res.(*grpcCommonV1.DeleteDomainEntitiesRes), nil
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
		errors.Contains(err, smqauth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, svcerr.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
