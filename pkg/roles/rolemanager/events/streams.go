// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/roles"
)

var _ roles.RoleManager = (*RoleManagerEventStore)(nil)

type RoleManagerEventStore struct {
	events.Publisher
	svc     roles.RoleManager
	svcName string
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store.
func NewRoleManagerEventStore(svcName string, svc roles.RoleManager, publisher events.Publisher) RoleManagerEventStore {
	return RoleManagerEventStore{
		svcName:   svcName,
		svc:       svc,
		Publisher: publisher,
	}
}

func (res *RoleManagerEventStore) AddRole(ctx context.Context, session authn.Session, entityID, roleName string, optionalActions []string, optionalMembers []string) (roles.Role, error) {
	return res.svc.AddRole(ctx, session, entityID, roleName, optionalActions, optionalMembers)
}
func (res *RoleManagerEventStore) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return res.svc.RemoveRole(ctx, session, entityID, roleName)
}
func (res *RoleManagerEventStore) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (roles.Role, error) {
	return res.svc.UpdateRoleName(ctx, session, entityID, oldRoleName, newRoleName)
}
func (res *RoleManagerEventStore) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (roles.Role, error) {
	return res.svc.RetrieveRole(ctx, session, entityID, roleName)
}
func (res *RoleManagerEventStore) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (roles.RolePage, error) {
	return res.svc.RetrieveAllRoles(ctx, session, entityID, limit, offset)
}
func (res *RoleManagerEventStore) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	return res.svc.ListAvailableActions(ctx, session)
}
func (res *RoleManagerEventStore) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (ops []string, err error) {
	return res.svc.RoleAddActions(ctx, session, entityID, roleName, actions)
}
func (res *RoleManagerEventStore) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error) {
	return res.svc.RoleListActions(ctx, session, entityID, roleName)
}
func (res *RoleManagerEventStore) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	return res.svc.RoleCheckActionsExists(ctx, session, entityID, roleName, actions)
}
func (res *RoleManagerEventStore) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	return res.svc.RoleRemoveActions(ctx, session, entityID, roleName, actions)
}
func (res *RoleManagerEventStore) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error {
	return res.svc.RoleRemoveAllActions(ctx, session, entityID, roleName)
}
func (res *RoleManagerEventStore) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) ([]string, error) {
	return res.svc.RoleAddMembers(ctx, session, entityID, roleName, members)
}
func (res *RoleManagerEventStore) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (roles.MembersPage, error) {
	return res.svc.RoleListMembers(ctx, session, entityID, roleName, limit, offset)
}
func (res *RoleManagerEventStore) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	return res.svc.RoleCheckMembersExists(ctx, session, entityID, roleName, members)
}
func (res *RoleManagerEventStore) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	return res.svc.RoleRemoveMembers(ctx, session, entityID, roleName, members)
}
func (res *RoleManagerEventStore) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	return res.svc.RoleRemoveAllMembers(ctx, session, entityID, roleName)
}
func (res *RoleManagerEventStore) RemoveMemberFromAllRoles(ctx context.Context, session authn.Session, membersID string) (err error) {
	return res.svc.RemoveMemberFromAllRoles(ctx, session, membersID)
}
