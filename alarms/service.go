// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
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

func (s *service) CreateRule(ctx context.Context, session authn.Session, rule Rule) (Rule, error) {
	id, err := s.idp.ID()
	if err != nil {
		return Rule{}, err
	}
	rule.ID = id
	rule.CreatedAt = time.Now()
	rule.CreatedBy = session.UserID
	rule.DomainID = session.DomainID

	return s.repo.CreateRule(ctx, rule)
}

func (s *service) UpdateRule(ctx context.Context, session authn.Session, rule Rule) (Rule, error) {
	rule.UpdatedAt = time.Now()
	rule.UpdatedBy = session.UserID

	return s.repo.UpdateRule(ctx, rule)
}

func (s *service) ViewRule(ctx context.Context, session authn.Session, ruleID string) (Rule, error) {
	return s.repo.ViewRule(ctx, ruleID)
}

func (s *service) ListRules(ctx context.Context, session authn.Session, pm PageMetadata) (RulesPage, error) {
	return s.repo.ListRules(ctx, pm)
}

func (s *service) DeleteRule(ctx context.Context, session authn.Session, ruleID string) error {
	return s.repo.DeleteRule(ctx, ruleID)
}

func (s *service) CreateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error) {
	id, err := s.idp.ID()
	if err != nil {
		return Alarm{}, err
	}
	alarm.ID = id
	alarm.CreatedAt = time.Now()
	alarm.CreatedBy = session.UserID
	alarm.DomainID = session.DomainID

	return s.repo.CreateAlarm(ctx, alarm)
}

func (s *service) ViewAlarm(ctx context.Context, session authn.Session, alarmID string) (Alarm, error) {
	return s.repo.ViewAlarm(ctx, alarmID)
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

func (s *service) AssignAlarm(ctx context.Context, session authn.Session, alarm Alarm) error {
	alarm.UpdatedAt = time.Now()
	alarm.UpdatedBy = session.UserID

	return s.repo.AssignAlarm(ctx, alarm)
}
