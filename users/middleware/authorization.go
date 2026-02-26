// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/users"
)

var _ users.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc          users.Service
	authz        smqauthz.Authorization
	selfRegister bool
}

// NewAuthorization adds authorization to the users service.
func NewAuthorization(svc users.Service, authz smqauthz.Authorization, selfRegister bool) users.Service {
	return &authorizationMiddleware{svc: svc, authz: authz, selfRegister: selfRegister}
}

func (am *authorizationMiddleware) SendVerification(ctx context.Context, session authn.Session) error {
	return am.svc.SendVerification(ctx, session)
}

func (am *authorizationMiddleware) VerifyEmail(ctx context.Context, verificationToken string) (users.User, error) {
	return am.svc.VerifyEmail(ctx, verificationToken)
}

func (am *authorizationMiddleware) Register(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (users.User, error) {
	if selfRegister {
		if err := am.checkSuperAdmin(ctx, session); err == nil {
			session.SuperAdmin = true
		}
	}

	return am.svc.Register(ctx, session, user, selfRegister)
}

func (am *authorizationMiddleware) View(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.View(ctx, session, id)
}

func (am *authorizationMiddleware) ViewProfile(ctx context.Context, session authn.Session) (users.User, error) {
	return am.svc.ViewProfile(ctx, session)
}

func (am *authorizationMiddleware) ListUsers(ctx context.Context, session authn.Session, pm users.Page) (users.UsersPage, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ListUsers(ctx, session, pm)
}

func (am *authorizationMiddleware) SearchUsers(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	return am.svc.SearchUsers(ctx, pm)
}

func (am *authorizationMiddleware) Update(ctx context.Context, session authn.Session, id string, user users.UserReq) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.Update(ctx, session, id, user)
}

func (am *authorizationMiddleware) UpdateTags(ctx context.Context, session authn.Session, id string, user users.UserReq) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateTags(ctx, session, id, user)
}

func (am *authorizationMiddleware) UpdateEmail(ctx context.Context, session authn.Session, id, email string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateEmail(ctx, session, id, email)
}

func (am *authorizationMiddleware) UpdateUsername(ctx context.Context, session authn.Session, id, username string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateUsername(ctx, session, id, username)
}

func (am *authorizationMiddleware) UpdateProfilePicture(ctx context.Context, session authn.Session, id string, usr users.UserReq) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateProfilePicture(ctx, session, id, usr)
}

func (am *authorizationMiddleware) SendPasswordReset(ctx context.Context, email string) error {
	return am.svc.SendPasswordReset(ctx, email)
}

func (am *authorizationMiddleware) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	return am.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
}

func (am *authorizationMiddleware) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	return am.svc.ResetSecret(ctx, session, secret)
}

func (am *authorizationMiddleware) UpdateRole(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err != nil {
		return users.User{}, err
	}
	session.SuperAdmin = true
	if err := am.authorize(ctx, session, "", policies.UserType, policies.UsersKind, user.ID, policies.MembershipPermission, policies.PlatformType, policies.SuperMQObject); err != nil {
		return users.User{}, err
	}

	return am.svc.UpdateRole(ctx, session, user)
}

func (am *authorizationMiddleware) Enable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.Enable(ctx, session, id)
}

func (am *authorizationMiddleware) Disable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.Disable(ctx, session, id)
}

func (am *authorizationMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.Delete(ctx, session, id)
}

func (am *authorizationMiddleware) Identify(ctx context.Context, session authn.Session) (string, error) {
	return am.svc.Identify(ctx, session)
}

func (am *authorizationMiddleware) IssueToken(ctx context.Context, username, secret string) (*grpcTokenV1.Token, error) {
	return am.svc.IssueToken(ctx, username, secret)
}

func (am *authorizationMiddleware) RefreshToken(ctx context.Context, session authn.Session, refreshToken string) (*grpcTokenV1.Token, error) {
	return am.svc.RefreshToken(ctx, session, refreshToken)
}

func (am *authorizationMiddleware) OAuthCallback(ctx context.Context, user users.User) (users.User, error) {
	return am.svc.OAuthCallback(ctx, user)
}

func (am *authorizationMiddleware) OAuthAddUserPolicy(ctx context.Context, user users.User) error {
	if err := am.authorize(ctx, authn.Session{}, "", policies.UserType, policies.UsersKind, user.ID, policies.MembershipPermission, policies.PlatformType, policies.SuperMQObject); err == nil {
		return nil
	}
	return am.svc.OAuthAddUserPolicy(ctx, user)
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, session authn.Session) error {
	if session.Role != authn.AdminRole {
		return svcerr.ErrSuperAdminAction
	}
	if err := am.authz.Authorize(ctx, smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     session.UserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	}, nil); err != nil {
		return err
	}
	return nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, session authn.Session, domain, subjType, subjKind, subj, perm, objType, obj string) error {
	req := smqauthz.PolicyReq{
		Domain:      domain,
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}

	var pat *smqauthz.PATReq
	if session.PatID != "" {
		pat = &smqauthz.PATReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityID:   subj,
			EntityType: auth.UsersType.String(),
			Operation:  perm,
			Domain:     domain,
		}
	}

	if err := am.authz.Authorize(ctx, req, pat); err != nil {
		return err
	}
	return nil
}
