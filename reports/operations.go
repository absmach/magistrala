// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
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

func GetPermission(op permissions.Operation) (string, error) {
	if op < OpAddReportConfig || op > OpDeleteReportTemplate {
		return "", errors.New("invalid operation")
	}
	return policies.MembershipPermission, nil
}
