// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
)

const (
	EntityType = "alarms"
)

const (
	OpAddAlarm = iota
	OpViewAlarm
	OpListAlarms
	OpUpdateAlarm
	OpDeleteAlarm
)

const (
	OpAddAlarmStr    = "OpAddAlarm"
	OpViewAlarmStr   = "OpViewAlarm"
	OpListAlarmsStr  = "OpListAlarms"
	OpUpdateAlarmStr = "OpUpdateAlarm"
	OpDeleteAlarmStr = "OpDeleteAlarm"
)

func GetPermission(op permissions.Operation) (string, error) {
	if op < OpAddAlarm || op > OpDeleteAlarm {
		return "", errors.New("invalid operation")
	}

	if op == OpUpdateAlarm || op == OpDeleteAlarm {
		return policies.AdminPermission, nil
	}

	return policies.MembershipPermission, nil
}

func OperationName(op permissions.Operation) string {
	switch op {
	case OpAddAlarm:
		return OpAddAlarmStr
	case OpViewAlarm:
		return OpViewAlarmStr
	case OpListAlarms:
		return OpListAlarmsStr
	case OpUpdateAlarm:
		return OpUpdateAlarmStr
	case OpDeleteAlarm:
		return OpDeleteAlarmStr
	default:
		return "unknown"
	}
}
