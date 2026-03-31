// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package operations

import "github.com/absmach/supermq/pkg/permissions"

const EntityType = "alarm"

// Alarm Operations.
const (
	OpViewAlarm permissions.Operation = iota
	OpDeleteAlarm
	OpListAlarms
	OpAssignAlarm
	OpAcknowledgeAlarm
	OpResolveAlarm
	OpUpdateAlarm
)

func OperationDetails() map[permissions.Operation]permissions.OperationDetails {
	return map[permissions.Operation]permissions.OperationDetails{
		OpViewAlarm: {
			Name:               "view",
			PermissionRequired: true,
		},
		OpDeleteAlarm: {
			Name:               "delete",
			PermissionRequired: true,
		},
		OpListAlarms: {
			Name:               "list",
			PermissionRequired: true,
		},
		OpAssignAlarm: {
			Name:               "assign",
			PermissionRequired: true,
		},
		OpAcknowledgeAlarm: {
			Name:               "acknowledge",
			PermissionRequired: true,
		},
		OpResolveAlarm: {
			Name:               "resolve",
			PermissionRequired: true,
		},
		OpUpdateAlarm: {
			Name:               "update",
			PermissionRequired: true,
		},
	}
}
