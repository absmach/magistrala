// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
)

func TestAtomServiceCreateAlarmProjectsCreatedAlarm(t *testing.T) {
	projector := &alarmProjector{}
	svc := WithAtom(alarmService{
		create: Alarm{
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
		},
	}, projector)

	created, err := svc.CreateAlarm(context.Background(), Alarm{RuleID: "rule-1"})
	if err != nil {
		t.Fatalf("create alarm: %v", err)
	}
	if created.ID != "alarm-1" {
		t.Fatalf("unexpected created alarm: %#v", created)
	}
	if projector.resource.ID != "alarm-1" || projector.resource.Kind != atom.KindAlarm {
		t.Fatalf("unexpected projection: %#v", projector.resource)
	}
	if projector.resource.Attributes["rule_id"] != "rule-1" {
		t.Fatalf("missing rule projection: %#v", projector.resource.Attributes)
	}
	if projector.resource.Attributes["value"] != "92.4" || projector.resource.Attributes["threshold"] != "80" {
		t.Fatalf("missing alarm value projection: %#v", projector.resource.Attributes)
	}
}

type alarmService struct {
	create Alarm
}

func (svc alarmService) CreateAlarm(context.Context, Alarm) (Alarm, error) {
	return svc.create, nil
}

func (svc alarmService) UpdateAlarm(context.Context, authn.Session, Alarm) (Alarm, error) {
	return Alarm{}, nil
}

func (svc alarmService) ViewAlarm(context.Context, authn.Session, string) (Alarm, error) {
	return Alarm{}, nil
}

func (svc alarmService) ListAlarms(context.Context, authn.Session, PageMetadata) (AlarmsPage, error) {
	return AlarmsPage{}, nil
}

func (svc alarmService) DeleteAlarm(context.Context, authn.Session, string) error {
	return nil
}

type alarmProjector struct {
	atom.Projector
	resource atom.Resource
}

func (p *alarmProjector) UpsertResource(_ context.Context, resource atom.Resource) error {
	p.resource = resource
	return nil
}
