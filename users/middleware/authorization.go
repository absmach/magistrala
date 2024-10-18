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

func (am *authorizationMiddleware) RegisterUser(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (users.User, error) {
	if selfRegister {
		if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
			session.SuperAdmin = true
		}
	}

	return am.svc.RegisterUser(ctx, session, user, selfRegister)
}

func (am *authorizationMiddleware) ViewUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ViewUser(ctx, session, id)
}

func (am *authorizationMiddleware) ViewProfile(ctx context.Context, session authn.Session) (users.User, error) {
	return am.svc.ViewProfile(ctx, session)
}

func (am *authorizationMiddleware) ViewUserByUserName(ctx context.Context, session authn.Session, userName string) (users.User, error) {
	return am.svc.ViewUserByUserName(ctx, session, userName)
}

func (am *authorizationMiddleware) ListUsers(ctx context.Context, session authn.Session, pm users.Page) (users.UsersPage, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ListUsers(ctx, session, pm)
}

func (am *authorizationMiddleware) ListMembers(ctx context.Context, session authn.Session, objectKind, objectID string, pm users.Page) (users.MembersPage, error) {
	if session.DomainUserID == "" {
		return users.MembersPage{}, svcerr.ErrDomainAuthorization
	}
	switch objectKind {
	case policies.GroupsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, mgauth.SwitchToPermission(pm.Permission), policies.GroupType, objectID); err != nil {
			return users.MembersPage{}, err
		}
	case policies.DomainsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, mgauth.SwitchToPermission(pm.Permission), policies.DomainType, objectID); err != nil {
			return users.MembersPage{}, err
		}
	case policies.ThingsKind:
		if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, mgauth.SwitchToPermission(pm.Permission), policies.ThingType, objectID); err != nil {
			return users.MembersPage{}, err
		}
	default:
		return users.MembersPage{}, svcerr.ErrAuthorization
	}

	return am.svc.ListMembers(ctx, session, objectKind, objectID, pm)
}

func (am *authorizationMiddleware) SearchUsers(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	return am.svc.SearchUsers(ctx, pm)
}

func (am *authorizationMiddleware) UpdateUser(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateUser(ctx, session, user)
}

func (am *authorizationMiddleware) UpdateUserTags(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateUserTags(ctx, session, user)
}

func (am *authorizationMiddleware) UpdateUserIdentity(ctx context.Context, session authn.Session, id, identity string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateUserIdentity(ctx, session, id, identity)
}

func (am *authorizationMiddleware) UpdateUserNames(ctx context.Context, session authn.Session, usr users.User) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateUserNames(ctx, session, usr)
}

func (am *authorizationMiddleware) UpdateProfilePicture(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	return am.svc.UpdateProfilePicture(ctx, session, user)
}

func (am *authorizationMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	return am.svc.GenerateResetToken(ctx, email, host)
}

func (am *authorizationMiddleware) UpdateUserSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	return am.svc.UpdateUserSecret(ctx, session, oldSecret, newSecret)
}

func (am *authorizationMiddleware) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	return am.svc.ResetSecret(ctx, session, secret)
}

func (am *authorizationMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) error {
	return am.svc.SendPasswordReset(ctx, host, email, user, token)
}

func (am *authorizationMiddleware) UpdateUserRole(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, user.ID, policies.MembershipPermission, policies.PlatformType, policies.MagistralaObject); err != nil {
		return users.User{}, err
	}

	return am.svc.UpdateUserRole(ctx, session, user)
}

func (am *authorizationMiddleware) EnableUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.EnableUser(ctx, session, id)
}

func (am *authorizationMiddleware) DisableUser(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.DisableUser(ctx, session, id)
}

func (am *authorizationMiddleware) DeleteUser(ctx context.Context, session authn.Session, id string) error {
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.DeleteUser(ctx, session, id)
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

func (am *authorizationMiddleware) OAuthCallback(ctx context.Context, user users.User) (users.User, error) {
	return am.svc.OAuthCallback(ctx, user)
}

func (am *authorizationMiddleware) OAuthAddUserPolicy(ctx context.Context, user users.User) error {
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, user.ID, policies.MembershipPermission, policies.PlatformType, policies.MagistralaObject); err == nil {
		return nil
	}
	return am.svc.OAuthAddUserPolicy(ctx, user)
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
