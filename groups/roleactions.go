package groups

import "github.com/absmach/magistrala/pkg/roles"

const (
	Update     = "update"
	Read       = "read"
	Membership = "membership"
	Delete     = "delete"
	SetChild   = "set_child"
	SetParent  = "set_parent"

	ManageRole      = "manage_role"
	AddRoleUsers    = "add_role_users"
	RemoveRoleUsers = "remove_role_users"
	ViewRoleUsers   = "view_role_users"

	ClientCreate          = "client_create"
	ChannelCreate         = "channel_create"
	SubgroupCreate        = "subgroup_create"
	SubgroupClientCreate   = "subgroup_client_create"
	SubgroupChannelCreate = "subgroup_channel_create"

	ClientUpdate           = "client_update"
	ClientRead             = "client_read"
	ClientDelete           = "client_delete"
	ClientSetParentGroup   = "client_set_parent_group"
	ClientConnectToChannel = "client_connect_to_channel"

	ClientManageRole      = "client_manage_role"
	ClientAddRoleUsers    = "client_add_role_users"
	ClientRemoveRoleUsers = "client_remove_role_users"
	ClientViewRoleUsers   = "client_view_role_users"

	ChannelUpdate         = "channel_update"
	ChannelRead           = "channel_read"
	ChannelDelete         = "channel_delete"
	ChannelSetParentGroup = "channel_set_parent_group"
	ChannelConnectToClient = "channel_connect_to_client"
	ChannelPublish        = "channel_publish"
	ChannelSubscribe      = "channel_subscribe"

	ChannelManageRole      = "channel_manage_role"
	ChannelAddRoleUsers    = "channel_add_role_users"
	ChannelRemoveRoleUsers = "channel_remove_role_users"
	ChannelViewRoleUsers   = "channel_view_role_users"

	SubgroupUpdate     = "subgroup_update"
	SubgroupRead       = "subgroup_read"
	SubgroupMembership = "subgroup_membership"
	SubgroupDelete     = "subgroup_delete"
	SubgroupSetChild   = "subgroup_set_child"
	SubgroupSetParent  = "subgroup_set_parent"

	SubgroupManageRole      = "subgroup_manage_role"
	SubgroupAddRoleUsers    = "subgroup_add_role_users"
	SubgroupRemoveRoleUsers = "subgroup_remove_role_users"
	SubgroupViewRoleUsers   = "subgroup_view_role_users"

	SubgroupClientUpdate           = "subgroup_client_update"
	SubgroupClientRead             = "subgroup_client_read"
	SubgroupClientDelete           = "subgroup_client_delete"
	SubgroupClientSetParentGroup   = "subgroup_client_set_parent_group"
	SubgroupClientConnectToChannel = "subgroup_client_connect_to_channel"

	SubgroupClientManageRole      = "subgroup_client_manage_role"
	SubgroupClientAddRoleUsers    = "subgroup_client_add_role_users"
	SubgroupClientRemoveRoleUsers = "subgroup_client_remove_role_users"
	SubgroupClientViewRoleUsers   = "subgroup_client_view_role_users"

	SubgroupChannelUpdate         = "subgroup_channel_update"
	SubgroupChannelRead           = "subgroup_channel_read"
	SubgroupChannelDelete         = "subgroup_channel_delete"
	SubgroupChannelSetParentGroup = "subgroup_channel_set_parent_group"
	SubgroupChannelConnectToClient = "subgroup_channel_connect_to_client"
	SubgroupChannelPublish        = "subgroup_channel_publish"
	SubgroupChannelSubscribe      = "subgroup_channel_subscribe"

	SubgroupChannelManageRole      = "subgroup_channel_manage_role"
	SubgroupChannelAddRoleUsers    = "subgroup_channel_add_role_users"
	SubgroupChannelRemoveRoleUsers = "subgroup_channel_remove_role_users"
	SubgroupChannelViewRoleUsers   = "subgroup_channel_view_role_users"
)

const (
	BuiltInRoleAdmin      = "admin"
	BuiltInRoleMembership = "membership"
)

func AvailableActions() []roles.Action {
	return []roles.Action{
		Update,
		Read,
		Membership,
		Delete,
		SetChild,
		SetParent,
		ManageRole,
		AddRoleUsers,
		RemoveRoleUsers,
		ViewRoleUsers,
		ClientCreate,
		ChannelCreate,
		SubgroupCreate,
		SubgroupClientCreate,
		SubgroupChannelCreate,
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
		SubgroupUpdate,
		SubgroupRead,
		SubgroupMembership,
		SubgroupDelete,
		SubgroupSetChild,
		SubgroupSetParent,
		SubgroupManageRole,
		SubgroupAddRoleUsers,
		SubgroupRemoveRoleUsers,
		SubgroupViewRoleUsers,
		SubgroupClientUpdate,
		SubgroupClientRead,
		SubgroupClientDelete,
		SubgroupClientSetParentGroup,
		SubgroupClientConnectToChannel,
		SubgroupClientManageRole,
		SubgroupClientAddRoleUsers,
		SubgroupClientRemoveRoleUsers,
		SubgroupClientViewRoleUsers,
		SubgroupChannelUpdate,
		SubgroupChannelRead,
		SubgroupChannelDelete,
		SubgroupChannelSetParentGroup,
		SubgroupChannelConnectToClient,
		SubgroupChannelPublish,
		SubgroupChannelSubscribe,
		SubgroupChannelManageRole,
		SubgroupChannelAddRoleUsers,
		SubgroupChannelRemoveRoleUsers,
		SubgroupChannelViewRoleUsers,
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
