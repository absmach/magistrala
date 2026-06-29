// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"fmt"
	"time"

	"github.com/absmach/magistrala/internal/atom"
)

func alarmProjection(a Alarm) atom.Resource {
	res := atom.ResourceFromFields(atom.ObjectFields{
		ID:        a.ID,
		Kind:      atom.KindAlarm,
		Name:      alarmName(a),
		TenantID:  a.DomainID,
		OwnerID:   a.AssigneeID,
		Status:    a.Status.String(),
		Metadata:  map[string]any(a.Metadata),
		UpdatedBy: a.UpdatedBy,
		CreatedAt: a.CreatedAt,
		UpdatedAt: a.UpdatedAt,
	})
	res.Attributes["rule_id"] = a.RuleID
	res.Attributes["channel_id"] = a.ChannelID
	res.Attributes["client_id"] = a.ClientID
	res.Attributes["subtopic"] = a.Subtopic
	res.Attributes["severity"] = a.Severity
	res.Attributes["measurement"] = a.Measurement
	res.Attributes["value"] = a.Value
	res.Attributes["unit"] = a.Unit
	res.Attributes["threshold"] = a.Threshold
	res.Attributes["cause"] = a.Cause
	res.Attributes["alarm_status"] = uint8(a.Status)
	res.Attributes["assignee_id"] = a.AssigneeID
	res.Attributes["assigned_at"] = alarmTimeString(a.AssignedAt)
	res.Attributes["assigned_by"] = a.AssignedBy
	res.Attributes["acknowledged_at"] = alarmTimeString(a.AcknowledgedAt)
	res.Attributes["acknowledged_by"] = a.AcknowledgedBy
	res.Attributes["resolved_at"] = alarmTimeString(a.ResolvedAt)
	res.Attributes["resolved_by"] = a.ResolvedBy
	return res
}

func alarmName(a Alarm) string {
	if a.Cause != "" && a.Measurement != "" {
		return fmt.Sprintf("%s: %s", a.Measurement, a.Cause)
	}
	if a.Cause != "" {
		return a.Cause
	}
	if a.Measurement != "" {
		return fmt.Sprintf("%s alarm", a.Measurement)
	}
	return a.ID
}

func alarmTimeString(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.Format(time.RFC3339Nano)
}
