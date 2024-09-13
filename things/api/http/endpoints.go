// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/policy"
	"github.com/absmach/magistrala/things"
	"github.com/go-kit/kit/endpoint"
)

func createClientEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, "", policy.UserType, policy.UsersKind, session.DomainUserID, policy.CreatePermission, policy.DomainType, session.DomainID); err != nil {
			return nil, err
		}

		client, err := svc.CreateThings(ctx, session, req.client)
		if err != nil {
			return nil, err
		}

		return createClientRes{
			Client:  client[0],
			created: true,
		}, nil
	}
}

func createClientsEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClientsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, "", policy.UserType, policy.UsersKind, session.DomainUserID, policy.CreatePermission, policy.DomainType, session.DomainID); err != nil {
			return nil, err
		}

		page, err := svc.CreateThings(ctx, session, req.Clients...)
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

func viewClientEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if _, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.ViewPermission, policy.ThingType, req.id); err != nil {
			return mgclients.Client{}, err
		}
		c, err := svc.ViewClient(ctx, req.id)
		if err != nil {
			return nil, err
		}

		return viewClientRes{Client: c}, nil
	}
}

func viewClientPermsEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewClientPermsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}

		p, err := svc.ViewClientPerms(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return viewClientPermsRes{Permissions: p}, nil
	}
}

func listClientsEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listClientsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		switch {
		case (req.userID != "" && req.userID != session.UserID):
			if _, err := authorize(ctx, authClient, "", policy.UserType, policy.UsersKind, session.DomainUserID, policy.AdminPermission, policy.DomainType, session.DomainID); err != nil {
				return nil, err
			}
		default:
			err := checkSuperAdmin(ctx, authClient, session.UserID)
			switch {
			case err != nil:
				if _, err := authorize(ctx, authClient, "", policy.UserType, policy.UsersKind, session.DomainUserID, policy.MembershipPermission, policy.DomainType, session.DomainID); err != nil {
					return nil, err
				}
			default:
				session.SuperAdmin = true
			}
		}
		pm := mgclients.Page{
			Status:     req.status,
			Offset:     req.offset,
			Limit:      req.limit,
			Name:       req.name,
			Tag:        req.tag,
			Permission: req.permission,
			Metadata:   req.metadata,
			ListPerms:  req.listPerms,
			Role:       mgclients.AllRole, // retrieve all things since things don't have roles
			Id:         req.id,
		}
		page, err := svc.ListClients(ctx, session, req.userID, pm)
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

func listMembersEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, req.Page.Permission, policy.GroupType, req.groupID); err != nil {
			return nil, err
		}

		req.Page.Role = mgclients.AllRole // retrieve all things since things don't have roles
		page, err := svc.ListClientsByGroup(ctx, session, req.groupID, req.Page)
		if err != nil {
			return nil, err
		}

		return buildClientsResponse(page), nil
	}
}

func updateClientEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.EditPermission, policy.ThingType, req.id)
		if err != nil {
			return nil, err
		}

		cli := mgclients.Client{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		client, err := svc.UpdateClient(ctx, session, cli)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientTagsEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.EditPermission, policy.ThingType, req.id)
		if err != nil {
			return nil, err
		}

		cli := mgclients.Client{
			ID:   req.id,
			Tags: req.Tags,
		}
		client, err := svc.UpdateClientTags(ctx, session, cli)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientSecretEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientCredentialsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.EditPermission, policy.ThingType, req.id)
		if err != nil {
			return nil, err
		}

		client, err := svc.UpdateClientSecret(ctx, session, req.id, req.Secret)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func enableClientEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.DeletePermission, policy.ThingType, req.id)
		if err != nil {
			return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
		}
		client, err := svc.EnableClient(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeClientStatusRes{Client: client}, nil
	}
}

func disableClientEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.DeletePermission, policy.ThingType, req.id)
		if err != nil {
			return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
		}
		client, err := svc.DisableClient(ctx, session, req.id)
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

func assignUsersEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignUsersRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.groupID); err != nil {
			return nil, err
		}
		if err := svc.Assign(ctx, session, req.groupID, req.Relation, policy.UsersKind, req.UserIDs...); err != nil {
			return nil, err
		}

		return assignUsersRes{}, nil
	}
}

func unassignUsersEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignUsersRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.groupID); err != nil {
			return nil, err
		}
		if err := svc.Unassign(ctx, session, req.groupID, req.Relation, policy.UsersKind, req.UserIDs...); err != nil {
			return nil, err
		}

		return unassignUsersRes{}, nil
	}
}

func assignUserGroupsEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignUserGroupsRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.groupID); err != nil {
			return nil, err
		}

		if err := svc.Assign(ctx, session, req.groupID, policy.ParentGroupRelation, policy.ChannelsKind, req.UserGroupIDs...); err != nil {
			return nil, err
		}

		return assignUserGroupsRes{}, nil
	}
}

func unassignUserGroupsEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignUserGroupsRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.groupID); err != nil {
			return nil, err
		}
		if err := svc.Unassign(ctx, session, req.groupID, policy.ParentGroupRelation, policy.ChannelsKind, req.UserGroupIDs...); err != nil {
			return nil, err
		}

		return unassignUserGroupsRes{}, nil
	}
}

func connectChannelThingEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.ChannelID); err != nil {
			return nil, err
		}

		if err := svc.Assign(ctx, session, req.ChannelID, policy.GroupRelation, policy.ThingsKind, req.ThingID); err != nil {
			return nil, err
		}

		return connectChannelThingRes{}, nil
	}
}

func disconnectChannelThingEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(disconnectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.ChannelID); err != nil {
			return nil, err
		}
		if err := svc.Unassign(ctx, session, req.ChannelID, policy.GroupRelation, policy.ThingsKind, req.ThingID); err != nil {
			return nil, err
		}

		return disconnectChannelThingRes{}, nil
	}
}

func connectEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.ChannelID); err != nil {
			return nil, err
		}
		if err := svc.Assign(ctx, session, req.ChannelID, policy.GroupRelation, policy.ThingsKind, req.ThingID); err != nil {
			return nil, err
		}

		return connectChannelThingRes{}, nil
	}
}

func disconnectEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(disconnectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.ChannelID); err != nil {
			return nil, err
		}
		if err := svc.Unassign(ctx, session, req.ChannelID, policy.GroupRelation, policy.ThingsKind, req.ThingID); err != nil {
			return nil, err
		}

		return disconnectChannelThingRes{}, nil
	}
}

func thingShareEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingShareRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.DeletePermission, policy.ThingType, req.thingID); err != nil {
			return nil, err
		}
		if err := svc.Share(ctx, session, req.thingID, req.Relation, req.UserIDs...); err != nil {
			return nil, err
		}

		return thingShareRes{}, nil
	}
}

func thingUnshareEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingUnshareRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.DeletePermission, policy.ThingType, req.thingID); err != nil {
			return nil, err
		}

		if err := svc.Unshare(ctx, session, req.thingID, req.Relation, req.UserIDs...); err != nil {
			return nil, err
		}

		return thingUnshareRes{}, nil
	}
}

func deleteClientEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return nil, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.DeletePermission, policy.ThingType, req.id); err != nil {
			return nil, err
		}
		if err := svc.DeleteClient(ctx, req.id); err != nil {
			return nil, err
		}

		return deleteClientRes{}, nil
	}
}

func identify(ctx context.Context, authClient auth.AuthClient, token string) (auth.Session, error) {
	resp, err := authClient.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return auth.Session{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if resp.GetId() == "" || resp.GetDomainId() == "" {
		return auth.Session{}, svcerr.ErrDomainAuthorization
	}
	return auth.Session{
		DomainUserID: resp.GetId(),
		UserID:       resp.GetUserId(),
		DomainID:     resp.GetDomainId(),
	}, nil
}

func checkSuperAdmin(ctx context.Context, authClient auth.AuthClient, adminID string) error {
	if _, err := authClient.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: policy.UserType,
		Subject:     adminID,
		Permission:  policy.AdminPermission,
		ObjectType:  policy.PlatformType,
		Object:      policy.MagistralaObject,
	}); err != nil {
		return err
	}
	return nil
}

func authorize(ctx context.Context, authClient auth.AuthClient, domainID, subjType, subjKind, subj, perm, objType, obj string) (auth.Session, error) {
	req := &magistrala.AuthorizeReq{
		Domain:      domainID,
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	res, err := authClient.Authorize(ctx, req)
	if err != nil {
		return auth.Session{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return auth.Session{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return auth.Session{
		UserID: res.GetId(),
	}, nil
}
