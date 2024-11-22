// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	mgauth "github.com/absmach/magistrala/auth"
	clients "github.com/absmach/magistrala/clients/private"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/magistrala/internal/grpc/common/v1"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ grpcClientsV1.ClientsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	grpcClientsV1.UnimplementedClientsServiceServer
	authenticate               kitgrpc.Handler
	retrieveEntity             kitgrpc.Handler
	retrieveEntities           kitgrpc.Handler
	addConnections             kitgrpc.Handler
	removeConnections          kitgrpc.Handler
	removeChannelConnections   kitgrpc.Handler
	unsetParentGroupFromClient kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(svc clients.Service) grpcClientsV1.ClientsServiceServer {
	return &grpcServer{
		authenticate: kitgrpc.NewServer(
			authenticateEndpoint(svc),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		retrieveEntity: kitgrpc.NewServer(
			retrieveEntityEndpoint(svc),
			decodeRetrieveEntityRequest,
			encodeRetrieveEntityResponse,
		),
		retrieveEntities: kitgrpc.NewServer(
			retrieveEntitiesEndpoint(svc),
			decodeRetrieveEntitiesRequest,
			encodeRetrieveEntitiesResponse,
		),
		addConnections: kitgrpc.NewServer(
			addConnectionsEndpoint(svc),
			decodeAddConnectionsRequest,
			encodeAddConnectionsResponse,
		),
		removeConnections: kitgrpc.NewServer(
			removeConnectionsEndpoint(svc),
			decodeRemoveConnectionsRequest,
			encodeRemoveConnectionsResponse,
		),
		removeChannelConnections: kitgrpc.NewServer(
			removeChannelConnectionsEndpoint(svc),
			decodeRemoveChannelConnectionsRequest,
			encodeRemoveChannelConnectionsResponse,
		),
		unsetParentGroupFromClient: kitgrpc.NewServer(
			UnsetParentGroupFromClientEndpoint(svc),
			decodeUnsetParentGroupFromClientRequest,
			encodeUnsetParentGroupFromClientResponse,
		),
	}
}

func (s *grpcServer) Authenticate(ctx context.Context, req *grpcClientsV1.AuthnReq) (*grpcClientsV1.AuthnRes, error) {
	_, res, err := s.authenticate.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcClientsV1.AuthnRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcClientsV1.AuthnReq)
	return authenticateReq{
		ClientID:     req.GetClientId(),
		ClientSecret: req.GetClientSecret(),
	}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authenticateRes)
	return &grpcClientsV1.AuthnRes{Authenticated: res.authenticated, Id: res.id}, nil
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

func (s *grpcServer) RetrieveEntities(ctx context.Context, req *grpcCommonV1.RetrieveEntitiesReq) (*grpcCommonV1.RetrieveEntitiesRes, error) {
	_, res, err := s.retrieveEntities.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcCommonV1.RetrieveEntitiesRes), nil
}

func decodeRetrieveEntitiesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcCommonV1.RetrieveEntitiesReq)
	return retrieveEntitiesReq{
		Ids: req.GetIds(),
	}, nil
}

func encodeRetrieveEntitiesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(retrieveEntitiesRes)

	entities := []*grpcCommonV1.EntityBasic{}
	for _, c := range res.clients {
		entities = append(entities, &grpcCommonV1.EntityBasic{
			Id:            c.id,
			DomainId:      c.domain,
			ParentGroupId: c.parentGroup,
			Status:        uint32(c.status),
		})
	}
	return &grpcCommonV1.RetrieveEntitiesRes{Total: res.total, Limit: res.limit, Offset: res.offset, Entities: entities}, nil
}

func (s *grpcServer) AddConnections(ctx context.Context, req *grpcCommonV1.AddConnectionsReq) (*grpcCommonV1.AddConnectionsRes, error) {
	_, res, err := s.addConnections.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcCommonV1.AddConnectionsRes), nil
}

func decodeAddConnectionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcCommonV1.AddConnectionsReq)

	conns := []connection{}
	for _, c := range req.Connections {
		connType := connections.ConnType(c.GetType())
		if err := connections.CheckConnType(connType); err != nil {
			return nil, err
		}
		conns = append(conns, connection{
			clientID:  c.GetClientId(),
			channelID: c.GetChannelId(),
			domainID:  c.GetDomainId(),
			connType:  connType,
		})
	}
	return connectionsReq{
		connections: conns,
	}, nil
}

func encodeAddConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(connectionsRes)

	return &grpcCommonV1.AddConnectionsRes{Ok: res.ok}, nil
}

func (s *grpcServer) RemoveConnections(ctx context.Context, req *grpcCommonV1.RemoveConnectionsReq) (*grpcCommonV1.RemoveConnectionsRes, error) {
	_, res, err := s.removeConnections.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcCommonV1.RemoveConnectionsRes), nil
}

func decodeRemoveConnectionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcCommonV1.RemoveConnectionsReq)

	conns := []connection{}
	for _, c := range req.Connections {
		connType := connections.ConnType(c.GetType())
		if err := connections.CheckConnType(connType); err != nil {
			return nil, err
		}
		conns = append(conns, connection{
			clientID:  c.GetClientId(),
			channelID: c.GetChannelId(),
			domainID:  c.GetDomainId(),
			connType:  connType,
		})
	}
	return connectionsReq{
		connections: conns,
	}, nil
}

func encodeRemoveConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(connectionsRes)

	return &grpcCommonV1.RemoveConnectionsRes{Ok: res.ok}, nil
}

func (s *grpcServer) RemoveChannelConnections(ctx context.Context, req *grpcClientsV1.RemoveChannelConnectionsReq) (*grpcClientsV1.RemoveChannelConnectionsRes, error) {
	_, res, err := s.removeChannelConnections.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcClientsV1.RemoveChannelConnectionsRes), nil
}

func decodeRemoveChannelConnectionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcClientsV1.RemoveChannelConnectionsReq)

	return removeChannelConnectionsReq{
		channelID: req.GetChannelId(),
	}, nil
}

func encodeRemoveChannelConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	_ = grpcRes.(removeChannelConnectionsRes)
	return &grpcClientsV1.RemoveChannelConnectionsRes{}, nil
}

func (s *grpcServer) UnsetParentGroupFromClient(ctx context.Context, req *grpcClientsV1.UnsetParentGroupFromClientReq) (*grpcClientsV1.UnsetParentGroupFromClientRes, error) {
	_, res, err := s.unsetParentGroupFromClient.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcClientsV1.UnsetParentGroupFromClientRes), nil
}

func decodeUnsetParentGroupFromClientRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcClientsV1.UnsetParentGroupFromClientReq)

	return UnsetParentGroupFromClientReq{
		parentGroupID: req.GetParentGroupId(),
	}, nil
}

func encodeUnsetParentGroupFromClientResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	_ = grpcRes.(UnsetParentGroupFromClientRes)
	return &grpcClientsV1.UnsetParentGroupFromClientRes{}, nil
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
