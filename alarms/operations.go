// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
)

const (
	OpAddAlarm = iota
	OpViewAlarm
	OpListAlarms
	OpUpdateAlarm
	OpDeleteAlarm
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
