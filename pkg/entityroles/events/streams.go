// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/roles"
)

var _ roles.Roles = (*RolesSvcEventStoreMiddleware)(nil)

type RolesSvcEventStoreMiddleware struct {
	events.Publisher
	svc     roles.Roles
	svcName string
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store.
func NewRolesSvcEventStoreMiddleware(svcName string, svc roles.Roles, publisher events.Publisher) RolesSvcEventStoreMiddleware {
	return RolesSvcEventStoreMiddleware{
		svcName:   svcName,
		svc:       svc,
		Publisher: publisher,
	}
}

func (res *RolesSvcEventStoreMiddleware) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (roles.Role, error) {
	return res.svc.AddRole(ctx, session, entityID, roleName, optionalActions, optionalMembers)
}
func (res *RolesSvcEventStoreMiddleware) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return res.svc.RemoveRole(ctx, session, entityID, roleName)
}
func (res *RolesSvcEventStoreMiddleware) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (roles.Role, error) {
	return res.svc.UpdateRoleName(ctx, session, entityID, oldRoleName, newRoleName)
}
func (res *RolesSvcEventStoreMiddleware) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (roles.Role, error) {
	return res.svc.RetrieveRole(ctx, session, entityID, roleName)
}
func (res *RolesSvcEventStoreMiddleware) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (roles.RolePage, error) {
	return res.svc.RetrieveAllRoles(ctx, session, entityID, limit, offset)
}
func (res *RolesSvcEventStoreMiddleware) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	return res.svc.ListAvailableActions(ctx, session)
}
func (res *RolesSvcEventStoreMiddleware) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (ops []string, err error) {
	return res.svc.RoleAddActions(ctx, session, entityID, roleName, actions)
}
func (res *RolesSvcEventStoreMiddleware) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error) {
	return res.svc.RoleListActions(ctx, session, entityID, roleName)
}
func (res *RolesSvcEventStoreMiddleware) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	return res.svc.RoleCheckActionsExists(ctx, session, entityID, roleName, actions)
}
func (res *RolesSvcEventStoreMiddleware) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	return res.svc.RoleRemoveActions(ctx, session, entityID, roleName, actions)
}
func (res *RolesSvcEventStoreMiddleware) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return res.svc.RoleRemoveAllActions(ctx, session, entityID, roleName)
}
func (res *RolesSvcEventStoreMiddleware) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error) {
	return res.svc.RoleAddMembers(ctx, session, entityID, roleName, members)
}
func (res *RolesSvcEventStoreMiddleware) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (roles.MembersPage, error) {
	return res.svc.RoleListMembers(ctx, session, entityID, roleName, limit, offset)
}
func (res *RolesSvcEventStoreMiddleware) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	return res.svc.RoleCheckMembersExists(ctx, session, entityID, roleName, members)
}
func (res *RolesSvcEventStoreMiddleware) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	return res.svc.RoleRemoveMembers(ctx, session, entityID, roleName, members)
}
func (res *RolesSvcEventStoreMiddleware) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	return res.svc.RoleRemoveAllMembers(ctx, session, entityID, roleName)
}
func (res *RolesSvcEventStoreMiddleware) RemoveMembersFromAllRoles(ctx context.Context, session authn.Session, members []string) (err error) {
	return res.svc.RemoveMembersFromAllRoles(ctx, session, members)
}
func (res *RolesSvcEventStoreMiddleware) RemoveMembersFromRoles(ctx context.Context, session authn.Session, members []string, roleNames []string) (err error) {
	return res.svc.RemoveMembersFromRoles(ctx, session, members, roleNames)
}
func (res *RolesSvcEventStoreMiddleware) RemoveActionsFromAllRoles(ctx context.Context, session authn.Session, actions []string) (err error) {
	return res.svc.RemoveActionsFromAllRoles(ctx, session, actions)
}
func (res *RolesSvcEventStoreMiddleware) RemoveActionsFromRoles(ctx context.Context, session authn.Session, actions []string, roleNames []string) (err error) {
	return res.svc.RemoveActionsFromRoles(ctx, session, actions, roleNames)
}
