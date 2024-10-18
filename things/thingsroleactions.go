// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package things

import "github.com/absmach/magistrala/pkg/roles"

const (
	ThingUpdate           roles.Action = "update"
	ThingRead             roles.Action = "read"
	ThingDelete           roles.Action = "delete"
	ThingSetParentGroup   roles.Action = "set_parent_group"
	ThingConnectToChannel roles.Action = "connect_to_thing"
	ThingManageRole       roles.Action = "manage_role"
	ThingAddRoleUsers     roles.Action = "add_role_users"
	ThingRemoveRoleUsers  roles.Action = "remove_role_users"
	ThingViewRoleUsers    roles.Action = "view_role_users"
)

const (
	ThingBuiltInRoleAdmin = "admin"
)

func ThingAvailableActions() []roles.Action {
	return []roles.Action{
		ThingUpdate,
		ThingRead,
		ThingDelete,
		ThingSetParentGroup,
		ThingConnectToChannel,
		ThingManageRole,
		ThingAddRoleUsers,
		ThingRemoveRoleUsers,
		ThingViewRoleUsers,
	}
}

func ThingBuiltInRoles() map[roles.BuiltInRoleName][]roles.Action {
	return map[roles.BuiltInRoleName][]roles.Action{
		ThingBuiltInRoleAdmin: ThingAvailableActions(),
	}
}
