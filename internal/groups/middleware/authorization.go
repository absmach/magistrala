// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/policies"
)

var _ groups.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc        groups.Service
	authClient auth.AuthClient
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(svc groups.Service, authClient auth.AuthClient) groups.Service {
	return &authorizationMiddleware{
		svc:        svc,
		authClient: authClient,
	}
}

func (am *authorizationMiddleware) CreateGroup(ctx context.Context, session auth.Session, kind string, g groups.Group) (groups.Group, error) {
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.CreatePermission, policies.DomainType, session.DomainID); err != nil {
		return groups.Group{}, err
	}
	if g.Parent != "" {
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.GroupType, g.Parent); err != nil {
			return groups.Group{}, err
		}
	}

	return am.svc.CreateGroup(ctx, session, kind, g)
}

func (am *authorizationMiddleware) UpdateGroup(ctx context.Context, session auth.Session, g groups.Group) (groups.Group, error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.GroupType, g.ID); err != nil {
		return groups.Group{}, err
	}

	return am.svc.UpdateGroup(ctx, session, g)
}

func (am *authorizationMiddleware) ViewGroup(ctx context.Context, session auth.Session, id string) (groups.Group, error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.ViewPermission, policies.GroupType, id); err != nil {
		return groups.Group{}, err
	}

	return am.svc.ViewGroup(ctx, session, id)
}

func (am *authorizationMiddleware) ViewGroupPerms(ctx context.Context, session auth.Session, id string) ([]string, error) {
	return am.svc.ViewGroupPerms(ctx, session, id)
}

func (am *authorizationMiddleware) ListGroups(ctx context.Context, session auth.Session, memberKind, memberID string, gm groups.Page) (groups.Page, error) {
	switch memberKind {
	case policies.ThingsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.ViewPermission, policies.ThingType, memberID); err != nil {
			return groups.Page{}, err
		}
	case policies.GroupsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, gm.Permission, policies.GroupType, memberID); err != nil {
			return groups.Page{}, err
		}
	case policies.ChannelsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.ViewPermission, policies.GroupType, memberID); err != nil {
			return groups.Page{}, err
		}
	case policies.UsersKind:
		switch {
		case memberID != "" && session.UserID != memberID:
			if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.AdminPermission, policies.DomainType, session.DomainID); err != nil {
				return groups.Page{}, err
			}
		default:
			err := am.checkSuperAdmin(ctx, session.UserID)
			switch {
			case err == nil:
				session.SuperAdmin = true
			default:
				if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID); err != nil {
					return groups.Page{}, err
				}
			}
		}
	default:
		return groups.Page{}, svcerr.ErrAuthorization
	}

	return am.svc.ListGroups(ctx, session, memberKind, memberID, gm)
}

func (am *authorizationMiddleware) ListMembers(ctx context.Context, session auth.Session, groupID, permission, memberKind string) (groups.MembersPage, error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.ViewPermission, policies.GroupType, groupID); err != nil {
		return groups.MembersPage{}, err
	}

	return am.svc.ListMembers(ctx, session, groupID, permission, memberKind)
}

func (am *authorizationMiddleware) EnableGroup(ctx context.Context, session auth.Session, id string) (groups.Group, error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.GroupType, id); err != nil {
		return groups.Group{}, err
	}

	return am.svc.EnableGroup(ctx, session, id)
}

func (am *authorizationMiddleware) DisableGroup(ctx context.Context, session auth.Session, id string) (groups.Group, error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.GroupType, id); err != nil {
		return groups.Group{}, err
	}

	return am.svc.DisableGroup(ctx, session, id)
}

func (am *authorizationMiddleware) DeleteGroup(ctx context.Context, session auth.Session, id string) error {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.DeletePermission, policies.GroupType, id); err != nil {
		return err
	}

	return am.svc.DeleteGroup(ctx, session, id)
}

func (am *authorizationMiddleware) Assign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) (err error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.GroupType, groupID); err != nil {
		return err
	}

	return am.svc.Assign(ctx, session, groupID, relation, memberKind, memberIDs...)
}

func (am *authorizationMiddleware) Unassign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) (err error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.GroupType, groupID); err != nil {
		return err
	}

	return am.svc.Unassign(ctx, session, groupID, relation, memberKind, memberIDs...)
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, adminID string) error {
	if _, err := am.authClient.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: policies.UserType,
		Subject:     adminID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	}); err != nil {
		return err
	}
	return nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, domain, subjType, subjKind, subj, perm, objType, obj string) error {
	req := &magistrala.AuthorizeReq{
		Domain:      domain,
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	res, err := am.authClient.Authorize(ctx, req)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}
