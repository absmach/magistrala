// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala/clients"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/magistrala/internal/grpc/common/v1"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const svcName = "clients.v1.ClientsService"

var _ grpcClientsV1.ClientsServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	timeout                    time.Duration
	authenticate               endpoint.Endpoint
	retrieveEntity             endpoint.Endpoint
	retrieveEntities           endpoint.Endpoint
	addConnections             endpoint.Endpoint
	removeConnections          endpoint.Endpoint
	removeChannelConnections   endpoint.Endpoint
	unsetParentGroupFromClient endpoint.Endpoint
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, timeout time.Duration) grpcClientsV1.ClientsServiceClient {
	return &grpcClient{
		authenticate: kitgrpc.NewClient(
			conn,
			svcName,
			"Authenticate",
			encodeAuthenticateRequest,
			decodeAuthenticateResponse,
			grpcClientsV1.AuthnRes{},
		).Endpoint(),

		retrieveEntity: kitgrpc.NewClient(
			conn,
			svcName,
			"RetrieveEntity",
			encodeRetrieveEntityRequest,
			decodeRetrieveEntityResponse,
			grpcCommonV1.RetrieveEntityRes{},
		).Endpoint(),

		retrieveEntities: kitgrpc.NewClient(
			conn,
			svcName,
			"RetrieveEntities",
			encodeRetrieveEntitiesRequest,
			decodeRetrieveEntitiesResponse,
			grpcCommonV1.RetrieveEntitiesRes{},
		).Endpoint(),

		addConnections: kitgrpc.NewClient(
			conn,
			svcName,
			"AddConnections",
			encodeAddConnectionsRequest,
			decodeAddConnectionsResponse,
			grpcCommonV1.AddConnectionsRes{},
		).Endpoint(),

		removeConnections: kitgrpc.NewClient(
			conn,
			svcName,
			"RemoveConnections",
			encodeRemoveConnectionsRequest,
			decodeRemoveConnectionsResponse,
			grpcCommonV1.RemoveConnectionsRes{},
		).Endpoint(),

		removeChannelConnections: kitgrpc.NewClient(
			conn,
			svcName,
			"RemoveChannelConnections",
			encodeRemoveChannelConnectionsRequest,
			decodeRemoveChannelConnectionsResponse,
			grpcClientsV1.RemoveChannelConnectionsRes{},
		).Endpoint(),

		unsetParentGroupFromClient: kitgrpc.NewClient(
			conn,
			svcName,
			"UnsetParentGroupFromClient",
			encodeUnsetParentGroupFromClientRequest,
			decodeUnsetParentGroupFromClientResponse,
			grpcClientsV1.UnsetParentGroupFromClientRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client grpcClient) Authenticate(ctx context.Context, req *grpcClientsV1.AuthnReq, _ ...grpc.CallOption) (r *grpcClientsV1.AuthnRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.authenticate(ctx, authenticateReq{
		ClientID:     req.GetClientId(),
		ClientSecret: req.GetClientSecret(),
	})
	if err != nil {
		return &grpcClientsV1.AuthnRes{}, decodeError(err)
	}

	ar := res.(authenticateRes)
	return &grpcClientsV1.AuthnRes{Authenticated: ar.authenticated, Id: ar.id}, nil
}

func encodeAuthenticateRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authenticateReq)
	return &grpcClientsV1.AuthnReq{
		ClientId:     req.ClientID,
		ClientSecret: req.ClientSecret,
	}, nil
}

func decodeAuthenticateResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcClientsV1.AuthnRes)
	return authenticateRes{authenticated: res.GetAuthenticated(), id: res.GetId()}, nil
}

func (client grpcClient) RetrieveEntity(ctx context.Context, req *grpcCommonV1.RetrieveEntityReq, _ ...grpc.CallOption) (r *grpcCommonV1.RetrieveEntityRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.retrieveEntity(ctx, req.GetId())
	if err != nil {
		return &grpcCommonV1.RetrieveEntityRes{}, decodeError(err)
	}

	ebr := res.(retrieveEntityRes)

	return &grpcCommonV1.RetrieveEntityRes{Entity: &grpcCommonV1.EntityBasic{Id: ebr.id, DomainId: ebr.domain, Status: uint32(ebr.status)}}, nil
}

func encodeRetrieveEntityRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(string)
	return &grpcCommonV1.RetrieveEntityReq{
		Id: req,
	}, nil
}

func decodeRetrieveEntityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcCommonV1.RetrieveEntityRes)

	return retrieveEntityRes{
		id:          res.Entity.GetId(),
		domain:      res.Entity.GetDomainId(),
		parentGroup: res.Entity.GetParentGroupId(),
		status:      uint8(res.Entity.GetStatus()),
	}, nil
}

func (client grpcClient) RetrieveEntities(ctx context.Context, req *grpcCommonV1.RetrieveEntitiesReq, _ ...grpc.CallOption) (r *grpcCommonV1.RetrieveEntitiesRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.retrieveEntities(ctx, req.GetIds())
	if err != nil {
		return &grpcCommonV1.RetrieveEntitiesRes{}, decodeError(err)
	}

	ep := res.(retrieveEntitiesRes)

	entities := []*grpcCommonV1.EntityBasic{}
	for _, c := range ep.clients {
		entities = append(entities, &grpcCommonV1.EntityBasic{
			Id:       c.id,
			DomainId: c.domain,
			Status:   uint32(c.status),
		})
	}
	return &grpcCommonV1.RetrieveEntitiesRes{Total: ep.total, Limit: ep.limit, Offset: ep.offset, Entities: entities}, nil
}

func encodeRetrieveEntitiesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.([]string)
	return &grpcCommonV1.RetrieveEntitiesReq{
		Ids: req,
	}, nil
}

func decodeRetrieveEntitiesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcCommonV1.RetrieveEntitiesRes)

	clis := []enitity{}

	for _, e := range res.Entities {
		clis = append(clis, enitity{
			id:          e.GetId(),
			domain:      e.GetDomainId(),
			parentGroup: e.GetParentGroupId(),
			status:      uint8(e.GetStatus()),
		})
	}
	return retrieveEntitiesRes{total: res.GetTotal(), limit: res.GetLimit(), offset: res.GetOffset(), clients: clis}, nil
}

func (client grpcClient) AddConnections(ctx context.Context, req *grpcCommonV1.AddConnectionsReq, _ ...grpc.CallOption) (r *grpcCommonV1.AddConnectionsRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	conns := []clients.Connection{}
	for _, c := range req.Connections {
		conns = append(conns, clients.Connection{
			ClientID:  c.GetClientId(),
			ChannelID: c.GetChannelId(),
			DomainID:  c.GetDomainId(),
			Type:      connections.ConnType(c.GetType()),
		})
	}

	res, err := client.addConnections(ctx, conns)
	if err != nil {
		return &grpcCommonV1.AddConnectionsRes{}, decodeError(err)
	}

	cr := res.(connectionsRes)

	return &grpcCommonV1.AddConnectionsRes{Ok: cr.ok}, nil
}

func encodeAddConnectionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.([]clients.Connection)

	conns := []*grpcCommonV1.Connection{}

	for _, r := range req {
		conns = append(conns, &grpcCommonV1.Connection{
			ClientId:  r.ClientID,
			ChannelId: r.ChannelID,
			DomainId:  r.DomainID,
			Type:      uint32(r.Type),
		})
	}
	return &grpcCommonV1.AddConnectionsReq{
		Connections: conns,
	}, nil
}

func decodeAddConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcCommonV1.AddConnectionsRes)

	return connectionsRes{ok: res.GetOk()}, nil
}

func (client grpcClient) RemoveConnections(ctx context.Context, req *grpcCommonV1.RemoveConnectionsReq, _ ...grpc.CallOption) (r *grpcCommonV1.RemoveConnectionsRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	conns := []clients.Connection{}
	for _, c := range req.Connections {
		conns = append(conns, clients.Connection{
			ClientID:  c.GetClientId(),
			ChannelID: c.GetChannelId(),
			DomainID:  c.GetDomainId(),
			Type:      connections.ConnType(c.GetType()),
		})
	}

	res, err := client.removeConnections(ctx, conns)
	if err != nil {
		return &grpcCommonV1.RemoveConnectionsRes{}, decodeError(err)
	}

	cr := res.(connectionsRes)

	return &grpcCommonV1.RemoveConnectionsRes{Ok: cr.ok}, nil
}

func encodeRemoveConnectionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.([]clients.Connection)

	conns := []*grpcCommonV1.Connection{}

	for _, r := range req {
		conns = append(conns, &grpcCommonV1.Connection{
			ClientId:  r.ClientID,
			ChannelId: r.ChannelID,
			DomainId:  r.DomainID,
			Type:      uint32(r.Type),
		})
	}
	return &grpcCommonV1.RemoveConnectionsReq{
		Connections: conns,
	}, nil
}

func decodeRemoveConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcCommonV1.RemoveConnectionsRes)

	return connectionsRes{ok: res.GetOk()}, nil
}

func (client grpcClient) RemoveChannelConnections(ctx context.Context, req *grpcClientsV1.RemoveChannelConnectionsReq, _ ...grpc.CallOption) (r *grpcClientsV1.RemoveChannelConnectionsRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	if _, err := client.removeChannelConnections(ctx, req); err != nil {
		return &grpcClientsV1.RemoveChannelConnectionsRes{}, decodeError(err)
	}

	return &grpcClientsV1.RemoveChannelConnectionsRes{}, nil
}

func encodeRemoveChannelConnectionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	return grpcReq.(*grpcClientsV1.RemoveChannelConnectionsReq), nil
}

func decodeRemoveChannelConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes.(*grpcClientsV1.RemoveChannelConnectionsRes), nil
}

func (client grpcClient) UnsetParentGroupFromClient(ctx context.Context, req *grpcClientsV1.UnsetParentGroupFromClientReq, _ ...grpc.CallOption) (r *grpcClientsV1.UnsetParentGroupFromClientRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	if _, err := client.unsetParentGroupFromClient(ctx, req); err != nil {
		return &grpcClientsV1.UnsetParentGroupFromClientRes{}, decodeError(err)
	}

	return &grpcClientsV1.UnsetParentGroupFromClientRes{}, nil
}

func encodeUnsetParentGroupFromClientRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	return grpcReq.(*grpcClientsV1.UnsetParentGroupFromClientReq), nil
}

func decodeUnsetParentGroupFromClientResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes.(*grpcClientsV1.UnsetParentGroupFromClientRes), nil
}

func decodeError(err error) error {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unauthenticated:
			return errors.Wrap(svcerr.ErrAuthentication, errors.New(st.Message()))
		case codes.PermissionDenied:
			return errors.Wrap(svcerr.ErrAuthorization, errors.New(st.Message()))
		case codes.InvalidArgument:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.FailedPrecondition:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.NotFound:
			return errors.Wrap(svcerr.ErrNotFound, errors.New(st.Message()))
		case codes.AlreadyExists:
			return errors.Wrap(svcerr.ErrConflict, errors.New(st.Message()))
		case codes.OK:
			if msg := st.Message(); msg != "" {
				return errors.Wrap(errors.ErrUnidentified, errors.New(msg))
			}
			return nil
		default:
			return errors.Wrap(fmt.Errorf("unexpected gRPC status: %s (status code:%v)", st.Code().String(), st.Code()), errors.New(st.Message()))
		}
	}
	return err
}
