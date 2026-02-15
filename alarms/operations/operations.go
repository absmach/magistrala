// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"github.com/absmach/supermq/pkg/permissions"
)

// Alarm Operations.
const (
	OpViewAlarm permissions.Operation = iota
	OpUpdateAlarm
	OpDeleteAlarm
	OpAddAlarm
	OpListAlarms
)

func OperationDetails() map[permissions.Operation]permissions.OperationDetails {
	return map[permissions.Operation]permissions.OperationDetails{
		OpAddAlarm: {
			Name:               "add",
			PermissionRequired: true,
		},
		OpViewAlarm: {
			Name:               "view",
			PermissionRequired: true,
		},
		OpUpdateAlarm: {
			Name:               "update",
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
	}
}
