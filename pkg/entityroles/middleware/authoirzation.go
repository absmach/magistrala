// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
)

var _ roles.Roles = (*RolesAuthorizationMiddleware)(nil)

type RolesAuthorizationMiddleware struct {
	entityType string
	svc        roles.Roles
	authz      mgauthz.Authorization
	opp        svcutil.OperationPerm
}

// AuthorizationMiddleware adds authorization to the clients service.
func NewRolesAuthorizationMiddleware(entityType string, svc roles.Roles, authz mgauthz.Authorization, opPerm map[svcutil.Operation]svcutil.Permission) (RolesAuthorizationMiddleware, error) {
	opp := roles.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(opPerm); err != nil {
		return RolesAuthorizationMiddleware{}, err
	}
	if err := opp.Validate(); err != nil {
		return RolesAuthorizationMiddleware{}, err
	}

	ram := RolesAuthorizationMiddleware{
		entityType: entityType,
		svc:        svc,
		authz:      authz,
		opp:        opp,
	}
	if err := ram.validate(); err != nil {
		return RolesAuthorizationMiddleware{}, err
	}
	return ram, nil
}

func (ram RolesAuthorizationMiddleware) validate() error {
	if err := ram.opp.Validate(); err != nil {
		return err
	}
	return nil
}

func (ram RolesAuthorizationMiddleware) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (roles.Role, error) {
	if err := ram.authorize(ctx, roles.OpAddRole, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return roles.Role{}, err
	}
	return ram.svc.AddRole(ctx, session, entityID, roleName, optionalActions, optionalMembers)
}
func (ram RolesAuthorizationMiddleware) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error {
	if err := ram.authorize(ctx, roles.OpRemoveRole, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return err
	}
	return ram.svc.RemoveRole(ctx, session, entityID, roleName)
}
func (ram RolesAuthorizationMiddleware) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (roles.Role, error) {
	if err := ram.authorize(ctx, roles.OpUpdateRoleName, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return roles.Role{}, err
	}
	return ram.svc.UpdateRoleName(ctx, session, entityID, oldRoleName, newRoleName)
}
func (ram RolesAuthorizationMiddleware) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (roles.Role, error) {
	if err := ram.authorize(ctx, roles.OpRetrieveRole, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return roles.Role{}, err
	}
	return ram.svc.RetrieveRole(ctx, session, entityID, roleName)
}
func (ram RolesAuthorizationMiddleware) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (roles.RolePage, error) {
	if err := ram.authorize(ctx, roles.OpRetrieveAllRoles, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return roles.RolePage{}, err
	}
	return ram.svc.RetrieveAllRoles(ctx, session, entityID, limit, offset)
}
func (ram RolesAuthorizationMiddleware) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	return ram.svc.ListAvailableActions(ctx, session)
}
func (ram RolesAuthorizationMiddleware) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (ops []string, err error) {
	if err := ram.authorize(ctx, roles.OpRoleAddActions, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return []string{}, err
	}

	return ram.svc.RoleAddActions(ctx, session, entityID, roleName, actions)
}
func (ram RolesAuthorizationMiddleware) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error) {
	if err := ram.authorize(ctx, roles.OpRoleListActions, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return []string{}, err
	}

	return ram.svc.RoleListActions(ctx, session, entityID, roleName)
}
func (ram RolesAuthorizationMiddleware) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	if err := ram.authorize(ctx, roles.OpRoleCheckActionsExists, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return false, err
	}
	return ram.svc.RoleCheckActionsExists(ctx, session, entityID, roleName, actions)
}
func (ram RolesAuthorizationMiddleware) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	if err := ram.authorize(ctx, roles.OpRoleRemoveActions, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return err
	}
	return ram.svc.RoleRemoveActions(ctx, session, entityID, roleName, actions)
}
func (ram RolesAuthorizationMiddleware) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error {
	if err := ram.authorize(ctx, roles.OpRoleRemoveAllActions, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return err
	}
	return ram.svc.RoleRemoveAllActions(ctx, session, entityID, roleName)
}
func (ram RolesAuthorizationMiddleware) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error) {
	if err := ram.authorize(ctx, roles.OpRoleAddMembers, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return []string{}, err
	}
	return ram.svc.RoleAddMembers(ctx, session, entityID, roleName, members)
}
func (ram RolesAuthorizationMiddleware) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (roles.MembersPage, error) {
	if err := ram.authorize(ctx, roles.OpRoleListMembers, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return roles.MembersPage{}, err
	}
	return ram.svc.RoleListMembers(ctx, session, entityID, roleName, limit, offset)
}
func (ram RolesAuthorizationMiddleware) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	if err := ram.authorize(ctx, roles.OpRoleCheckMembersExists, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return false, err
	}
	return ram.svc.RoleCheckMembersExists(ctx, session, entityID, roleName, members)
}
func (ram RolesAuthorizationMiddleware) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	if err := ram.authorize(ctx, roles.OpRoleRemoveMembers, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return err
	}
	return ram.svc.RoleRemoveMembers(ctx, session, entityID, roleName, members)
}
func (ram RolesAuthorizationMiddleware) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	if err := ram.authorize(ctx, roles.OpRoleRemoveAllMembers, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		Subject:     session.DomainUserID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Object:      entityID,
		ObjectType:  ram.entityType,
	}); err != nil {
		return err
	}
	return ram.svc.RoleRemoveAllMembers(ctx, session, entityID, roleName)
}

func (ram RolesAuthorizationMiddleware) RemoveMembersFromAllRoles(ctx context.Context, session authn.Session, members []string) (err error) {
	return ram.svc.RemoveMembersFromAllRoles(ctx, session, members)
}
func (ram RolesAuthorizationMiddleware) RemoveMembersFromRoles(ctx context.Context, session authn.Session, members []string, roleNames []string) (err error) {
	return ram.svc.RemoveMembersFromRoles(ctx, session, members, roleNames)
}
func (ram RolesAuthorizationMiddleware) RemoveActionsFromAllRoles(ctx context.Context, session authn.Session, actions []string) (err error) {
	return ram.svc.RemoveActionsFromAllRoles(ctx, session, actions)
}
func (ram RolesAuthorizationMiddleware) RemoveActionsFromRoles(ctx context.Context, session authn.Session, actions []string, roleNames []string) (err error) {
	return ram.svc.RemoveActionsFromRoles(ctx, session, actions, roleNames)
}

func (ram RolesAuthorizationMiddleware) authorize(ctx context.Context, op svcutil.Operation, pr mgauthz.PolicyReq) error {
	perm, err := ram.opp.GetPermission(op)
	if err != nil {
		return err
	}

	pr.Permission = perm.String()

	if err := ram.authz.Authorize(ctx, pr); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}
