// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	smqauth "github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
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

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(svc users.Service, authz smqauthz.Authorization, selfRegister bool) users.Service {
	return &authorizationMiddleware{
		svc:          svc,
		authz:        authz,
		selfRegister: selfRegister,
	}
}

func (am *authorizationMiddleware) Register(ctx context.Context, session authn.Session, user users.User, selfRegister bool) (users.User, error) {
	if selfRegister {
		if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
			session.SuperAdmin = true
		}
	}

	return am.svc.Register(ctx, session, user, selfRegister)
}

func (am *authorizationMiddleware) View(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.ReadOp,
			EntityID:   id,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.View(ctx, session, id)
}

func (am *authorizationMiddleware) ViewProfile(ctx context.Context, session authn.Session) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.ReadOp,
			EntityID:   session.UserID,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}
	return am.svc.ViewProfile(ctx, session)
}

func (am *authorizationMiddleware) ListUsers(ctx context.Context, session authn.Session, pm users.Page) (users.UsersPage, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.ListOp,
			EntityID:   smqauth.AnyIDs,
		}); err != nil {
			return users.UsersPage{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ListUsers(ctx, session, pm)
}

func (am *authorizationMiddleware) SearchUsers(ctx context.Context, pm users.Page) (users.UsersPage, error) {
	return am.svc.SearchUsers(ctx, pm)
}

func (am *authorizationMiddleware) Update(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   user.ID,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.Update(ctx, session, user)
}

func (am *authorizationMiddleware) UpdateTags(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   user.ID,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateTags(ctx, session, user)
}

func (am *authorizationMiddleware) UpdateEmail(ctx context.Context, session authn.Session, id, email string) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   id,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}
	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateEmail(ctx, session, id, email)
}

func (am *authorizationMiddleware) UpdateUsername(ctx context.Context, session authn.Session, id, username string) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   id,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateUsername(ctx, session, id, username)
}

func (am *authorizationMiddleware) UpdateProfilePicture(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   user.ID,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.UpdateProfilePicture(ctx, session, user)
}

func (am *authorizationMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	return am.svc.GenerateResetToken(ctx, email, host)
}

func (am *authorizationMiddleware) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   session.UserID,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	return am.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
}

func (am *authorizationMiddleware) ResetSecret(ctx context.Context, session authn.Session, secret string) error {
	return am.svc.ResetSecret(ctx, session, secret)
}

func (am *authorizationMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) error {
	return am.svc.SendPasswordReset(ctx, host, email, user, token)
}

func (am *authorizationMiddleware) UpdateRole(ctx context.Context, session authn.Session, user users.User) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   user.ID,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		return users.User{}, err
	}
	session.SuperAdmin = true
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, user.ID, policies.MembershipPermission, policies.PlatformType, policies.SuperMQObject); err != nil {
		return users.User{}, err
	}

	return am.svc.UpdateRole(ctx, session, user)
}

func (am *authorizationMiddleware) Enable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   id,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.Enable(ctx, session, id)
}

func (am *authorizationMiddleware) Disable(ctx context.Context, session authn.Session, id string) (users.User, error) {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.UpdateOp,
			EntityID:   id,
		}); err != nil {
			return users.User{}, errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.Disable(ctx, session, id)
}

func (am *authorizationMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	if session.Type == authn.PersonalAccessToken {
		if err := am.authz.AuthorizePAT(ctx, smqauthz.PatReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityType: smqauth.UsersType,
			Operation:  smqauth.DeleteOp,
			EntityID:   id,
		}); err != nil {
			return errors.Wrap(svcerr.ErrUnauthorizedPAT, err)
		}
	}

	if err := am.checkSuperAdmin(ctx, session.UserID); err == nil {
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
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, user.ID, policies.MembershipPermission, policies.PlatformType, policies.SuperMQObject); err == nil {
		return nil
	}
	return am.svc.OAuthAddUserPolicy(ctx, user)
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, adminID string) error {
	if err := am.authz.Authorize(ctx, smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     adminID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	}); err != nil {
		return err
	}
	return nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, domain, subjType, subjKind, subj, perm, objType, obj string) error {
	req := smqauthz.PolicyReq{
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
