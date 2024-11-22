// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala/clients"
	pClients "github.com/absmach/magistrala/clients/private"
	"github.com/go-kit/kit/endpoint"
)

func authenticateEndpoint(svc pClients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authenticateReq)
		id, err := svc.Authenticate(ctx, req.ClientSecret)
		if err != nil {
			return authenticateRes{}, err
		}
		return authenticateRes{
			authenticated: true,
			id:            id,
		}, err
	}
}

func retrieveEntityEndpoint(svc pClients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveEntityReq)
		client, err := svc.RetrieveById(ctx, req.Id)
		if err != nil {
			return retrieveEntityRes{}, err
		}

		return retrieveEntityRes{id: client.ID, domain: client.Domain, parentGroup: client.ParentGroup, status: uint8(client.Status)}, nil
	}
}

func retrieveEntitiesEndpoint(svc pClients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveEntitiesReq)
		tp, err := svc.RetrieveByIds(ctx, req.Ids)
		if err != nil {
			return retrieveEntitiesRes{}, err
		}
		clientsBasic := []enitity{}
		for _, client := range tp.Clients {
			clientsBasic = append(clientsBasic, enitity{id: client.ID, domain: client.Domain, parentGroup: client.ParentGroup, status: uint8(client.Status)})
		}
		return retrieveEntitiesRes{
			total:   tp.Total,
			limit:   tp.Limit,
			offset:  tp.Offset,
			clients: clientsBasic,
		}, nil
	}
}

func addConnectionsEndpoint(svc pClients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectionsReq)

		var conns []clients.Connection

		for _, c := range req.connections {
			conns = append(conns, clients.Connection{
				ClientID:  c.clientID,
				ChannelID: c.channelID,
				DomainID:  c.domainID,
				Type:      c.connType,
			})
		}

		if err := svc.AddConnections(ctx, conns); err != nil {
			return connectionsRes{ok: false}, err
		}

		return connectionsRes{ok: true}, nil
	}
}

func removeConnectionsEndpoint(svc pClients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectionsReq)

		var conns []clients.Connection

		for _, c := range req.connections {
			conns = append(conns, clients.Connection{
				ClientID:  c.clientID,
				ChannelID: c.channelID,
				DomainID:  c.domainID,
				Type:      c.connType,
			})
		}
		if err := svc.RemoveConnections(ctx, conns); err != nil {
			return connectionsRes{ok: false}, err
		}

		return connectionsRes{ok: true}, nil
	}
}

func removeChannelConnectionsEndpoint(svc pClients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeChannelConnectionsReq)

		if err := svc.RemoveChannelConnections(ctx, req.channelID); err != nil {
			return removeChannelConnectionsRes{}, err
		}

		return removeChannelConnectionsRes{}, nil
	}
}

func UnsetParentGroupFromClientEndpoint(svc pClients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UnsetParentGroupFromClientReq)

		if err := svc.UnsetParentGroupFromClient(ctx, req.parentGroupID); err != nil {
			return UnsetParentGroupFromClientRes{}, err
		}

		return UnsetParentGroupFromClientRes{}, nil
	}
}
