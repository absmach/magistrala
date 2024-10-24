// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import "github.com/absmach/magistrala/pkg/roles"

// Below codes should moved out of service, may be can be kept in `cmd/<svc>/main.go`

const (
	ClientUpdate           roles.Action = "update"
	ClientRead             roles.Action = "read"
	ClientDelete           roles.Action = "delete"
	ClientSetParentGroup   roles.Action = "set_parent_group"
	ClientConnectToChannel roles.Action = "connect_to_channel"
	ClientManageRole       roles.Action = "manage_role"
	ClientAddRoleUsers     roles.Action = "add_role_users"
	ClientRemoveRoleUsers  roles.Action = "remove_role_users"
	ClientViewRoleUsers    roles.Action = "view_role_users"
)

const (
	ClientBuiltInRoleAdmin = "admin"
)

func AvailableActions() []roles.Action {
	return []roles.Action{
		ClientUpdate,
		ClientRead,
		ClientDelete,
		ClientSetParentGroup,
		ClientConnectToChannel,
		ClientManageRole,
		ClientAddRoleUsers,
		ClientRemoveRoleUsers,
		ClientViewRoleUsers,
	}
}

func BuiltInRoles() map[roles.BuiltInRoleName][]roles.Action {
	return map[roles.BuiltInRoleName][]roles.Action{
		ClientBuiltInRoleAdmin: AvailableActions(),
	}
}
