// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package operations

import (
	"github.com/absmach/supermq/pkg/permissions"
)

// Report Operations.
const (
	OpAddReportConfig permissions.Operation = iota
	OpViewReportConfig
	OpUpdateReportConfig
	OpUpdateReportSchedule
	OpRemoveReportConfig
	OpListReportsConfig
	OpEnableReportConfig
	OpDisableReportConfig
	OpGenerateReport
	OpUpdateReportTemplate
	OpViewReportTemplate
	OpDeleteReportTemplate
)

func OperationDetails() map[permissions.Operation]permissions.OperationDetails {
	return map[permissions.Operation]permissions.OperationDetails{
		OpAddReportConfig: {
			Name:               "add",
			PermissionRequired: true,
		},
		OpViewReportConfig: {
			Name:               "view",
			PermissionRequired: true,
		},
		OpUpdateReportConfig: {
			Name:               "update",
			PermissionRequired: true,
		},
		OpUpdateReportSchedule: {
			Name:               "update_schedule",
			PermissionRequired: true,
		},
		OpRemoveReportConfig: {
			Name:               "delete",
			PermissionRequired: true,
		},
		OpListReportsConfig: {
			Name:               "list_reports",
			PermissionRequired: false, // hardcoded to superadmin
		},
		OpEnableReportConfig: {
			Name:               "enable",
			PermissionRequired: true,
		},
		OpDisableReportConfig: {
			Name:               "disable",
			PermissionRequired: true,
		},
		OpGenerateReport: {
			Name:               "generate",
			PermissionRequired: true,
		},
		OpUpdateReportTemplate: {
			Name:               "update_template",
			PermissionRequired: true,
		},
		OpViewReportTemplate: {
			Name:               "view_template",
			PermissionRequired: true,
		},
		OpDeleteReportTemplate: {
			Name:               "delete_template",
			PermissionRequired: true,
		},
	}
}
