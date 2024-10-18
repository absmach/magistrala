package channels

import (
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
)

const (
	updatePermission           = "update_permission"
	readPermission             = "read_permission"
	deletePermission           = "delete_permission"
	setParentGroupPermission   = "set_parent_group_permission"
	connectToChannelPermission = "connect_to_channel_permission"

	manageRolePermission      = "manage_role_permission"
	addRoleUsersPermission    = "add_role_users_permission"
	removeRoleUsersPermission = "remove_role_users_permission"
	viewRoleUsersPermission   = "view_role_users_permission"
)

const (
	OpCreateChannel svcutil.Operation = iota
	OpListChannel
	OpViewChannel
	OpUpdateChannel
	OpUpdateChannelTags
	OpEnableChannel
	OpDisableChannel
	OpDeleteChannel
	OpConnectThingChannel
	OpDisconnectThingChannel
)

var expectedOperations = []svcutil.Operation{
	OpCreateChannel,
	OpListChannel,
	OpViewChannel,
	OpUpdateChannel,
	OpUpdateChannelTags,
	OpEnableChannel,
	OpDisableChannel,
	OpDeleteChannel,
	OpConnectThingChannel,
	OpDisconnectThingChannel,
}

var operationNames = []string{
	"OpCreateChannel",
	"OpListChannel",
	"OpViewChannel",
	"OpUpdateChannel",
	"OpUpdateChannelTags",
	"OpEnableChannel",
	"OpDisableChannel",
	"OpDeleteChannel",
	"OpConnectThingChannel",
	"OpDisconnectThingChannel",
}

func NewOperationPerm() svcutil.OperationPerm {
	return svcutil.NewOperationPerm(expectedOperations, operationNames)
}

func NewOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		OpCreateChannel:          "",
		OpListChannel:            "",
		OpViewChannel:            "",
		OpUpdateChannel:          "",
		OpUpdateChannelTags:      "",
		OpEnableChannel:          "",
		OpDisableChannel:         "",
		OpDeleteChannel:          "",
		OpConnectThingChannel:    "",
		OpDisconnectThingChannel: "",
	}
	return opPerm
}

func NewRolesOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		roles.OpAddRole:                manageRolePermission,
		roles.OpRemoveRole:             manageRolePermission,
		roles.OpUpdateRoleName:         manageRolePermission,
		roles.OpRetrieveRole:           manageRolePermission,
		roles.OpRetrieveAllRoles:       manageRolePermission,
		roles.OpRoleAddActions:         manageRolePermission,
		roles.OpRoleListActions:        manageRolePermission,
		roles.OpRoleCheckActionsExists: manageRolePermission,
		roles.OpRoleRemoveActions:      manageRolePermission,
		roles.OpRoleRemoveAllActions:   manageRolePermission,
		roles.OpRoleAddMembers:         addRoleUsersPermission,
		roles.OpRoleListMembers:        viewRoleUsersPermission,
		roles.OpRoleCheckMembersExists: viewRoleUsersPermission,
		roles.OpRoleRemoveMembers:      removeRoleUsersPermission,
		roles.OpRoleRemoveAllMembers:   manageRolePermission,
	}
	return opPerm
}
