// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/roles"
)

const (
	AddRole                  = "role.add"
	RemoveRole               = "role.remove"
	UpdateRole               = "role.update"
	ViewRole                 = "role.view"
	ViewAllRole              = "role.view_all"
	ListAvailableActions     = "role.list_available_actions"
	AddRoleActions           = "role.actions.add"
	ListRoleActions          = "role.actions.ist"
	CheckRoleActions         = "role.actions.check"
	RemoveRoleActions        = "role.actions.remove"
	RemoveAllRoleActions     = "role.actions.remove_all"
	AddRoleMembers           = "role.members.add"
	ListRoleMembers          = "role.members.list"
	CheckRoleMembers         = "role.members.check"
	RemoveRoleMembers        = "role.members.remove"
	RemoveRoleAllMembers     = "role.members.remove_all"
	RemoveMemberFromDomain   = "role.domain.member.remove"
	ListEntityMembers        = "members.list"
	RemoveEntityMembers      = "members.remove"
	RemoveMemberFromAllRoles = "role.members.remove_from_all_roles"
)

var (
	_ events.Event = (*addRoleEvent)(nil)
	_ events.Event = (*removeRoleEvent)(nil)
	_ events.Event = (*updateRoleEvent)(nil)
	_ events.Event = (*retrieveRoleEvent)(nil)
	_ events.Event = (*retrieveAllRolesEvent)(nil)
	_ events.Event = (*listAvailableActionsEvent)(nil)
	_ events.Event = (*roleAddActionsEvent)(nil)
	_ events.Event = (*roleListActionsEvent)(nil)
	_ events.Event = (*roleCheckActionsExistsEvent)(nil)
	_ events.Event = (*roleRemoveActionsEvent)(nil)
	_ events.Event = (*roleRemoveAllActionsEvent)(nil)
	_ events.Event = (*roleAddMembersEvent)(nil)
	_ events.Event = (*roleListMembersEvent)(nil)
	_ events.Event = (*roleCheckMembersExistsEvent)(nil)
	_ events.Event = (*roleRemoveMembersEvent)(nil)
	_ events.Event = (*roleRemoveAllMembersEvent)(nil)
	_ events.Event = (*listEntityMembersEvent)(nil)
	_ events.Event = (*removeEntityMembersEvent)(nil)
	_ events.Event = (*removeMemberFromAllRolesEvent)(nil)
)

type addRoleEvent struct {
	operationPrefix string
	roles.RoleProvision
	requestID string
}

func (are addRoleEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":        are.operationPrefix + AddRole,
		"id":               are.ID,
		"name":             are.Name,
		"entity_id":        are.EntityID,
		"created_by":       are.CreatedBy,
		"created_at":       are.CreatedAt,
		"updated_by":       are.UpdatedBy,
		"updated_at":       are.UpdatedAt,
		"optional_actions": are.OptionalActions,
		"optional_members": are.OptionalMembers,
		"request_id":       are.requestID,
	}
	return val, nil
}

type removeRoleEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	requestID       string
}

func (rre removeRoleEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rre.operationPrefix + RemoveRole,
		"entity_id":  rre.entityID,
		"role_id":    rre.roleID,
		"request_id": rre.requestID,
	}
	return val, nil
}

type updateRoleEvent struct {
	operationPrefix string
	roles.Role
	requestID string
}

func (ure updateRoleEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  ure.operationPrefix + UpdateRole,
		"id":         ure.ID,
		"name":       ure.Name,
		"entity_id":  ure.EntityID,
		"created_by": ure.CreatedBy,
		"created_at": ure.CreatedAt,
		"updated_by": ure.UpdatedBy,
		"updated_at": ure.UpdatedAt,
		"request_id": ure.requestID,
	}
	return val, nil
}

type retrieveRoleEvent struct {
	operationPrefix string
	roles.Role
	requestID string
}

func (rre retrieveRoleEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rre.operationPrefix + ViewRole,
		"id":         rre.ID,
		"name":       rre.Name,
		"entity_id":  rre.EntityID,
		"created_by": rre.CreatedBy,
		"created_at": rre.CreatedAt,
		"updated_by": rre.UpdatedBy,
		"updated_at": rre.UpdatedAt,
		"request_id": rre.requestID,
	}
	return val, nil
}

type retrieveAllRolesEvent struct {
	operationPrefix string
	entityID        string
	limit           uint64
	offset          uint64
	requestID       string
}

func (rare retrieveAllRolesEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rare.operationPrefix + ViewAllRole,
		"entity_id":  rare.entityID,
		"limit":      rare.limit,
		"offset":     rare.offset,
		"request_id": rare.requestID,
	}
	return val, nil
}

type listAvailableActionsEvent struct {
	operationPrefix string
	requestID       string
}

func (laae listAvailableActionsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  laae.operationPrefix + ListAvailableActions,
		"request_id": laae.requestID,
	}
	return val, nil
}

type roleAddActionsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	actions         []string
	requestID       string
}

func (raae roleAddActionsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  raae.operationPrefix + AddRoleActions,
		"entity_id":  raae.entityID,
		"role_id":    raae.roleID,
		"actions":    raae.actions,
		"request_id": raae.requestID,
	}
	return val, nil
}

type roleListActionsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	requestID       string
}

func (rlae roleListActionsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rlae.operationPrefix + ListRoleActions,
		"entity_id":  rlae.entityID,
		"role_id":    rlae.roleID,
		"request_id": rlae.requestID,
	}
	return val, nil
}

type roleCheckActionsExistsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	actions         []string
	isAllExists     bool
	requestID       string
}

func (rcaee roleCheckActionsExistsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":     rcaee.operationPrefix + CheckRoleActions,
		"entity_id":     rcaee.entityID,
		"role_id":       rcaee.roleID,
		"actions":       rcaee.actions,
		"is_all_exists": rcaee.isAllExists,
		"request_id":    rcaee.requestID,
	}
	return val, nil
}

type roleRemoveActionsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	actions         []string
	requestID       string
}

func (rrae roleRemoveActionsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rrae.operationPrefix + RemoveRoleActions,
		"entity_id":  rrae.entityID,
		"role_id":    rrae.roleID,
		"actions":    rrae.actions,
		"request_id": rrae.requestID,
	}
	return val, nil
}

type roleRemoveAllActionsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	requestID       string
}

func (rraae roleRemoveAllActionsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rraae.operationPrefix + RemoveAllRoleActions,
		"entity_id":  rraae.entityID,
		"role_id":    rraae.roleID,
		"request_id": rraae.requestID,
	}
	return val, nil
}

type roleAddMembersEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	members         []string
	requestID       string
}

func (rame roleAddMembersEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rame.operationPrefix + AddRoleMembers,
		"entity_id":  rame.entityID,
		"role_id":    rame.roleID,
		"members":    rame.members,
		"request_id": rame.requestID,
	}
	return val, nil
}

type roleListMembersEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	limit           uint64
	offset          uint64
	requestID       string
}

func (rlme roleListMembersEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rlme.operationPrefix + ListRoleMembers,
		"entity_id":  rlme.entityID,
		"role_id":    rlme.roleID,
		"limit":      rlme.limit,
		"offset":     rlme.offset,
		"request_id": rlme.requestID,
	}
	return val, nil
}

type roleCheckMembersExistsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	members         []string
	requestID       string
}

func (rcmee roleCheckMembersExistsEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rcmee.operationPrefix + CheckRoleMembers,
		"entity_id":  rcmee.entityID,
		"role_id":    rcmee.roleID,
		"members":    rcmee.members,
		"request_id": rcmee.requestID,
	}
	return val, nil
}

type roleRemoveMembersEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	members         []string
	requestID       string
}

func (rrme roleRemoveMembersEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rrme.operationPrefix + RemoveRoleMembers,
		"entity_id":  rrme.entityID,
		"role_id":    rrme.roleID,
		"members":    rrme.members,
		"request_id": rrme.requestID,
	}
	return val, nil
}

type roleRemoveAllMembersEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	requestID       string
}

func (rrame roleRemoveAllMembersEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rrame.operationPrefix + RemoveRoleAllMembers,
		"entity_id":  rrame.entityID,
		"role_id":    rrame.roleID,
		"request_id": rrame.requestID,
	}
	return val, nil
}

type listEntityMembersEvent struct {
	operationPrefix string
	entityID        string
	limit           uint64
	offset          uint64
	requestID       string
}

func (leme listEntityMembersEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  leme.operationPrefix + ListEntityMembers,
		"entity_id":  leme.entityID,
		"limit":      leme.limit,
		"offset":     leme.offset,
		"request_id": leme.requestID,
	}
	return val, nil
}

type removeMemberFromDomainEvent struct {
	operationPrefix string
	domainID        string
	memberID        string
	requestID       string
}

func (rmde removeMemberFromDomainEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rmde.operationPrefix + RemoveMemberFromDomain,
		"domain_id":  rmde.domainID,
		"member_id":  rmde.memberID,
		"request_id": rmde.requestID,
	}
	return val, nil
}

type removeEntityMembersEvent struct {
	operationPrefix string
	entityID        string
	members         []string
	requestID       string
}

func (reme removeEntityMembersEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  reme.operationPrefix + RemoveEntityMembers,
		"entity_id":  reme.entityID,
		"members":    reme.members,
		"request_id": reme.requestID,
	}
	return val, nil
}

type removeMemberFromAllRolesEvent struct {
	operationPrefix string
	memberID        string
	requestID       string
}

func (rmare removeMemberFromAllRolesEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"operation":  rmare.operationPrefix + RemoveMemberFromAllRoles,
		"member_id":  rmare.memberID,
		"request_id": rmare.requestID,
	}
	return val, nil
}
