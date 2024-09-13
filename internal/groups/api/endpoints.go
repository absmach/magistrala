// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/policy"
	"github.com/go-kit/kit/endpoint"
)

const groupTypeChannels = "channels"

func CreateGroupEndpoint(svc groups.Service, authClient auth.AuthClient, kind string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupReq)
		if err := req.validate(); err != nil {
			return createGroupRes{created: false}, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return createGroupRes{created: false}, err
		}
		if _, err := authorize(ctx, authClient, "", policy.UserType, policy.UsersKind, session.DomainUserID, policy.CreatePermission, policy.DomainType, session.DomainID); err != nil {
			return createGroupRes{created: false}, err
		}
		if req.Group.Parent != "" {
			if _, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.EditPermission, policy.GroupType, req.Group.Parent); err != nil {
				return createGroupRes{created: false}, errors.Wrap(svcerr.ErrParentGroupAuthorization, err)
			}
		}

		group, err := svc.CreateGroup(ctx, session, kind, req.Group)
		if err != nil {
			return createGroupRes{created: false}, err
		}

		return createGroupRes{created: true, Group: group}, nil
	}
}

func ViewGroupEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return viewGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		if _, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.ViewPermission, policy.GroupType, req.id); err != nil {
			return viewGroupRes{}, err
		}

		group, err := svc.ViewGroup(ctx, req.id)
		if err != nil {
			return viewGroupRes{}, err
		}

		return viewGroupRes{Group: group}, nil
	}
}

func ViewGroupPermsEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupPermsReq)
		if err := req.validate(); err != nil {
			return viewGroupPermsRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return viewGroupPermsRes{}, err
		}

		p, err := svc.ViewGroupPerms(ctx, session, req.id)
		if err != nil {
			return viewGroupPermsRes{}, err
		}

		return viewGroupPermsRes{Permissions: p}, nil
	}
}

func UpdateGroupEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupReq)
		if err := req.validate(); err != nil {
			return updateGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.EditPermission, policy.GroupType, req.id)
		if err != nil {
			return updateGroupRes{}, err
		}
		group := groups.Group{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		group, err = svc.UpdateGroup(ctx, session, group)
		if err != nil {
			return updateGroupRes{}, err
		}

		return updateGroupRes{Group: group}, nil
	}
}

func EnableGroupEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeGroupStatusReq)
		if err := req.validate(); err != nil {
			return changeStatusRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.EditPermission, policy.GroupType, req.id)
		if err != nil {
			return changeStatusRes{}, err
		}
		group, err := svc.EnableGroup(ctx, session, req.id)
		if err != nil {
			return changeStatusRes{}, err
		}
		return changeStatusRes{Group: group}, nil
	}
}

func DisableGroupEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeGroupStatusReq)
		if err := req.validate(); err != nil {
			return changeStatusRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.EditPermission, policy.GroupType, req.id)
		if err != nil {
			return changeStatusRes{}, err
		}
		group, err := svc.DisableGroup(ctx, session, req.id)
		if err != nil {
			return changeStatusRes{}, err
		}
		return changeStatusRes{Group: group}, nil
	}
}

func ListGroupsEndpoint(svc groups.Service, authClient auth.AuthClient, groupType, memberKind string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if memberKind != "" {
			req.memberKind = memberKind
		}
		if err := req.validate(); err != nil {
			if groupType == groupTypeChannels {
				return channelPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
			}
			return groupPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			if groupType == groupTypeChannels {
				return channelPageRes{}, err
			}
			return groupPageRes{}, err
		}
		switch req.memberKind {
		case policy.ThingsKind:
			if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.ViewPermission, policy.ThingType, req.memberID); err != nil {
				if groupType == groupTypeChannels {
					return channelPageRes{}, err
				}
				return groupPageRes{}, err
			}
		case policy.GroupsKind:
			if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, req.Page.Permission, policy.GroupType, req.memberID); err != nil {
				if groupType == groupTypeChannels {
					return channelPageRes{}, err
				}
				return groupPageRes{}, err
			}
		case policy.ChannelsKind:
			if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.ViewPermission, policy.GroupType, req.memberID); err != nil {
				if groupType == groupTypeChannels {
					return channelPageRes{}, err
				}
				return groupPageRes{}, err
			}
		case policy.UsersKind:
			switch {
			case req.memberID != "" && session.UserID != req.memberID:
				if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.AdminPermission, policy.DomainType, session.DomainID); err != nil {
					if groupType == groupTypeChannels {
						return channelPageRes{}, err
					}
					return groupPageRes{}, err
				}
			default:
				switch checkSuperAdmin(ctx, authClient, session.DomainUserID) {
				case nil:
					session.SuperAdmin = true
				default:
					if _, err := authorize(ctx, authClient, "", policy.UserType, policy.UsersKind, session.DomainUserID, policy.MembershipPermission, policy.DomainType, session.DomainID); err != nil {
						if groupType == groupTypeChannels {
							return channelPageRes{}, err
						}
						return groupPageRes{}, err
					}
				}
			}
		}

		page, err := svc.ListGroups(ctx, session, req.memberKind, req.memberID, req.Page)
		if err != nil {
			if groupType == groupTypeChannels {
				return channelPageRes{}, err
			}
			return groupPageRes{}, err
		}

		if req.tree {
			return buildGroupsResponseTree(page), nil
		}
		filterByID := req.Page.ParentID != ""

		if groupType == groupTypeChannels {
			return buildChannelsResponse(page, filterByID), nil
		}
		return buildGroupsResponse(page, filterByID), nil
	}
}

func ListMembersEndpoint(svc groups.Service, authClient auth.AuthClient, memberKind string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if memberKind != "" {
			req.memberKind = memberKind
		}
		if err := req.validate(); err != nil {
			return listMembersRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		if _, err := authorize(ctx, authClient, "", policy.UserType, policy.TokenKind, req.token, policy.ViewPermission, policy.GroupType, req.groupID); err != nil {
			return listMembersRes{}, err
		}

		page, err := svc.ListMembers(ctx, req.groupID, req.permission, req.memberKind)
		if err != nil {
			return listMembersRes{}, err
		}

		return listMembersRes{
			pageRes: pageRes{
				Limit:  page.Limit,
				Offset: page.Offset,
				Total:  page.Total,
			},
			Members: page.Members,
		}, nil
	}
}

func AssignMembersEndpoint(svc groups.Service, authClient auth.AuthClient, relation, memberKind string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignReq)
		if relation != "" {
			req.Relation = relation
		}
		if memberKind != "" {
			req.MemberKind = memberKind
		}
		if err := req.validate(); err != nil {
			return assignRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return assignRes{}, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.groupID); err != nil {
			return assignRes{}, err
		}
		if err := svc.Assign(ctx, session, req.groupID, req.Relation, req.MemberKind, req.Members...); err != nil {
			return assignRes{}, err
		}
		return assignRes{assigned: true}, nil
	}
}

func UnassignMembersEndpoint(svc groups.Service, authClient auth.AuthClient, relation, memberKind string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignReq)
		if relation != "" {
			req.Relation = relation
		}
		if memberKind != "" {
			req.MemberKind = memberKind
		}
		if err := req.validate(); err != nil {
			return unassignRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return unassignRes{}, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.EditPermission, policy.GroupType, req.groupID); err != nil {
			return unassignRes{}, err
		}

		if err := svc.Unassign(ctx, session, req.groupID, req.Relation, req.MemberKind, req.Members...); err != nil {
			return unassignRes{}, err
		}
		return unassignRes{unassigned: true}, nil
	}
}

func DeleteGroupEndpoint(svc groups.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return deleteGroupRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, err := identify(ctx, authClient, req.token)
		if err != nil {
			return deleteGroupRes{}, err
		}
		if _, err := authorize(ctx, authClient, session.DomainID, policy.UserType, policy.UsersKind, session.DomainUserID, policy.DeletePermission, policy.GroupType, req.id); err != nil {
			return deleteGroupRes{}, err
		}
		if err := svc.DeleteGroup(ctx, req.id); err != nil {
			return deleteGroupRes{}, err
		}
		return deleteGroupRes{deleted: true}, nil
	}
}

func buildGroupsResponseTree(page groups.Page) groupPageRes {
	groupsMap := map[string]*groups.Group{}
	// Parents' map keeps its array of children.
	parentsMap := map[string][]*groups.Group{}
	for i := range page.Groups {
		if _, ok := groupsMap[page.Groups[i].ID]; !ok {
			groupsMap[page.Groups[i].ID] = &page.Groups[i]
			parentsMap[page.Groups[i].ID] = make([]*groups.Group, 0)
		}
	}

	for _, group := range groupsMap {
		if children, ok := parentsMap[group.Parent]; ok {
			children = append(children, group)
			parentsMap[group.Parent] = children
		}
	}

	res := groupPageRes{
		pageRes: pageRes{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
			Level:  page.Level,
		},
		Groups: []viewGroupRes{},
	}

	for _, group := range groupsMap {
		if children, ok := parentsMap[group.ID]; ok {
			group.Children = children
		}
	}

	for _, group := range groupsMap {
		view := toViewGroupRes(*group)
		if children, ok := parentsMap[group.Parent]; len(children) == 0 || !ok {
			res.Groups = append(res.Groups, view)
		}
	}

	return res
}

func toViewGroupRes(group groups.Group) viewGroupRes {
	view := viewGroupRes{
		Group: group,
	}
	return view
}

func buildGroupsResponse(gp groups.Page, filterByID bool) groupPageRes {
	res := groupPageRes{
		pageRes: pageRes{
			Total: gp.Total,
			Level: gp.Level,
		},
		Groups: []viewGroupRes{},
	}

	for _, group := range gp.Groups {
		view := viewGroupRes{
			Group: group,
		}
		if filterByID && group.Level == 0 {
			continue
		}
		res.Groups = append(res.Groups, view)
	}

	return res
}

func buildChannelsResponse(cp groups.Page, filterByID bool) channelPageRes {
	res := channelPageRes{
		pageRes: pageRes{
			Total: cp.Total,
			Level: cp.Level,
		},
		Channels: []viewGroupRes{},
	}

	for _, channel := range cp.Groups {
		if filterByID && channel.Level == 0 {
			continue
		}
		view := viewGroupRes{
			Group: channel,
		}
		res.Channels = append(res.Channels, view)
	}

	return res
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
