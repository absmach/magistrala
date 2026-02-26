// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/domains/operations"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
	rolemgr "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
)

var _ domains.Service = (*authorizationMiddleware)(nil)

// ErrMemberExist indicates that the user is already a member of the domain.
var ErrMemberExist = errors.New("user is already a member of the domain")

type authorizationMiddleware struct {
	svc         domains.Service
	authz       smqauthz.Authorization
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rOps        permissions.Operations[permissions.RoleOperation]
	rolemgr.RoleManagerAuthorizationMiddleware
}

// NewAuthorization adds authorization to the domains service.
func NewAuthorization(entityType string, svc domains.Service, authz smqauthz.Authorization, entitiesOps permissions.EntitiesOperations[permissions.Operation], domainRoleOps permissions.Operations[permissions.RoleOperation]) (domains.Service, error) {
	if err := entitiesOps.Validate(); err != nil {
		return &authorizationMiddleware{}, err
	}

	ram, err := rolemgr.NewAuthorization(entityType, svc, authz, domainRoleOps)
	if err != nil {
		return &authorizationMiddleware{}, err
	}
	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		entitiesOps:                        entitiesOps,
		rOps:                               domainRoleOps,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (domains.Domain, []roles.RoleProvision, error) {
	return am.svc.CreateDomain(ctx, session, d)
}

func (am *authorizationMiddleware) RetrieveDomain(ctx context.Context, session authn.Session, id string, withRoles bool) (domains.Domain, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
		return am.svc.RetrieveDomain(ctx, session, id, withRoles)
	}

	if err := am.authorize(ctx, session, policies.DomainType, operations.OpRetrieveDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}

	return am.svc.RetrieveDomain(ctx, session, id, withRoles)
}

func (am *authorizationMiddleware) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	if err := am.authorize(ctx, session, policies.DomainType, operations.OpUpdateDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}

	return am.svc.UpdateDomain(ctx, session, id, d)
}

func (am *authorizationMiddleware) EnableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	if err := am.authorize(ctx, session, policies.DomainType, operations.OpEnableDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}

	return am.svc.EnableDomain(ctx, session, id)
}

func (am *authorizationMiddleware) DisableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	if err := am.authorize(ctx, session, policies.DomainType, operations.OpDisableDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}

	return am.svc.DisableDomain(ctx, session, id)
}

func (am *authorizationMiddleware) FreezeDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	// Only SuperAdmin can freeze the domain
	if err := am.authz.Authorize(ctx, authz.PolicyReq{
		Subject:     session.UserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Permission:  policies.AdminPermission,
		Object:      policies.SuperMQObject,
		ObjectType:  policies.PlatformType,
	}, nil); err != nil {
		return domains.Domain{}, err
	}

	return am.svc.FreezeDomain(ctx, session, id)
}

func (am *authorizationMiddleware) ListDomains(ctx context.Context, session authn.Session, page domains.Page) (domains.DomainsPage, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ListDomains(ctx, session, page)
}

func (am *authorizationMiddleware) SendInvitation(ctx context.Context, session authn.Session, invitation domains.Invitation) (domains.Invitation, error) {
	if err := am.authorize(ctx, session, policies.DomainType, operations.OpSendDomainInvitation, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Invitation{}, err
	}

	if err := am.checkAdmin(ctx, session); err != nil {
		return domains.Invitation{}, err
	}

	return am.svc.SendInvitation(ctx, session, invitation)
}

func (am *authorizationMiddleware) ListInvitations(ctx context.Context, session authn.Session, page domains.InvitationPageMeta) (invs domains.InvitationPage, err error) {
	return am.svc.ListInvitations(ctx, session, page)
}

func (am *authorizationMiddleware) ListDomainInvitations(ctx context.Context, session authn.Session, page domains.InvitationPageMeta) (invs domains.InvitationPage, err error) {
	if err := am.authorize(ctx, session, policies.DomainType, operations.OpListDomainInvitations, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.InvitationPage{}, err
	}

	return am.svc.ListDomainInvitations(ctx, session, page)
}

func (am *authorizationMiddleware) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (inv domains.Invitation, err error) {
	return am.svc.AcceptInvitation(ctx, session, domainID)
}

func (am *authorizationMiddleware) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (domains.Invitation, error) {
	return am.svc.RejectInvitation(ctx, session, domainID)
}

func (am *authorizationMiddleware) DeleteInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) (err error) {
	if err := am.authorize(ctx, session, policies.DomainType, operations.OpDeleteDomainInvitation, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return err
	}

	return am.svc.DeleteInvitation(ctx, session, inviteeUserID, domainID)
}

func (am *authorizationMiddleware) authorize(ctx context.Context, session authn.Session, entityType string, op permissions.Operation, authReq authz.PolicyReq) error {
	authReq.Domain = session.DomainID

	perm, err := am.entitiesOps.GetPermission(entityType, op)
	if err != nil {
		return err
	}

	authReq.Permission = perm.String()

	var pat *smqauthz.PATReq
	if session.PatID != "" {
		entityID := authReq.Object
		opName := am.entitiesOps.OperationName(entityType, op)
		pat = &smqauthz.PATReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityID:   entityID,
			EntityType: auth.DomainsType.String(),
			Operation:  opName,
			Domain:     session.DomainID,
		}
	}

	if err := am.authz.Authorize(ctx, authReq, pat); err != nil {
		return err
	}

	return nil
}

// checkAdmin checks if the given user is a domain or platform administrator.
func (am *authorizationMiddleware) checkAdmin(ctx context.Context, session authn.Session) error {
	req := smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}
	if err := am.authz.Authorize(ctx, req, nil); err == nil {
		return nil
	}

	req = smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.UserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.SuperMQObject,
	}

	if err := am.authz.Authorize(ctx, req, nil); err == nil {
		return nil
	}

	return svcerr.ErrAuthorization
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
