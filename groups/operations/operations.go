// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package operations

import "github.com/absmach/magistrala/pkg/permissions"

// Group Operations.
const (
	OpViewGroup permissions.Operation = iota
	OpUpdateGroup
	OpUpdateGroupTags
	OpEnableGroup
	OpDisableGroup
	OpRetrieveGroupHierarchy
	OpAddParentGroup
	OpRemoveParentGroup
	OpAddChildrenGroups
	OpRemoveChildrenGroups
	OpRemoveAllChildrenGroups
	OpListChildrenGroups
	OpDeleteGroup
	OpGroupSetChildClient
	OpGroupRemoveChildClient
	OpGroupSetChildChannel
	OpGroupRemoveChildChannel
	OpListUserGroups
)

func OperationDetails() map[permissions.Operation]permissions.OperationDetails {
	return map[permissions.Operation]permissions.OperationDetails{
		OpViewGroup: {
			Name:               "view",
			PermissionRequired: true,
		},
		OpUpdateGroup: {
			Name:               "update",
			PermissionRequired: true,
		},
		OpUpdateGroupTags: {
			Name:               "update_tags",
			PermissionRequired: true,
		},
		OpEnableGroup: {
			Name:               "enable",
			PermissionRequired: true,
		},
		OpDisableGroup: {
			Name:               "disable",
			PermissionRequired: true,
		},
		OpRetrieveGroupHierarchy: {
			Name:               "retrieve_group_hierarchy",
			PermissionRequired: true,
		},
		OpAddParentGroup: {
			Name:               "add_parent_group",
			PermissionRequired: true,
		},
		OpRemoveParentGroup: {
			Name:               "remove_parent_group",
			PermissionRequired: true,
		},
		OpAddChildrenGroups: {
			Name:               "add_children_groups",
			PermissionRequired: true,
		},
		OpRemoveChildrenGroups: {
			Name:               "remove_children_groups",
			PermissionRequired: true,
		},
		OpRemoveAllChildrenGroups: {
			Name:               "remove_all_children_groups",
			PermissionRequired: true,
		},
		OpListChildrenGroups: {
			Name:               "list_children_groups",
			PermissionRequired: true,
		},
		OpDeleteGroup: {
			Name:               "delete",
			PermissionRequired: true,
		},
		OpGroupSetChildClient: {
			Name:               "set_child_client",
			PermissionRequired: true,
		},
		OpGroupRemoveChildClient: {
			Name:               "remove_child_client",
			PermissionRequired: true,
		},
		OpGroupSetChildChannel: {
			Name:               "set_child_channel",
			PermissionRequired: true,
		},
		OpGroupRemoveChildChannel: {
			Name:               "remove_child_channel",
			PermissionRequired: true,
		},
		OpListUserGroups: {
			Name:               "list_user_groups",
			PermissionRequired: false, // hardcoded to superadmin
		},
	}
}
