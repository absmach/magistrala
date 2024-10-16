// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/clients"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/users"
)

var _ users.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc          users.Service
	authz        mgauthz.Authorization
	selfRegister bool
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(svc users.Service, authz mgauthz.Authorization, selfRegister bool) users.Service {
	return &authorizationMiddleware{
		svc:          svc,
		authz:        authz,
		selfRegister: selfRegister,
	}
}

func (am *authorizationMiddleware) RegisterClient(ctx context.Context, session authn.Session, client clients.Client, selfRegister bool) (clients.Client, error) {
	if selfRegister {
		if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
			session.SuperAdmin = true
		}
	}

	return am.svc.RegisterClient(ctx, session, client, selfRegister)
}

func (am *authorizationMiddleware) ViewClient(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ViewClient(ctx, session, id)
}

func (am *authorizationMiddleware) ViewProfile(ctx context.Context, session authn.Session) (clients.Client, error) {
	return am.svc.ViewProfile(ctx, session)
}

func (am *authorizationMiddleware) ListClients(ctx context.Context, session authn.Session, pm clients.Page) (clients.ClientsPage, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ListClients(ctx, session, pm)
}

func (am *authorizationMiddleware) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm clients.Page) (clients.MembersPage, error) {
	if session.DomainUserID == "" {
		return clients.MembersPage{}, svcerr.ErrDomainAuthorization
	}
	switch objectKind {
	case policies.GroupsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, mgauth.SwitchToPermission(pm.Permission), policies.GroupType, objectID); err != nil {
			return clients.MembersPage{}, err
		}
	case policies.DomainsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, mgauth.SwitchToPermission(pm.Permission), policies.DomainType, objectID); err != nil {
			return clients.MembersPage{}, err
		}
	case policies.ThingsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, mgauth.SwitchToPermission(pm.Permission), policies.ThingType, objectID); err != nil {
			return clients.MembersPage{}, err
		}
	default:
		return clients.MembersPage{}, svcerr.ErrAuthorization
	}

	return am.svc.ListMembers(ctx, session, objectKind, objectID, pm)
}

func (am *authorizationMiddleware) SearchUsers(ctx context.Context, pm clients.Page) (clients.ClientsPage, error) {
	return am.svc.SearchUsers(ctx, pm)
}

func (am *authorizationMiddleware) UpdateClient(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateClient(ctx, session, client)
}

func (am *authorizationMiddleware) UpdateClientTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateClientTags(ctx, session, client)
}

func (am *authorizationMiddleware) UpdateClientIdentity(ctx context.Context, session authn.Session, id, identity string) (clients.Client, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateClientIdentity(ctx, session, id, identity)
}

func (am *authorizationMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	return am.svc.GenerateResetToken(ctx, email, host)
}

func (am *authorizationMiddleware) UpdateClientSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (clients.Client, error) {
	return am.svc.UpdateClientSecret(ctx, session, oldSecret, newSecret)
}

func (am *authorizationMiddleware) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	return am.svc.ResetSecret(ctx, session, secret)
}

func (am *authorizationMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) error {
	return am.svc.SendPasswordReset(ctx, host, email, user, token)
}

func (am *authorizationMiddleware) UpdateClientRole(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, client.ID, policies.MembershipPermission, policies.PlatformType, policies.MagistralaObject); err != nil {
		return clients.Client{}, err
	}

	return am.svc.UpdateClientRole(ctx, session, client)
}

func (am *authorizationMiddleware) EnableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.EnableClient(ctx, session, id)
}

func (am *authorizationMiddleware) DisableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.DisableClient(ctx, session, id)
}

func (am *authorizationMiddleware) DeleteClient(ctx context.Context, session authn.Session, id string) error {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.DeleteClient(ctx, session, id)
}

func (am *authorizationMiddleware) Identify(ctx context.Context, session authn.Session) (string, error) {
	return am.svc.Identify(ctx, session)
}

func (am *authorizationMiddleware) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	return am.svc.IssueToken(ctx, identity, secret, domainID)
}

func (am *authorizationMiddleware) RefreshToken(ctx context.Context, session authn.Session, refreshToken, domainID string) (*magistrala.Token, error) {
	return am.svc.RefreshToken(ctx, session, refreshToken, domainID)
}

func (am *authorizationMiddleware) OAuthCallback(ctx context.Context, client clients.Client) (clients.Client, error) {
	return am.svc.OAuthCallback(ctx, client)
}

func (am *authorizationMiddleware) OAuthAddClientPolicy(ctx context.Context, client clients.Client) error {
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, client.ID, policies.MembershipPermission, policies.PlatformType, policies.MagistralaObject); err == nil {
		return nil
	}
	return am.svc.OAuthAddClientPolicy(ctx, client)
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, adminID string) error {
	if err := am.authz.Authorize(ctx, authz.PolicyReq{
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
	req := authz.PolicyReq{
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
