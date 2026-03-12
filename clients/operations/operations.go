// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"github.com/absmach/supermq/pkg/permissions"
)

const EntityType = "clients"

// Client Operations.
const (
	OpViewClient permissions.Operation = iota
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
	OpListUserClients
)

func OperationDetails() map[permissions.Operation]permissions.OperationDetails {
	return map[permissions.Operation]permissions.OperationDetails{
		OpViewClient: {
			Name:               "view",
			PermissionRequired: true,
		},
		OpUpdateClient: {
			Name:               "update",
			PermissionRequired: true,
		},
		OpUpdateClientTags: {
			Name:               "update_tags",
			PermissionRequired: true,
		},
		OpUpdateClientSecret: {
			Name:               "update_secret",
			PermissionRequired: true,
		},
		OpEnableClient: {
			Name:               "enable",
			PermissionRequired: true,
		},
		OpDisableClient: {
			Name:               "disable",
			PermissionRequired: true,
		},
		OpDeleteClient: {
			Name:               "delete",
			PermissionRequired: true,
		},
		OpSetParentGroup: {
			Name:               "set_parent_group",
			PermissionRequired: true,
		},
		OpRemoveParentGroup: {
			Name:               "remove_parent_group",
			PermissionRequired: true,
		},
		OpConnectToChannel: {
			Name:               "connect_to_channel",
			PermissionRequired: true,
		},
		OpDisconnectFromChannel: {
			Name:               "disconnect_from_channel",
			PermissionRequired: true,
		},
		OpListUserClients: {
			Name:               "list_user_clients",
			PermissionRequired: false, // hardcoded to superadmin
		},
	}
}
