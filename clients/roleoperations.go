// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
)

// Internal Operations

const (
	OpViewClient svcutil.Operation = iota
	OpUpdateClient
	OpUpdateClientTags
	OpUpdateClientSecret
	OpEnableClient
	OpDisableClient
	OpDeleteClient
	OpSetParentGroup
	OpRemoveParentGroup
	OpConnectToChannel
	OpDisconnectFromChannel
)

var expectedOperations = []svcutil.Operation{
	OpViewClient,
	OpUpdateClient,
	OpUpdateClientTags,
	OpUpdateClientSecret,
	OpEnableClient,
	OpDisableClient,
	OpDeleteClient,
	OpSetParentGroup,
	OpRemoveParentGroup,
	OpConnectToChannel,
	OpDisconnectFromChannel,
}

var operationNames = []string{
	"OpViewClient",
	"OpUpdateClient",
	"OpUpdateClientTags",
	"OpUpdateClientSecret",
	"OpEnableClient",
	"OpDisableClient",
	"OpDeleteClient",
	"OpSetParentGroup",
	"OpRemoveParentGroup",
	"OpConnectToChannel",
	"OpDisconnectFromChannel",
}

func NewOperationPerm() svcutil.OperationPerm {
	return svcutil.NewOperationPerm(expectedOperations, operationNames)
}

// External Operations
const (
	DomainOpCreateClient svcutil.ExternalOperation = iota
	DomainOpListClients
	GroupOpSetChildClient
	GroupsOpRemoveChildClient
	ChannelsOpConnectChannel
	ChannelsOpDisconnectChannel
)

var expectedExternalOperations = []svcutil.ExternalOperation{
	DomainOpCreateClient,
	DomainOpListClients,
	GroupOpSetChildClient,
	GroupsOpRemoveChildClient,
	ChannelsOpConnectChannel,
	ChannelsOpDisconnectChannel,
}

var externalOperationNames = []string{
	"DomainOpCreateClient",
	"DomainOpListClients",
	"GroupOpSetChildClient",
	"GroupsOpRemoveChildClient",
	"ChannelsOpConnectChannel",
	"ChannelsOpDisconnectChannel",
}

func NewExternalOperationPerm() svcutil.ExternalOperationPerm {
	return svcutil.NewExternalOperationPerm(expectedExternalOperations, externalOperationNames)
}

// Below codes should moved out of service, may be can be kept in `cmd/<svc>/main.go`

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

func NewOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		OpViewClient:            readPermission,
		OpUpdateClient:          updatePermission,
		OpUpdateClientTags:      updatePermission,
		OpUpdateClientSecret:    updatePermission,
		OpEnableClient:          updatePermission,
		OpDisableClient:         updatePermission,
		OpDeleteClient:          deletePermission,
		OpSetParentGroup:        setParentGroupPermission,
		OpRemoveParentGroup:     setParentGroupPermission,
		OpConnectToChannel:      connectToChannelPermission,
		OpDisconnectFromChannel: connectToChannelPermission,
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
	domainCreateClientPermission = "client_create_permission"
	domainListClientsPermission  = "list_clients_permission"
	// Groups
	groupSetChildClientPermission    = "client_create_permission"
	groupRemoveChildClientPermission = "client_create_permission"
	// Channels
	channelsConnectClientPermission    = "connect_to_client_permission"
	channelsDisconnectClientPermission = "connect_to_client_permission"
)

func NewExternalOperationPermissionMap() map[svcutil.ExternalOperation]svcutil.Permission {
	extOpPerm := map[svcutil.ExternalOperation]svcutil.Permission{
		DomainOpCreateClient:        domainCreateClientPermission,
		DomainOpListClients:         domainListClientsPermission,
		GroupOpSetChildClient:       groupSetChildClientPermission,
		GroupsOpRemoveChildClient:   groupRemoveChildClientPermission,
		ChannelsOpConnectChannel:    channelsConnectClientPermission,
		ChannelsOpDisconnectChannel: channelsDisconnectClientPermission,
	}
	return extOpPerm
}
