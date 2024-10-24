// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/policies"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
	"github.com/absmach/magistrala/pkg/svcutil"
)

var _ domains.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc   domains.Service
	authz mgauthz.Authorization
	opp   svcutil.OperationPerm
	rmMW.RoleManagerAuthorizationMiddleware
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(entityType string, svc domains.Service, authz mgauthz.Authorization, domainsOpPerm, rolesOpPerm map[svcutil.Operation]svcutil.Permission) (domains.Service, error) {
	opp := domains.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(domainsOpPerm); err != nil {
		return nil, err
	}
	if err := opp.Validate(); err != nil {
		return nil, err
	}

	ram, err := rmMW.NewRoleManagerAuthorizationMiddleware(entityType, svc, authz, rolesOpPerm)
	if err != nil {
		return nil, err
	}
	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		opp:                                opp,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (domains.Domain, error) {
	return am.svc.CreateDomain(ctx, session, d)
}

func (am *authorizationMiddleware) RetrieveDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	if err := am.authorize(ctx, domains.OpRetrieveDomain, authz.PolicyReq{
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}
	return am.svc.RetrieveDomain(ctx, session, id)
}

func (am *authorizationMiddleware) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	if err := am.authorize(ctx, domains.OpUpdateDomain, authz.PolicyReq{
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
	if err := am.authorize(ctx, domains.OpEnableDomain, authz.PolicyReq{
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
	if err := am.authorize(ctx, domains.OpDisableDomain, authz.PolicyReq{
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
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Permission:  policies.AdminPermission,
		Object:      id,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return domains.Domain{}, err
	}
	return am.svc.FreezeDomain(ctx, session, id)
}

func (am *authorizationMiddleware) ListDomains(ctx context.Context, session authn.Session, page domains.Page) (domains.DomainsPage, error) {
	if err := am.authz.Authorize(ctx, authz.PolicyReq{
		Subject:     session.UserID,
		SubjectType: policies.UserType,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	}); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ListDomains(ctx, session, page)
}

func (am *authorizationMiddleware) DeleteUserFromDomains(ctx context.Context, id string) (err error) {
	return am.svc.DeleteUserFromDomains(ctx, id)
}

func (am *authorizationMiddleware) authorize(ctx context.Context, op svcutil.Operation, authReq authz.PolicyReq) error {
	perm, err := am.opp.GetPermission(op)
	if err != nil {
		return err
	}
	authReq.Permission = perm.String()

	if err := am.authz.Authorize(ctx, authReq); err != nil {
		return err
	}

	return nil
}
