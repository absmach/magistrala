// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"github.com/absmach/supermq/pkg/permissions"
)

// Rule Operations.
const (
	OpAddRule permissions.Operation = iota
	OpViewRule
	OpUpdateRule
	OpUpdateRuleTags
	OpUpdateRuleSchedule
	OpRemoveRule
	OpListRules
	OpEnableRule
	OpDisableRule
)

func OperationDetails() map[permissions.Operation]permissions.OperationDetails {
	return map[permissions.Operation]permissions.OperationDetails{
		OpAddRule: {
			Name:               "add",
			PermissionRequired: true,
		},
		OpViewRule: {
			Name:               "view",
			PermissionRequired: true,
		},
		OpUpdateRule: {
			Name:               "update",
			PermissionRequired: true,
		},
		OpUpdateRuleTags: {
			Name:               "update_tags",
			PermissionRequired: true,
		},
		OpUpdateRuleSchedule: {
			Name:               "update_schedule",
			PermissionRequired: true,
		},
		OpRemoveRule: {
			Name:               "delete",
			PermissionRequired: true,
		},
		OpListRules: {
			Name:               "list",
			PermissionRequired: true,
		},
		OpEnableRule: {
			Name:               "enable",
			PermissionRequired: true,
		},
		OpDisableRule: {
			Name:               "disable",
			PermissionRequired: true,
		},
	}
}
