// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"testing"
	"time"

	"github.com/absmach/magistrala/pkg/schedule"
)

func TestRuleProjectionOmitsMetadataAndSchedule(t *testing.T) {
	got := ruleProjection(Rule{
		ID:           "rule-1",
		Name:         "high-temp",
		DomainID:     "domain-1",
		CreatedBy:    "user-1",
		Status:       EnabledStatus,
		Tags:         []string{"smoke"},
		Metadata:     Metadata{"flow": "encoded-flow", "other": "value"},
		InputChannel: "channel-1",
		InputTopic:   "messages",
		Schedule: schedule.Schedule{
			Time: time.Date(2026, 6, 26, 17, 0, 0, 0, time.UTC),
		},
	})

	if _, ok := got.Attributes["metadata"]; ok {
		t.Fatalf("rule metadata should not be projected to Atom attributes: %+v", got.Attributes)
	}
	if _, ok := got.Attributes["scheduled_at"]; ok {
		t.Fatalf("rule schedule should not be projected to Atom attributes: %+v", got.Attributes)
	}
	if got.Attributes["input_channel"] != "channel-1" {
		t.Fatalf("unexpected input_channel: %+v", got.Attributes)
	}
	if got.Attributes["input_topic"] != "messages" {
		t.Fatalf("unexpected input_topic: %+v", got.Attributes)
	}
}
