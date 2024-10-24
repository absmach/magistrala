package channels

import (
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
)

// Internal Operations

const (
	OpViewChannel svcutil.Operation = iota
	OpUpdateChannel
	OpUpdateChannelTags
	OpEnableChannel
	OpDisableChannel
	OpDeleteChannel
	OpSetParentGroup
	OpRemoveParentGroup
	OpConnectClient
	OpDisconnectClient
)

var expectedOperations = []svcutil.Operation{
	OpViewChannel,
	OpUpdateChannel,
	OpUpdateChannelTags,
	OpEnableChannel,
	OpDisableChannel,
	OpDeleteChannel,
	OpSetParentGroup,
	OpRemoveParentGroup,
	OpConnectClient,
	OpDisconnectClient,
}

var operationNames = []string{
	"OpViewChannel",
	"OpUpdateChannel",
	"OpUpdateChannelTags",
	"OpEnableChannel",
	"OpDisableChannel",
	"OpDeleteChannel",
	"OpSetParentGroup",
	"OpRemoveParentGroup",
	"OpConnectClient",
	"OpDisconnectClient",
}

func NewOperationPerm() svcutil.OperationPerm {
	return svcutil.NewOperationPerm(expectedOperations, operationNames)
}

// External Operations
const (
	DomainOpCreateChannel svcutil.ExternalOperation = iota
	DomainOpListChannel
	GroupOpSetChildChannel
	GroupsOpRemoveChildChannel
	ClientsOpConnectChannel
	ClientsOpDisconnectChannel
)

var expectedExternalOperations = []svcutil.ExternalOperation{
	DomainOpCreateChannel,
	DomainOpListChannel,
	GroupOpSetChildChannel,
	GroupsOpRemoveChildChannel,
	ClientsOpConnectChannel,
	ClientsOpDisconnectChannel,
}

var externalOperationNames = []string{
	"DomainOpCreateChannel",
	"DomainOpListChannel",
	"GroupOpSetChildChannel",
	"GroupsOpRemoveChildChannel",
	"ClientsOpConnectChannel",
	"ClientsOpDisconnectChannel",
}

func NewExternalOperationPerm() svcutil.ExternalOperationPerm {
	return svcutil.NewExternalOperationPerm(expectedExternalOperations, externalOperationNames)
}

// Below codes should moved out of service, may be can be kept in `cmd/<svc>/main.go`

const (
	updatePermission          = "update_permission"
	readPermission            = "read_permission"
	deletePermission          = "delete_permission"
	setParentGroupPermission  = "set_parent_group_permission"
	connectToClientPermission = "connect_to_client_permission"

	manageRolePermission      = "manage_role_permission"
	addRoleUsersPermission    = "add_role_users_permission"
	removeRoleUsersPermission = "remove_role_users_permission"
	viewRoleUsersPermission   = "view_role_users_permission"
)

func NewOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		OpViewChannel:       readPermission,
		OpUpdateChannel:     updatePermission,
		OpUpdateChannelTags: updatePermission,
		OpEnableChannel:     updatePermission,
		OpDisableChannel:    updatePermission,
		OpDeleteChannel:     deletePermission,
		OpSetParentGroup:    setParentGroupPermission,
		OpRemoveParentGroup: setParentGroupPermission,
		OpConnectClient:     connectToClientPermission,
		OpDisconnectClient:  connectToClientPermission,
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

const (
	// External Permission
	// Domains
	domainCreateChannelPermission = "channel_create_permission"
	domainListChanelPermission    = "list_channels_permission"
	// Groups
	groupSetChildChannelPermission    = "channel_create_permission"
	groupRemoveChildChannelPermission = "channel_create_permission"
	// Client
	clientsConnectChannelPermission    = "connect_to_channel_permission"
	clientsDisconnectChannelPermission = "connect_to_channel_permission"
)

func NewExternalOperationPermissionMap() map[svcutil.ExternalOperation]svcutil.Permission {
	extOpPerm := map[svcutil.ExternalOperation]svcutil.Permission{
		DomainOpCreateChannel:      domainCreateChannelPermission,
		DomainOpListChannel:        domainListChanelPermission,
		GroupOpSetChildChannel:     groupSetChildChannelPermission,
		GroupsOpRemoveChildChannel: groupRemoveChildChannelPermission,
		ClientsOpConnectChannel:    clientsConnectChannelPermission,
		ClientsOpDisconnectChannel: clientsDisconnectChannelPermission,
	}
	return extOpPerm
}
