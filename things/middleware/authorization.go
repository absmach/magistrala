// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/things"
)

var _ things.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc   things.Service
	authz mgauthz.Authorization
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(svc things.Service, authz mgauthz.Authorization) things.Service {
	return &authorizationMiddleware{
		svc:   svc,
		authz: authz,
	}
}

func (am *authorizationMiddleware) CreateClients(ctx context.Context, session authn.Session, client ...things.Client) ([]things.Client, error) {
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.CreatePermission, policies.DomainType, session.DomainID); err != nil {
		return nil, err
	}

	return am.svc.CreateClients(ctx, session, client...)
}

func (am *authorizationMiddleware) View(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	if session.DomainUserID == "" {
		return things.Client{}, svcerr.ErrDomainAuthorization
	}
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.ViewPermission, policies.ThingType, id); err != nil {
		return things.Client{}, err
	}

	return am.svc.View(ctx, session, id)
}

func (am *authorizationMiddleware) ViewPerms(ctx context.Context, session authn.Session, id string) ([]string, error) {
	return am.svc.ViewPerms(ctx, session, id)
}

func (am *authorizationMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm things.Page) (things.ClientsPage, error) {
	if session.DomainUserID == "" {
		return things.ClientsPage{}, svcerr.ErrDomainAuthorization
	}
	switch {
	case reqUserID != "" && reqUserID != session.UserID:
		if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.AdminPermission, policies.DomainType, session.DomainID); err != nil {
			return things.ClientsPage{}, err
		}
	default:
		err := am.checkSuperAdmin(ctx, session.UserID)
		switch {
		case err == nil:
			session.SuperAdmin = true
		default:
			if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID); err != nil {
				return things.ClientsPage{}, err
			}
		}
	}

	return am.svc.ListClients(ctx, session, reqUserID, pm)
}

func (am *authorizationMiddleware) ListClientsByGroup(ctx context.Context, session authn.Session, groupID string, pm things.Page) (things.MembersPage, error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, pm.Permission, policies.GroupType, groupID); err != nil {
		return things.MembersPage{}, err
	}

	return am.svc.ListClientsByGroup(ctx, session, groupID, pm)
}

func (am *authorizationMiddleware) Update(ctx context.Context, session authn.Session, client things.Client) (things.Client, error) {
	if session.DomainUserID == "" {
		return things.Client{}, svcerr.ErrDomainAuthorization
	}
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.ThingType, client.ID); err != nil {
		return things.Client{}, err
	}

	return am.svc.Update(ctx, session, client)
}

func (am *authorizationMiddleware) UpdateTags(ctx context.Context, session authn.Session, client things.Client) (things.Client, error) {
	if session.DomainUserID == "" {
		return things.Client{}, svcerr.ErrDomainAuthorization
	}
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.ThingType, client.ID); err != nil {
		return things.Client{}, err
	}

	return am.svc.UpdateTags(ctx, session, client)
}

func (am *authorizationMiddleware) UpdateSecret(ctx context.Context, session authn.Session, id, key string) (things.Client, error) {
	if session.DomainUserID == "" {
		return things.Client{}, svcerr.ErrDomainAuthorization
	}
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.ThingType, id); err != nil {
		return things.Client{}, err
	}

	return am.svc.UpdateSecret(ctx, session, id, key)
}

func (am *authorizationMiddleware) Enable(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	if session.DomainUserID == "" {
		return things.Client{}, svcerr.ErrDomainAuthorization
	}
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.DeletePermission, policies.ThingType, id); err != nil {
		return things.Client{}, err
	}

	return am.svc.Enable(ctx, session, id)
}

func (am *authorizationMiddleware) Disable(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	if session.DomainUserID == "" {
		return things.Client{}, svcerr.ErrDomainAuthorization
	}
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.DeletePermission, policies.ThingType, id); err != nil {
		return things.Client{}, err
	}

	return am.svc.Disable(ctx, session, id)
}

func (am *authorizationMiddleware) Share(ctx context.Context, session authn.Session, id string, relation string, userids ...string) error {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.DeletePermission, policies.ThingType, id); err != nil {
		return err
	}

	return am.svc.Share(ctx, session, id, relation, userids...)
}

func (am *authorizationMiddleware) Unshare(ctx context.Context, session authn.Session, id string, relation string, userids ...string) error {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.DeletePermission, policies.ThingType, id); err != nil {
		return err
	}

	return am.svc.Unshare(ctx, session, id, relation, userids...)
}

func (am *authorizationMiddleware) Identify(ctx context.Context, key string) (string, error) {
	return am.svc.Identify(ctx, key)
}

func (am *authorizationMiddleware) Authorize(ctx context.Context, req things.AuthzReq) (string, error) {
	return am.svc.Authorize(ctx, req)
}

func (am *authorizationMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.DeletePermission, policies.ThingType, id); err != nil {
		return err
	}

	return am.svc.Delete(ctx, session, id)
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, adminID string) error {
	if err := am.authz.Authorize(ctx, mgauthz.PolicyReq{
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
	req := mgauthz.PolicyReq{
		Domain:      domain,
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}
	return nil
}
