// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/bootstrap"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
)

const (
	updatePermission = "update_permission"
	readPermission   = "read_permission"
	deletePermission = "delete_permission"

	createOperation            = "create"
	viewOperation              = "view"
	updateOperation            = "update"
	updateCertOperation        = "update_cert"
	listOperation = "list"
	removeOperation            = "remove"
	changeStateOperation       = "change_state"
)

var _ bootstrap.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc   bootstrap.Service
	authz authz.Authorization
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(svc bootstrap.Service, authz authz.Authorization) bootstrap.Service {
	return &authorizationMiddleware{
		svc:   svc,
		authz: authz,
	}
}

func (am *authorizationMiddleware) Add(ctx context.Context, session smqauthn.Session, token string, cfg bootstrap.Config) (bootstrap.Config, error) {
	if err := am.authorize(ctx, session, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID, createOperation, auth.AnyIDs); err != nil {
		return bootstrap.Config{}, err
	}

	return am.svc.Add(ctx, session, token, cfg)
}

func (am *authorizationMiddleware) View(ctx context.Context, session smqauthn.Session, id string) (bootstrap.Config, error) {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, readPermission, policies.ClientType, id, viewOperation, id); err != nil {
		return bootstrap.Config{}, err
	}

	return am.svc.View(ctx, session, id)
}

func (am *authorizationMiddleware) Update(ctx context.Context, session smqauthn.Session, cfg bootstrap.Config) error {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, updatePermission, policies.ClientType, cfg.ClientID, updateOperation, cfg.ClientID); err != nil {
		return err
	}

	return am.svc.Update(ctx, session, cfg)
}

func (am *authorizationMiddleware) UpdateCert(ctx context.Context, session smqauthn.Session, clientID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, updatePermission, policies.ClientType, clientID, updateCertOperation, clientID); err != nil {
		return bootstrap.Config{}, err
	}

	return am.svc.UpdateCert(ctx, session, clientID, clientCert, clientKey, caCert)
}

func (am *authorizationMiddleware) List(ctx context.Context, session smqauthn.Session, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}
	if err := am.authorize(ctx, session, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.AdminPermission, policies.DomainType, session.DomainID, listOperation, auth.AnyIDs); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.List(ctx, session, filter, offset, limit)
}

func (am *authorizationMiddleware) Remove(ctx context.Context, session smqauthn.Session, id string) error {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, deletePermission, policies.ClientType, id, removeOperation, id); err != nil {
		return err
	}

	return am.svc.Remove(ctx, session, id)
}

func (am *authorizationMiddleware) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (bootstrap.Config, error) {
	return am.svc.Bootstrap(ctx, externalKey, externalID, secure)
}

func (am *authorizationMiddleware) EnableConfig(ctx context.Context, session smqauthn.Session, id string) (bootstrap.Config, error) {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, updatePermission, policies.ClientType, id, changeStateOperation, id); err != nil {
		return bootstrap.Config{}, err
	}

	return am.svc.EnableConfig(ctx, session, id)
}

func (am *authorizationMiddleware) DisableConfig(ctx context.Context, session smqauthn.Session, id string) (bootstrap.Config, error) {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, updatePermission, policies.ClientType, id, changeStateOperation, id); err != nil {
		return bootstrap.Config{}, err
	}

	return am.svc.DisableConfig(ctx, session, id)
}

func (am *authorizationMiddleware) RemoveConfigHandler(ctx context.Context, id string) error {
	return am.svc.RemoveConfigHandler(ctx, id)
}

func (am *authorizationMiddleware) CreateProfile(ctx context.Context, session smqauthn.Session, p bootstrap.Profile) (bootstrap.Profile, error) {
	if err := am.authorize(ctx, session, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID, createOperation, auth.AnyIDs); err != nil {
		return bootstrap.Profile{}, err
	}
	return am.svc.CreateProfile(ctx, session, p)
}

func (am *authorizationMiddleware) ViewProfile(ctx context.Context, session smqauthn.Session, profileID string) (bootstrap.Profile, error) {
	if err := am.authorize(ctx, session, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID, viewOperation, auth.AnyIDs); err != nil {
		return bootstrap.Profile{}, err
	}
	return am.svc.ViewProfile(ctx, session, profileID)
}

func (am *authorizationMiddleware) UpdateProfile(ctx context.Context, session smqauthn.Session, p bootstrap.Profile) error {
	if err := am.authorize(ctx, session, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID, updateOperation, auth.AnyIDs); err != nil {
		return err
	}
	return am.svc.UpdateProfile(ctx, session, p)
}

func (am *authorizationMiddleware) ListProfiles(ctx context.Context, session smqauthn.Session, offset, limit uint64) (bootstrap.ProfilesPage, error) {
	if err := am.authorize(ctx, session, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID, listOperation, auth.AnyIDs); err != nil {
		return bootstrap.ProfilesPage{}, err
	}
	return am.svc.ListProfiles(ctx, session, offset, limit)
}

func (am *authorizationMiddleware) DeleteProfile(ctx context.Context, session smqauthn.Session, profileID string) error {
	if err := am.authorize(ctx, session, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID, removeOperation, auth.AnyIDs); err != nil {
		return err
	}
	return am.svc.DeleteProfile(ctx, session, profileID)
}

func (am *authorizationMiddleware) AssignProfile(ctx context.Context, session smqauthn.Session, configID, profileID string) error {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, updatePermission, policies.ClientType, configID, updateOperation, configID); err != nil {
		return err
	}
	return am.svc.AssignProfile(ctx, session, configID, profileID)
}

func (am *authorizationMiddleware) BindResources(ctx context.Context, session smqauthn.Session, token, configID string, bindings []bootstrap.BindingRequest) error {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, updatePermission, policies.ClientType, configID, updateOperation, configID); err != nil {
		return err
	}
	return am.svc.BindResources(ctx, session, token, configID, bindings)
}

func (am *authorizationMiddleware) ListBindings(ctx context.Context, session smqauthn.Session, configID string) ([]bootstrap.BindingSnapshot, error) {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, readPermission, policies.ClientType, configID, viewOperation, configID); err != nil {
		return nil, err
	}
	return am.svc.ListBindings(ctx, session, configID)
}

func (am *authorizationMiddleware) RefreshBindings(ctx context.Context, session smqauthn.Session, token, configID string) error {
	if err := am.authorize(ctx, session, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, updatePermission, policies.ClientType, configID, updateOperation, configID); err != nil {
		return err
	}
	return am.svc.RefreshBindings(ctx, session, token, configID)
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, session smqauthn.Session) error {
	if session.Role != smqauthn.SuperAdminRole {
		return svcerr.ErrSuperAdminAction
	}
	if err := am.authz.Authorize(ctx, authz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     session.UserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	}, nil); err != nil {
		return err
	}
	return nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, session smqauthn.Session, domain, subjType, subjKind, subj, perm, objType, obj, operation, entityID string) error {
	req := authz.PolicyReq{
		Domain:      domain,
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}

	var pat *authz.PATReq
	if session.PatID != "" {
		pat = &authz.PATReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityID:   entityID,
			EntityType: auth.BootstrapType.String(),
			Operation:  operation,
			Domain:     session.DomainID,
		}
	}

	if err := am.authz.Authorize(ctx, req, pat); err != nil {
		return err
	}
	return nil
}
