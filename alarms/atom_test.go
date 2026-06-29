// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"testing"

	"github.com/absmach/magistrala/internal/atom"
)

func TestAlarmProjectionBuildsAtomResource(t *testing.T) {
	resource := alarmProjection(Alarm{
		ID:          "alarm-1",
		RuleID:      "rule-1",
		DomainID:    "domain-1",
		ChannelID:   "channel-1",
		ClientID:    "client-1",
		Cause:       "high temperature",
		Measurement: "temperature",
		Value:       "92.4",
		Unit:        "C",
		Threshold:   "80",
		Severity:    90,
		Status:      ActiveStatus,
	})

	if resource.ID != "alarm-1" || resource.Kind != atom.KindAlarm || resource.Name != "alarm-1" {
		t.Fatalf("unexpected projection: %#v", resource)
	}
	if resource.Attributes["rule_id"] != "rule-1" {
		t.Fatalf("missing rule projection: %#v", resource.Attributes)
	}
	if resource.Attributes["value"] != "92.4" || resource.Attributes["threshold"] != "80" {
		t.Fatalf("missing alarm value projection: %#v", resource.Attributes)
	}
}
