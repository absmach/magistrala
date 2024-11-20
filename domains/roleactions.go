// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import "github.com/absmach/magistrala/pkg/roles"

const (
	// Domain Roles : Actions related to manage the domain.
	Update          roles.Action = "update"
	Enable          roles.Action = "enable"
	Disable         roles.Action = "disable"
	Read            roles.Action = "read"
	Delete          roles.Action = "delete"
	Membership      roles.Action = "membership"
	ManageRole      roles.Action = "manage_role"
	AddRoleUsers    roles.Action = "add_role_users"
	RemoveRoleUsers roles.Action = "remove_role_users"
	ViewRoleUsers   roles.Action = "view_role_users"

	// Domain Roles : Actions related to entity creation and entity listing within domain
	ClientCreate  roles.Action = "client_create"
	ChannelCreate roles.Action = "channel_create"
	GroupCreate   roles.Action = "group_create"

	// Domain Clients Roles: Actions related to clients present within the Domain
	ClientUpdate           roles.Action = "client_update"
	ClientRead             roles.Action = "client_read"
	ClientDelete           roles.Action = "client_delete"
	ClientSetParentGroup   roles.Action = "client_set_parent_group"
	ClientConnectToChannel roles.Action = "client_connect_to_channel"
	ClientManageRole       roles.Action = "client_manage_role"
	ClientAddRoleUsers     roles.Action = "client_add_role_users"
	ClientRemoveRoleUsers  roles.Action = "client_remove_role_users"
	ClientViewRoleUsers    roles.Action = "client_view_role_users"

	// Domain Channels Roles: Actions related to channels present within the Domain.
	ChannelUpdate          roles.Action = "channel_update"
	ChannelRead            roles.Action = "channel_read"
	ChannelDelete          roles.Action = "channel_delete"
	ChannelSetParentGroup  roles.Action = "channel_set_parent_group"
	ChannelConnectToClient  roles.Action = "channel_connect_to_client"
	ChannelPublish         roles.Action = "channel_publish"
	ChannelSubscribe       roles.Action = "channel_subscribe"
	ChannelManageRole      roles.Action = "channel_manage_role"
	ChannelAddRoleUsers    roles.Action = "channel_add_role_users"
	ChannelRemoveRoleUsers roles.Action = "channel_remove_role_users"
	ChannelViewRoleUsers   roles.Action = "channel_view_role_users"

	// Domain Groups Roles: Actions related to Groups present within the Domain.
	GroupUpdate          roles.Action = "group_update"
	GroupMembership      roles.Action = "group_membership"
	GroupRead            roles.Action = "group_read"
	GroupDelete          roles.Action = "group_delete"
	GroupSetChild        roles.Action = "group_set_child"
	GroupSetParent       roles.Action = "group_set_parent"
	GroupManageRole      roles.Action = "group_manage_role"
	GroupAddRoleUsers    roles.Action = "group_add_role_users"
	GroupRemoveRoleUsers roles.Action = "group_remove_role_users"
	GroupViewRoleUsers   roles.Action = "group_view_role_users"
)

const (
	BuiltInRoleAdmin      = "admin"
	BuiltInRoleMembership = "membership"
)

func AvailableActions() []roles.Action {
	return []roles.Action{
		Update,
		Enable,
		Disable,
		Read,
		Delete,
		Membership,
		ManageRole,
		AddRoleUsers,
		RemoveRoleUsers,
		ViewRoleUsers,
		ClientCreate,
		ChannelCreate,
		GroupCreate,
		ClientUpdate,
		ClientRead,
		ClientDelete,
		ClientSetParentGroup,
		ClientConnectToChannel,
		ClientManageRole,
		ClientAddRoleUsers,
		ClientRemoveRoleUsers,
		ClientViewRoleUsers,
		ChannelUpdate,
		ChannelRead,
		ChannelDelete,
		ChannelSetParentGroup,
		ChannelConnectToClient,
		ChannelPublish,
		ChannelSubscribe,
		ChannelManageRole,
		ChannelAddRoleUsers,
		ChannelRemoveRoleUsers,
		ChannelViewRoleUsers,
		GroupUpdate,
		GroupMembership,
		GroupRead,
		GroupDelete,
		GroupSetChild,
		GroupSetParent,
		GroupManageRole,
		GroupAddRoleUsers,
		GroupRemoveRoleUsers,
		GroupViewRoleUsers,
	}
}

func membershipRoleActions() []roles.Action {
	return []roles.Action{
		Membership,
	}
}

func BuiltInRoles() map[roles.BuiltInRoleName][]roles.Action {
	return map[roles.BuiltInRoleName][]roles.Action{
		BuiltInRoleAdmin:      AvailableActions(),
		BuiltInRoleMembership: membershipRoleActions(),
	}
}
