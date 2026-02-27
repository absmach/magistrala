// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
)

const (
	EntityType = "reports"
)

const (
	OpAddReportConfig = iota
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

const (
	OpAddReportConfigStr      = "OpAddReportConfig"
	OpViewReportConfigStr     = "OpViewReportConfig"
	OpUpdateReportConfigStr   = "OpUpdateReportConfig"
	OpUpdateReportScheduleStr = "OpUpdateReportSchedule"
	OpRemoveReportConfigStr   = "OpRemoveReportConfig"
	OpListReportsConfigStr    = "OpListReportsConfig"
	OpEnableReportConfigStr   = "OpEnableReportConfig"
	OpDisableReportConfigStr  = "OpDisableReportConfig"
	OpGenerateReportStr       = "OpGenerateReport"
	OpUpdateReportTemplateStr = "OpUpdateReportTemplate"
	OpViewReportTemplateStr   = "OpViewReportTemplate"
	OpDeleteReportTemplateStr = "OpDeleteReportTemplate"
)

func GetPermission(op permissions.Operation) (string, error) {
	if op < OpAddReportConfig || op > OpDeleteReportTemplate {
		return "", errors.New("invalid operation")
	}
	return policies.MembershipPermission, nil
}

func OperationName(op permissions.Operation) string {
	switch op {
	case OpAddReportConfig:
		return OpAddReportConfigStr
	case OpViewReportConfig:
		return OpViewReportConfigStr
	case OpUpdateReportConfig:
		return OpUpdateReportConfigStr
	case OpUpdateReportSchedule:
		return OpUpdateReportScheduleStr
	case OpRemoveReportConfig:
		return OpRemoveReportConfigStr
	case OpListReportsConfig:
		return OpListReportsConfigStr
	case OpEnableReportConfig:
		return OpEnableReportConfigStr
	case OpDisableReportConfig:
		return OpDisableReportConfigStr
	case OpGenerateReport:
		return OpGenerateReportStr
	case OpUpdateReportTemplate:
		return OpUpdateReportTemplateStr
	case OpViewReportTemplate:
		return OpViewReportTemplateStr
	case OpDeleteReportTemplate:
		return OpDeleteReportTemplateStr
	default:
		return "unknown"
	}
}
