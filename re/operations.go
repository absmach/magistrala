// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
)

const (
	EntityType = "rules"
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

func OperationName(op permissions.Operation) string {
	switch op {
	case OpAddRule:
		return OpAddRuleStr
	case OpViewRule:
		return OpViewRuleStr
	case OpUpdateRule:
		return OpUpdateRuleStr
	case OpUpdateRuleTags:
		return OpUpdateRuleTagsStr
	case OpUpdateRuleSchedule:
		return OpUpdateRuleScheduleStr
	case OpListRules:
		return OpListRulesStr
	case OpRemoveRule:
		return OpRemoveRuleStr
	case OpEnableRule:
		return OpEnableRuleStr
	case OpDisableRule:
		return OpDisableRuleStr
	default:
		return "unknown"
	}
}
