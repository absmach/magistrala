// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package channels

import "github.com/absmach/magistrala/pkg/roles"

const (
	ChannelUpdate           roles.Action = "update"
	ChannelRead             roles.Action = "read"
	ChannelDelete           roles.Action = "delete"
	ChannelSetParentGroup   roles.Action = "set_parent_group"
	ChannelConnectToChannel roles.Action = "connect_to_thing"
	ChannelManageRole       roles.Action = "manage_role"
	ChannelAddRoleUsers     roles.Action = "add_role_users"
	ChannelRemoveRoleUsers  roles.Action = "remove_role_users"
	ChannelViewRoleUsers    roles.Action = "view_role_users"
)

const (
	BuiltInRoleAdmin = "admin"
)

func AvailableActions() []roles.Action {
	return []roles.Action{
		ChannelUpdate,
		ChannelRead,
		ChannelDelete,
		ChannelSetParentGroup,
		ChannelConnectToChannel,
		ChannelManageRole,
		ChannelAddRoleUsers,
		ChannelRemoveRoleUsers,
		ChannelViewRoleUsers,
	}
}

func BuiltInRoles() map[roles.BuiltInRoleName][]roles.Action {
	return map[roles.BuiltInRoleName][]roles.Action{
		BuiltInRoleAdmin: AvailableActions(),
	}
}
