// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
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

	if _, err = s.repo.CreateAlarm(ctx, alarm); err != nil && err != repoerr.ErrNotFound {
		return err
	}
	return nil
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
