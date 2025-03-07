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
}

func (are addRoleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
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
	}
	return val, nil
}

type removeRoleEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
}

func (rre removeRoleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rre.operationPrefix + RemoveRole,
		"entity_id": rre.entityID,
		"role_id":   rre.roleID,
	}
	return val, nil
}

type updateRoleEvent struct {
	operationPrefix string
	roles.Role
}

func (ure updateRoleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  ure.operationPrefix + UpdateRole,
		"id":         ure.ID,
		"name":       ure.Name,
		"entity_id":  ure.EntityID,
		"created_by": ure.CreatedBy,
		"created_at": ure.CreatedAt,
		"updated_by": ure.UpdatedBy,
		"updated_at": ure.UpdatedAt,
	}
	return val, nil
}

type retrieveRoleEvent struct {
	operationPrefix string
	roles.Role
}

func (rre retrieveRoleEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":  rre.operationPrefix + ViewRole,
		"id":         rre.ID,
		"name":       rre.Name,
		"entity_id":  rre.EntityID,
		"created_by": rre.CreatedBy,
		"created_at": rre.CreatedAt,
		"updated_by": rre.UpdatedBy,
		"updated_at": rre.UpdatedAt,
	}
	return val, nil
}

type retrieveAllRolesEvent struct {
	operationPrefix string
	entityID        string
	limit           uint64
	offset          uint64
}

func (rare retrieveAllRolesEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rare.operationPrefix + ViewAllRole,
		"entity_id": rare.entityID,
		"limit":     rare.limit,
		"offset":    rare.offset,
	}
	return val, nil
}

type listAvailableActionsEvent struct {
	operationPrefix string
}

func (laae listAvailableActionsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": laae.operationPrefix + ListAvailableActions,
	}
	return val, nil
}

type roleAddActionsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	actions         []string
}

func (raae roleAddActionsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": raae.operationPrefix + AddRoleActions,
		"entity_id": raae.entityID,
		"role_id":   raae.roleID,
		"actions":   raae.actions,
	}
	return val, nil
}

type roleListActionsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
}

func (rlae roleListActionsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rlae.operationPrefix + ListRoleActions,
		"entity_id": rlae.entityID,
		"role_id":   rlae.roleID,
	}
	return val, nil
}

type roleCheckActionsExistsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	actions         []string
	isAllExists     bool
}

func (rcaee roleCheckActionsExistsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation":     rcaee.operationPrefix + CheckRoleActions,
		"entity_id":     rcaee.entityID,
		"role_id":       rcaee.roleID,
		"actions":       rcaee.actions,
		"is_all_exists": rcaee.isAllExists,
	}
	return val, nil
}

type roleRemoveActionsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	actions         []string
}

func (rrae roleRemoveActionsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rrae.operationPrefix + RemoveRoleActions,
		"entity_id": rrae.entityID,
		"role_id":   rrae.roleID,
		"actions":   rrae.actions,
	}
	return val, nil
}

type roleRemoveAllActionsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
}

func (rraae roleRemoveAllActionsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rraae.operationPrefix + RemoveAllRoleActions,
		"entity_id": rraae.entityID,
		"role_id":   rraae.roleID,
	}
	return val, nil
}

type roleAddMembersEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	members         []string
}

func (rame roleAddMembersEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rame.operationPrefix + AddRoleMembers,
		"entity_id": rame.entityID,
		"role_id":   rame.roleID,
		"members":   rame.members,
	}
	return val, nil
}

type roleListMembersEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	limit           uint64
	offset          uint64
}

func (rlme roleListMembersEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rlme.operationPrefix + ListRoleMembers,
		"entity_id": rlme.entityID,
		"role_id":   rlme.roleID,
		"limit":     rlme.limit,
		"offset":    rlme.offset,
	}
	return val, nil
}

type roleCheckMembersExistsEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	members         []string
}

func (rcmee roleCheckMembersExistsEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rcmee.operationPrefix + CheckRoleMembers,
		"entity_id": rcmee.entityID,
		"role_id":   rcmee.roleID,
		"members":   rcmee.members,
	}
	return val, nil
}

type roleRemoveMembersEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
	members         []string
}

func (rrme roleRemoveMembersEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rrme.operationPrefix + RemoveRoleMembers,
		"entity_id": rrme.entityID,
		"role_id":   rrme.roleID,
		"members":   rrme.members,
	}
	return val, nil
}

type roleRemoveAllMembersEvent struct {
	operationPrefix string
	entityID        string
	roleID          string
}

func (rrame roleRemoveAllMembersEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rrame.operationPrefix + RemoveRoleAllMembers,
		"entity_id": rrame.entityID,
		"role_id":   rrame.roleID,
	}
	return val, nil
}

type listEntityMembersEvent struct {
	operationPrefix string
	entityID        string
	limit           uint64
	offset          uint64
}

func (leme listEntityMembersEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": leme.operationPrefix + ListEntityMembers,
		"entity_id": leme.entityID,
		"limit":     leme.limit,
		"offset":    leme.offset,
	}
	return val, nil
}

type removeEntityMembersEvent struct {
	operationPrefix string
	entityID        string
	members         []string
}

func (reme removeEntityMembersEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": reme.operationPrefix + RemoveEntityMembers,
		"entity_id": reme.entityID,
		"members":   reme.members,
	}
	return val, nil
}

type removeMemberFromAllRolesEvent struct {
	operationPrefix string
	memberID        string
}

func (rmare removeMemberFromAllRolesEvent) Encode() (map[string]interface{}, error) {
	val := map[string]interface{}{
		"operation": rmare.operationPrefix + RemoveMemberFromAllRoles,
		"member_id": rmare.memberID,
	}
	return val, nil
}
