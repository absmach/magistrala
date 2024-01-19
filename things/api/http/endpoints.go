// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/things"
	"github.com/go-kit/kit/endpoint"
)

func createClientEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.CreateThings(ctx, req.token, req.client)
		if err != nil {
			return nil, err
		}

		return createClientRes{
			Client:  client[0],
			created: true,
		}, nil
	}
}

func createClientsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClientsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.CreateThings(ctx, req.token, req.Clients...)
		if err != nil {
			return nil, err
		}

		res := clientsPageRes{
			pageRes: pageRes{
				Total: uint64(len(page)),
			},
			Clients: []viewClientRes{},
		}
		for _, c := range page {
			res.Clients = append(res.Clients, viewClientRes{Client: c})
		}

		return res, nil
	}
}

func viewClientEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		c, err := svc.ViewClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return viewClientRes{Client: c}, nil
	}
}

func viewClientPermsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewClientPermsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		p, err := svc.ViewClientPerms(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return viewClientPermsRes{Permissions: p}, nil
	}
}

func listClientsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listClientsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		pm := mgclients.Page{
			Status:     req.status,
			Offset:     req.offset,
			Limit:      req.limit,
			Owner:      req.owner,
			Name:       req.name,
			Tag:        req.tag,
			Permission: req.permission,
			Metadata:   req.metadata,
			ListPerms:  req.listPerms,
			Role:       mgclients.AllRole, // retrieve all things since things don't have roles
		}
		page, err := svc.ListClients(ctx, req.token, req.userID, pm)
		if err != nil {
			return nil, err
		}

		res := clientsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Clients: []viewClientRes{},
		}
		for _, c := range page.Clients {
			res.Clients = append(res.Clients, viewClientRes{Client: c})
		}

		return res, nil
	}
}

func listMembersEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.ListClientsByGroup(ctx, req.token, req.groupID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildClientsResponse(page), nil
	}
}

func updateClientEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cli := mgclients.Client{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		client, err := svc.UpdateClient(ctx, req.token, cli)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientTagsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cli := mgclients.Client{
			ID:   req.id,
			Tags: req.Tags,
		}
		client, err := svc.UpdateClientTags(ctx, req.token, cli)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientSecretEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientCredentialsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.UpdateClientSecret(ctx, req.token, req.id, req.Secret)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func enableClientEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.EnableClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return changeClientStatusRes{Client: client}, nil
	}
}

func disableClientEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.DisableClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return changeClientStatusRes{Client: client}, nil
	}
}

func buildClientsResponse(cp mgclients.MembersPage) clientsPageRes {
	res := clientsPageRes{
		pageRes: pageRes{
			Total:  cp.Total,
			Offset: cp.Offset,
			Limit:  cp.Limit,
		},
		Clients: []viewClientRes{},
	}
	for _, c := range cp.Members {
		res.Clients = append(res.Clients, viewClientRes{Client: c})
	}

	return res
}

func assignUsersEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignUsersRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Assign(ctx, req.token, req.groupID, req.Relation, auth.UsersKind, req.UserIDs...); err != nil {
			return nil, err
		}

		return assignUsersRes{}, nil
	}
}

func unassignUsersEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignUsersRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Unassign(ctx, req.token, req.groupID, req.Relation, auth.UsersKind, req.UserIDs...); err != nil {
			return nil, err
		}

		return unassignUsersRes{}, nil
	}
}

func assignUserGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignUserGroupsRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Assign(ctx, req.token, req.groupID, auth.ParentGroupRelation, auth.ChannelsKind, req.UserGroupIDs...); err != nil {
			return nil, err
		}

		return assignUserGroupsRes{}, nil
	}
}

func unassignUserGroupsEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignUserGroupsRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Unassign(ctx, req.token, req.groupID, auth.ParentGroupRelation, auth.ChannelsKind, req.UserGroupIDs...); err != nil {
			return nil, err
		}

		return unassignUserGroupsRes{}, nil
	}
}

func connectChannelThingEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Assign(ctx, req.token, req.ChannelID, auth.GroupRelation, auth.ThingsKind, req.ThingID); err != nil {
			return nil, err
		}

		return connectChannelThingRes{}, nil
	}
}

func disconnectChannelThingEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(disconnectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Unassign(ctx, req.token, req.ChannelID, auth.GroupRelation, auth.ThingsKind, req.ThingID); err != nil {
			return nil, err
		}

		return disconnectChannelThingRes{}, nil
	}
}

func connectEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Assign(ctx, req.token, req.ChannelID, auth.GroupRelation, auth.ThingsKind, req.ThingID); err != nil {
			return nil, err
		}

		return connectChannelThingRes{}, nil
	}
}

func disconnectEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(disconnectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Unassign(ctx, req.token, req.ChannelID, auth.GroupRelation, auth.ThingsKind, req.ThingID); err != nil {
			return nil, err
		}

		return disconnectChannelThingRes{}, nil
	}
}

func thingShareEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingShareRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Share(ctx, req.token, req.thingID, req.Relation, req.UserIDs...); err != nil {
			return nil, err
		}

		return thingShareRes{}, nil
	}
}

func thingUnshareEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingUnshareRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Unshare(ctx, req.token, req.thingID, req.Relation, req.UserIDs...); err != nil {
			return nil, err
		}

		return thingUnshareRes{}, nil
	}
}

func deleteClientEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.DeleteClient(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return deleteClientRes{}, nil
	}
}
