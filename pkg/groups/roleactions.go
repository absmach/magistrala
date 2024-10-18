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

	ThingCreate           = "thing_create"
	ChannelCreate         = "channel_create"
	SubgroupCreate        = "subgroup_create"
	SubgroupThingCreate   = "subgroup_thing_create"
	SubgroupChannelCreate = "subgroup_channel_create"

	ThingUpdate           = "thing_update"
	ThingRead             = "thing_read"
	ThingDelete           = "thing_delete"
	ThingSetParentGroup   = "thing_set_parent_group"
	ThingConnectToChannel = "thing_connect_to_channel"

	ThingManageRole      = "thing_manage_role"
	ThingAddRoleUsers    = "thing_add_role_users"
	ThingRemoveRoleUsers = "thing_remove_role_users"
	ThingViewRoleUsers   = "thing_view_role_users"

	ChannelUpdate         = "channel_update"
	ChannelRead           = "channel_read"
	ChannelDelete         = "channel_delete"
	ChannelSetParentGroup = "channel_set_parent_group"
	ChannelConnectToThing = "channel_connect_to_thing"
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

	SubgroupThingUpdate           = "subgroup_thing_update"
	SubgroupThingRead             = "subgroup_thing_read"
	SubgroupThingDelete           = "subgroup_thing_delete"
	SubgroupThingSetParentGroup   = "subgroup_thing_set_parent_group"
	SubgroupThingConnectToChannel = "subgroup_thing_connect_to_channel"

	SubgroupThingManageRole      = "subgroup_thing_manage_role"
	SubgroupThingAddRoleUsers    = "subgroup_thing_add_role_users"
	SubgroupThingRemoveRoleUsers = "subgroup_thing_remove_role_users"
	SubgroupThingViewRoleUsers   = "subgroup_thing_view_role_users"

	SubgroupChannelUpdate         = "subgroup_channel_update"
	SubgroupChannelRead           = "subgroup_channel_read"
	SubgroupChannelDelete         = "subgroup_channel_delete"
	SubgroupChannelSetParentGroup = "subgroup_channel_set_parent_group"
	SubgroupChannelConnectToThing = "subgroup_channel_connect_to_thing"
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
		ThingCreate,
		ChannelCreate,
		SubgroupCreate,
		SubgroupThingCreate,
		SubgroupChannelCreate,
		ThingUpdate,
		ThingRead,
		ThingDelete,
		ThingSetParentGroup,
		ThingConnectToChannel,
		ThingManageRole,
		ThingAddRoleUsers,
		ThingRemoveRoleUsers,
		ThingViewRoleUsers,
		ChannelUpdate,
		ChannelRead,
		ChannelDelete,
		ChannelSetParentGroup,
		ChannelConnectToThing,
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
		SubgroupThingUpdate,
		SubgroupThingRead,
		SubgroupThingDelete,
		SubgroupThingSetParentGroup,
		SubgroupThingConnectToChannel,
		SubgroupThingManageRole,
		SubgroupThingAddRoleUsers,
		SubgroupThingRemoveRoleUsers,
		SubgroupThingViewRoleUsers,
		SubgroupChannelUpdate,
		SubgroupChannelRead,
		SubgroupChannelDelete,
		SubgroupChannelSetParentGroup,
		SubgroupChannelConnectToThing,
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
