// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"

	"github.com/absmach/supermq/auth"
	dOperations "github.com/absmach/supermq/domains/operations"
	"github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/groups/operations"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
	rolemgr "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
)

var (
	errView                        = errors.New("not authorized to view group")
	errUpdate                      = errors.New("not authorized to update group")
	errUpdateTags                  = errors.New("not authorized to update group tags")
	errEnable                      = errors.New("not authorized to enable group")
	errDisable                     = errors.New("not authorized to disable group")
	errDelete                      = errors.New("not authorized to delete group")
	errViewHierarchy               = errors.New("not authorized to view group parent/children hierarchy")
	errListChildrenGroups          = errors.New("not authorized to view chidden groups of group")
	errSetParentGroup              = errors.New("not authorized to set parent group to group")
	errRemoveParentGroup           = errors.New("not authorized to remove parent group from group")
	errSetChildrenGroups           = errors.New("not authorized to set children groups to group")
	errRemoveChildrenGroups        = errors.New("not authorized to remove children groups from group")
	errParentGroupSetChildGroup    = errors.New("not authorized to set child group in parent group")
	errParentGroupRemoveChildGroup = errors.New("not authorized to remove child group from parent group")
	errChildGroupSetParentGroup    = errors.New("not authorized to set parent group to child group")
	errDomainCreateGroups          = errors.New("not authorized to create groups in domain")
	errDomainListGroups            = errors.New("not authorized to list groups in domain")
)

var _ groups.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc         groups.Service
	repo        groups.Repository
	authz       smqauthz.Authorization
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rolemgr.RoleManagerAuthorizationMiddleware
}

// NewAuthorization adds authorization to the groups service.
func NewAuthorization(
	entityType string,
	svc groups.Service,
	authz smqauthz.Authorization,
	repo groups.Repository,
	entitiesOps permissions.EntitiesOperations[permissions.Operation],
	roleOps permissions.Operations[permissions.RoleOperation],
) (groups.Service, error) {
	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}
	ram, err := rolemgr.NewAuthorization(policies.GroupType, svc, authz, roleOps)
	if err != nil {
		return nil, err
	}

	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		repo:                               repo,
		entitiesOps:                        entitiesOps,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) CreateGroup(ctx context.Context, session authn.Session, g groups.Group) (groups.Group, []roles.RoleProvision, error) {
	if err := am.authorize(ctx, session, policies.DomainType, dOperations.OpCreateDomainGroups, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return groups.Group{}, []roles.RoleProvision{}, errors.Wrap(errDomainCreateGroups, err)
	}

	if g.Parent != "" {
		if err := am.authorize(ctx, session, policies.GroupType, operations.OpAddChildrenGroups, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Subject:     session.DomainUserID,
			Object:      g.Parent,
			ObjectType:  policies.GroupType,
		}); err != nil {
			return groups.Group{}, []roles.RoleProvision{}, errors.Wrap(errParentGroupSetChildGroup, err)
		}
	}

	return am.svc.CreateGroup(ctx, session, g)
}

func (am *authorizationMiddleware) UpdateGroup(ctx context.Context, session authn.Session, g groups.Group) (groups.Group, error) {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpUpdateGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      g.ID,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, errors.Wrap(errUpdate, err)
	}

	return am.svc.UpdateGroup(ctx, session, g)
}

func (am *authorizationMiddleware) UpdateGroupTags(ctx context.Context, session authn.Session, group groups.Group) (groups.Group, error) {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpUpdateGroupTags, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.GroupType,
		Object:      group.ID,
	}); err != nil {
		return groups.Group{}, errors.Wrap(errUpdateTags, err)
	}

	return am.svc.UpdateGroupTags(ctx, session, group)
}

func (am *authorizationMiddleware) ViewGroup(ctx context.Context, session authn.Session, id string, withRoles bool) (groups.Group, error) {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpViewGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, errors.Wrap(errView, err)
	}

	return am.svc.ViewGroup(ctx, session, id, withRoles)
}

func (am *authorizationMiddleware) ListGroups(ctx context.Context, session authn.Session, gm groups.PageMeta) (groups.Page, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
		return am.svc.ListGroups(ctx, session, gm)
	}
	if err := am.authorize(ctx, session, policies.DomainType, dOperations.OpListDomainGroups, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return groups.Page{}, errors.Wrap(errDomainListGroups, err)
	}

	return am.svc.ListGroups(ctx, session, gm)
}

func (am *authorizationMiddleware) ListUserGroups(ctx context.Context, session authn.Session, userID string, pm groups.PageMeta) (groups.Page, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
		return am.svc.ListGroups(ctx, session, pm)
	}
	if err := am.authorize(ctx, session, policies.DomainType, dOperations.OpListDomainGroups, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return groups.Page{}, errors.Wrap(errDomainListGroups, err)
	}

	return am.svc.ListUserGroups(ctx, session, userID, pm)
}

func (am *authorizationMiddleware) EnableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpEnableGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, errors.Wrap(errEnable, err)
	}

	return am.svc.EnableGroup(ctx, session, id)
}

func (am *authorizationMiddleware) DisableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpDisableGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, errors.Wrap(errDisable, err)
	}

	return am.svc.DisableGroup(ctx, session, id)
}

func (am *authorizationMiddleware) DeleteGroup(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpDeleteGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return errors.Wrap(errDelete, err)
	}

	return am.svc.DeleteGroup(ctx, session, id)
}

func (am *authorizationMiddleware) RetrieveGroupHierarchy(ctx context.Context, session authn.Session, id string, hm groups.HierarchyPageMeta) (groups.HierarchyPage, error) {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpRetrieveGroupHierarchy, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.HierarchyPage{}, errors.Wrap(errViewHierarchy, err)
	}

	return am.svc.RetrieveGroupHierarchy(ctx, session, id, hm)
}

func (am *authorizationMiddleware) AddParentGroup(ctx context.Context, session authn.Session, id, parentID string) error {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpAddParentGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return errors.Wrap(errSetParentGroup, err)
	}

	if err := am.authorize(ctx, session, policies.GroupType, operations.OpAddChildrenGroups, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      parentID,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return errors.Wrap(errParentGroupSetChildGroup, err)
	}

	return am.svc.AddParentGroup(ctx, session, id, parentID)
}

func (am *authorizationMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpRemoveParentGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return errors.Wrap(errRemoveParentGroup, err)
	}

	group, err := am.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if group.Parent != "" {
		if err := am.authorize(ctx, session, policies.GroupType, operations.OpRemoveParentGroup, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			Object:      group.Parent,
			ObjectType:  policies.GroupType,
		}); err != nil {
			return errors.Wrap(errParentGroupRemoveChildGroup, err)
		}
	}

	return am.svc.RemoveParentGroup(ctx, session, id)
}

func (am *authorizationMiddleware) AddChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpAddChildrenGroups, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return errors.Wrap(errSetChildrenGroups, err)
	}

	for _, childID := range childrenGroupIDs {
		if err := am.authorize(ctx, session, policies.GroupType, operations.OpAddParentGroup, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			Object:      childID,
			ObjectType:  policies.GroupType,
		}); err != nil {
			return errors.Wrap(errChildGroupSetParentGroup, errors.Wrap(fmt.Errorf("child group id: %s", childID), err))
		}
	}

	return am.svc.AddChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (am *authorizationMiddleware) RemoveChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpRemoveChildrenGroups, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return errors.Wrap(errRemoveChildrenGroups, err)
	}

	return am.svc.RemoveChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (am *authorizationMiddleware) RemoveAllChildrenGroups(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpRemoveAllChildrenGroups, smqauthz.PolicyReq{
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

func (am *authorizationMiddleware) ListChildrenGroups(ctx context.Context, session authn.Session, id string, startLevel, endLevel int64, pm groups.PageMeta) (groups.Page, error) {
	if err := am.authorize(ctx, session, policies.GroupType, operations.OpListChildrenGroups, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      id,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Page{}, errors.Wrap(errListChildrenGroups, err)
	}

	return am.svc.ListChildrenGroups(ctx, session, id, startLevel, endLevel, pm)
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

func (am *authorizationMiddleware) authorize(ctx context.Context, session authn.Session, entityType string, op permissions.Operation, pr smqauthz.PolicyReq) error {
	pr.Domain = session.DomainID

	perm, err := am.entitiesOps.GetPermission(entityType, op)
	if err != nil {
		return err
	}
	pr.Permission = perm.String()

	var pat *smqauthz.PATReq
	if session.PatID != "" {
		entityID := pr.Object
		opName := am.entitiesOps.OperationName(entityType, op)
		if op == dOperations.OpListDomainGroups || op == dOperations.OpCreateDomainGroups {
			entityID = auth.AnyIDs
		}
		pat = &smqauthz.PATReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityID:   entityID,
			EntityType: auth.GroupsType.String(),
			Operation:  opName,
			Domain:     session.DomainID,
		}
	}

	if err := am.authz.Authorize(ctx, pr, pat); err != nil {
		return err
	}
	return nil
}
