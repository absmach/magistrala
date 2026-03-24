// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"github.com/absmach/supermq/pkg/permissions"
)

const EntityType = "channels"

// Channel Operations.
const (
	OpViewChannel permissions.Operation = iota
	OpUpdateChannel
	OpUpdateChannelTags
	OpEnableChannel
	OpDisableChannel
	OpDeleteChannel
	OpSetParentGroup
	OpRemoveParentGroup
	OpConnectClient
	OpDisconnectClient
	OpListUserChannels
)

func OperationDetails() map[permissions.Operation]permissions.OperationDetails {
	return map[permissions.Operation]permissions.OperationDetails{
		OpViewChannel: {
			Name:               "view",
			PermissionRequired: true,
		},
		OpUpdateChannel: {
			Name:               "update",
			PermissionRequired: true,
		},
		OpUpdateChannelTags: {
			Name:               "update_tags",
			PermissionRequired: true,
		},
		OpEnableChannel: {
			Name:               "enable",
			PermissionRequired: true,
		},
		OpDisableChannel: {
			Name:               "disable",
			PermissionRequired: true,
		},
		OpDeleteChannel: {
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
		OpConnectClient: {
			Name:               "connect_client",
			PermissionRequired: true,
		},
		OpDisconnectClient: {
			Name:               "disconnect_client",
			PermissionRequired: true,
		},
		OpListUserChannels: {
			Name:               "list_user_channels",
			PermissionRequired: false, // hardcoded to superadmin
		},
	}
}
