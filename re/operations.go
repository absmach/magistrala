// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
)

const (
	OpAddRule permissions.Operation = iota
	OpViewRule
	OpUpdateRule
	OpUpdateRuleTags
	OpUpdateRuleSchedule
	OpListRules
	OpRemoveRule
	OpEnableRule
	OpDisableRule
)

const (
	OpAddRuleStr            = "OpAddRule"
	OpViewRuleStr           = "OpViewRule"
	OpUpdateRuleStr         = "OpUpdateRule"
	OpUpdateRuleTagsStr     = "OpUpdateRuleTags"
	OpUpdateRuleScheduleStr = "OpUpdateRuleSchedule"
	OpListRulesStr          = "OpListRules"
	OpRemoveRuleStr         = "OpRemoveRule"
	OpEnableRuleStr         = "OpEnableRule"
	OpDisableRuleStr        = "OpDisableRule"
)

func GetPermission(op permissions.Operation) (string, error) {
	if op < OpAddRule || op > OpDisableRule {
		return "", errors.New("invalid operation")
	}
	return policies.MembershipPermission, nil
}
