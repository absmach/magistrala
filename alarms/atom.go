// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"

	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
)

type atomService struct {
	Service
	projector atom.Projector
}

func WithAtom(svc Service, projector atom.Projector) Service {
	if projector == nil {
		return svc
	}
	return atomService{Service: svc, projector: projector}
}

func (svc atomService) CreateAlarm(ctx context.Context, alarm Alarm) (Alarm, error) {
	created, err := svc.Service.CreateAlarm(ctx, alarm)
	if err != nil {
		return created, err
	}
	if created.ID == "" {
		return created, nil
	}
	if err := svc.projector.UpsertResource(ctx, alarmProjection(created)); err != nil {
		return created, nil
	}
	return created, nil
}

func (svc atomService) UpdateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error) {
	updated, err := svc.Service.UpdateAlarm(ctx, session, alarm)
	if err != nil {
		return updated, err
	}
	if err := svc.projector.UpsertResource(ctx, alarmProjection(updated)); err != nil {
		return updated, nil
	}
	return updated, nil
}

func (svc atomService) DeleteAlarm(ctx context.Context, session authn.Session, id string) error {
	if err := svc.Service.DeleteAlarm(ctx, session, id); err != nil {
		return err
	}
	_ = svc.projector.DeleteResource(ctx, id)
	return nil
}

func alarmProjection(a Alarm) atom.Resource {
	res := atom.ResourceFromFields(atom.ObjectFields{
		ID:        a.ID,
		Kind:      atom.KindAlarm,
		Name:      a.Cause,
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
	res.Attributes["severity"] = a.Severity
	res.Attributes["measurement"] = a.Measurement
	res.Attributes["assignee_id"] = a.AssigneeID
	return res
}
