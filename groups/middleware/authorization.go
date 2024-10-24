// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/policies"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
	"github.com/absmach/magistrala/pkg/svcutil"
)

var errParentUnAuthz = errors.New("parent group unauthorized")

var _ groups.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc   groups.Service
	authz mgauthz.Authorization
	opp   svcutil.OperationPerm
	rmMW.RoleManagerAuthorizationMiddleware
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(entityType string, svc groups.Service, authz mgauthz.Authorization, groupsOpPerm, rolesOpPerm map[svcutil.Operation]svcutil.Permission) (groups.Service, error) {
	opp := groups.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(groupsOpPerm); err != nil {
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

func (am *authorizationMiddleware) CreateGroup(ctx context.Context, session authn.Session, g groups.Group) (groups.Group, error) {

	if err := am.authorize(ctx, groups.OpCreateGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return groups.Group{}, errors.Wrap(errParentUnAuthz, err)
	}

	if g.Parent != "" {
		if err := am.authorize(ctx, groups.OpAddChildrenGroups, mgauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Subject:     session.DomainUserID,
			Object:      g.Parent,
			ObjectType:  policies.GroupType,
		}); err != nil {
			return groups.Group{}, errors.Wrap(errParentUnAuthz, err)
		}
	}

	return am.svc.CreateGroup(ctx, session, g)
}

func (am *authorizationMiddleware) UpdateGroup(ctx context.Context, session authn.Session, g groups.Group) (groups.Group, error) {
	if err := am.authorize(ctx, groups.OpUpdateGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      g.ID,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, errors.Wrap(errParentUnAuthz, err)
	}

	return am.svc.UpdateGroup(ctx, session, g)
}

func (am *authorizationMiddleware) ViewGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	if err := am.authorize(ctx, groups.OpViewGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, errors.Wrap(errParentUnAuthz, err)
	}

	return am.svc.ViewGroup(ctx, session, id)
}

func (am *authorizationMiddleware) ListGroups(ctx context.Context, session authn.Session, gm groups.PageMeta) (groups.Page, error) {
	err := am.checkSuperAdmin(ctx, session.UserID)
	switch {
	case err == nil:
		session.SuperAdmin = true
	default:
		if err := am.authorize(ctx, groups.OpListGroups, mgauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Subject:     session.DomainUserID,
			Object:      session.DomainID,
			ObjectType:  policies.DomainType,
		}); err != nil {
			return groups.Page{}, errors.Wrap(errParentUnAuthz, err)
		}
	}

	return am.svc.ListGroups(ctx, session, gm)
}

func (am *authorizationMiddleware) EnableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	if err := am.authorize(ctx, groups.OpEnableGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, err
	}

	return am.svc.EnableGroup(ctx, session, id)
}

func (am *authorizationMiddleware) DisableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	if err := am.authorize(ctx, groups.OpDisableGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, err
	}

	return am.svc.DisableGroup(ctx, session, id)
}

func (am *authorizationMiddleware) DeleteGroup(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, groups.OpDeleteGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return err
	}

	return am.svc.DeleteGroup(ctx, session, id)
}

func (am *authorizationMiddleware) RetrieveGroupHierarchy(ctx context.Context, session authn.Session, id string, hm groups.HierarchyPageMeta) (groups.HierarchyPage, error) {
	if err := am.authorize(ctx, groups.OpRetrieveGroupHierarchy, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.HierarchyPage{}, err
	}
	return am.svc.RetrieveGroupHierarchy(ctx, session, id, hm)
}

func (am *authorizationMiddleware) AddParentGroup(ctx context.Context, session authn.Session, id, parentID string) error {
	if err := am.authorize(ctx, groups.OpAddParentGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return err
	}

	if err := am.authorize(ctx, groups.OpAddChildrenGroups, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      parentID,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return err
	}
	return am.svc.AddParentGroup(ctx, session, id, parentID)
}

func (am *authorizationMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, groups.OpRemoveParentGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return err
	}

	return am.svc.RemoveParentGroup(ctx, session, id)
}

func (am *authorizationMiddleware) AddChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	if err := am.authorize(ctx, groups.OpAddChildrenGroups, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return err
	}

	for _, childID := range childrenGroupIDs {
		if err := am.authorize(ctx, groups.OpAddParentGroup, mgauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			Object:      childID,
			ObjectType:  policies.GroupType,
		}); err != nil {
			return err
		}
	}

	return am.svc.AddChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (am *authorizationMiddleware) RemoveChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	if err := am.authorize(ctx, groups.OpRemoveChildrenGroups, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return err
	}

	return am.svc.RemoveChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (am *authorizationMiddleware) RemoveAllChildrenGroups(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, groups.OpRemoveAllChildrenGroups, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return err
	}
	return am.svc.RemoveAllChildrenGroups(ctx, session, id)
}

func (am *authorizationMiddleware) ListChildrenGroups(ctx context.Context, session authn.Session, id string, pm groups.PageMeta) (groups.Page, error) {
	if err := am.authorize(ctx, groups.OpListChildrenGroups, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Page{}, err
	}

	return am.svc.ListChildrenGroups(ctx, session, id, pm)
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

func (am *authorizationMiddleware) authorize(ctx context.Context, op svcutil.Operation, pr authz.PolicyReq) error {
	perm, err := am.opp.GetPermission(op)
	if err != nil {
		return err
	}
	pr.Permission = perm.String()
	if err := am.authz.Authorize(ctx, pr); err != nil {
		return err
	}
	return nil
}
