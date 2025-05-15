// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"math"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
)

type service struct {
	idp  supermq.IDProvider
	repo Repository
}

var _ Service = (*service)(nil)

func NewService(idp supermq.IDProvider, repo Repository) Service {
	return &service{
		idp:  idp,
		repo: repo,
	}
}

func (s *service) CreateAlarm(ctx context.Context, alarm Alarm) error {
	id, err := s.idp.ID()
	if err != nil {
		return err
	}
	alarm.ID = id
	if alarm.CreatedAt.IsZero() {
		alarm.CreatedAt = time.Now()
	}

	if err := alarm.Validate(); err != nil {
		return err
	}

	pm := PageMetadata{
		Limit:       1,
		Offset:      0,
		DomainID:    alarm.DomainID,
		RuleID:      alarm.RuleID,
		ChannelID:   alarm.ChannelID,
		ClientID:    alarm.ClientID,
		Subtopic:    alarm.Subtopic,
		Measurement: alarm.Measurement,
		Status:      AllStatus,
		Severity:    math.MaxUint8,
		CreatedTill: alarm.CreatedAt,
	}

	// Retrieve the last alarm of (DomainID, RuleID, ChannelID, ClientID, Subtopic, Measurement)
	lastAlarms, err := s.repo.ListAlarms(ctx, pm)
	if err != nil {
		return err
	}

	// Exit conditions
	switch alarm.Status {
	case ClearedStatus:
		// No alarm created yet and received alarm cleared status
		// Which mean no alarm created yet.
		if len(lastAlarms.Alarms) == 0 || lastAlarms.Alarms[0].Status == ClearedStatus {
			return nil
		}
	case ActiveStatus:
		if len(lastAlarms.Alarms) > 0 && lastAlarms.Alarms[0].Status == ActiveStatus && lastAlarms.Alarms[0].Severity == alarm.Severity {
			return nil
		}
	}

	_, err = s.repo.CreateAlarm(ctx, alarm)

	return err
}

func (s *service) ViewAlarm(ctx context.Context, session authn.Session, alarmID string) (Alarm, error) {
	return s.repo.ViewAlarm(ctx, alarmID, session.DomainID)
}

func (s *service) ListAlarms(ctx context.Context, session authn.Session, pm PageMetadata) (AlarmsPage, error) {
	return s.repo.ListAlarms(ctx, pm)
}

func (s *service) DeleteAlarm(ctx context.Context, session authn.Session, alarmID string) error {
	return s.repo.DeleteAlarm(ctx, alarmID)
}

func (s *service) UpdateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error) {
	alarm.UpdatedAt = time.Now()
	alarm.UpdatedBy = session.UserID

	return s.repo.UpdateAlarm(ctx, alarm)
}
