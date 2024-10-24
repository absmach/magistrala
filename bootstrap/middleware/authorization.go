// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/policies"
)

var _ bootstrap.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc   bootstrap.Service
	authz mgauthz.Authorization
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(svc bootstrap.Service, authz mgauthz.Authorization) bootstrap.Service {
	return &authorizationMiddleware{
		svc:   svc,
		authz: authz,
	}
}

func (am *authorizationMiddleware) Add(ctx context.Context, session mgauthn.Session, token string, cfg bootstrap.Config) (bootstrap.Config, error) {
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.MembershipPermission, policies.DomainType, session.DomainID); err != nil {
		return bootstrap.Config{}, err
	}

	return am.svc.Add(ctx, session, token, cfg)
}

func (am *authorizationMiddleware) View(ctx context.Context, session mgauthn.Session, id string) (bootstrap.Config, error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.ViewPermission, policies.ClientType, id); err != nil {
		return bootstrap.Config{}, err
	}

	return am.svc.View(ctx, session, id)
}

func (am *authorizationMiddleware) Update(ctx context.Context, session mgauthn.Session, cfg bootstrap.Config) error {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.ClientType, cfg.ClientID); err != nil {
		return err
	}

	return am.svc.Update(ctx, session, cfg)
}

func (am *authorizationMiddleware) UpdateCert(ctx context.Context, session mgauthn.Session, clientID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.ClientType, clientID); err != nil {
		return bootstrap.Config{}, err
	}

	return am.svc.UpdateCert(ctx, session, clientID, clientCert, clientKey, caCert)
}

func (am *authorizationMiddleware) UpdateConnections(ctx context.Context, session mgauthn.Session, token, id string, connections []string) error {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.EditPermission, policies.ClientType, id); err != nil {
		return err
	}

	return am.svc.UpdateConnections(ctx, session, token, id, connections)
}

func (am *authorizationMiddleware) List(ctx context.Context, session mgauthn.Session, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
	if err := am.checkSuperAdmin(ctx, session.DomainUserID); err == nil {
		session.SuperAdmin = true
	}
	if err := am.authorize(ctx, "", policies.UserType, policies.UsersKind, session.DomainUserID, policies.AdminPermission, policies.DomainType, session.DomainID); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.List(ctx, session, filter, offset, limit)
}

func (am *authorizationMiddleware) Remove(ctx context.Context, session mgauthn.Session, id string) error {
	if err := am.authorize(ctx, session.DomainID, policies.UserType, policies.UsersKind, session.DomainUserID, policies.DeletePermission, policies.ClientType, id); err != nil {
		return err
	}

	return am.svc.Remove(ctx, session, id)
}

func (am *authorizationMiddleware) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (bootstrap.Config, error) {
	return am.svc.Bootstrap(ctx, externalKey, externalID, secure)
}

func (am *authorizationMiddleware) ChangeState(ctx context.Context, session mgauthn.Session, token, id string, state bootstrap.State) error {
	return am.svc.ChangeState(ctx, session, token, id, state)
}

func (am *authorizationMiddleware) UpdateChannelHandler(ctx context.Context, channel bootstrap.Channel) error {
	return am.svc.UpdateChannelHandler(ctx, channel)
}

func (am *authorizationMiddleware) RemoveConfigHandler(ctx context.Context, id string) error {
	return am.svc.RemoveConfigHandler(ctx, id)
}

func (am *authorizationMiddleware) RemoveChannelHandler(ctx context.Context, id string) error {
	return am.svc.RemoveChannelHandler(ctx, id)
}

func (am *authorizationMiddleware) ConnectClientHandler(ctx context.Context, channelID, clientID string) error {
	return am.svc.ConnectClientHandler(ctx, channelID, clientID)
}

func (am *authorizationMiddleware) DisconnectClientHandler(ctx context.Context, channelID, clientID string) error {
	return am.svc.DisconnectClientHandler(ctx, channelID, clientID)
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
