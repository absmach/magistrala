// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"time"

	"github.com/absmach/magistrala/alarms/operations"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
)

type service struct {
	idp    supermq.IDProvider
	repo   Repository
	policy policies.Service
}

var _ Service = (*service)(nil)

func NewService(idp supermq.IDProvider, repo Repository, policy policies.Service) Service {
	return &service{
		idp:    idp,
		repo:   repo,
		policy: policy,
	}
}

func (s *service) CreateAlarm(ctx context.Context, alarm Alarm) (retErr error) {
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

	defer func() {
		if retErr != nil {
			if errRollBack := s.repo.DeleteAlarm(ctx, alarm.ID); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(svcerr.ErrRollbackRepo, errRollBack))
			}
		}
	}()

	if err := s.policy.AddPolicies(ctx, []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     alarm.DomainID,
			Relation:    policies.DomainRelation,
			ObjectType:  operations.EntityType,
			Object:      alarm.ID,
		},
	}); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
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
	if err := s.repo.DeleteAlarm(ctx, alarmID); err != nil {
		return err
	}

	if err := s.policy.DeletePolicies(ctx, []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     session.DomainID,
			Relation:    policies.DomainRelation,
			ObjectType:  operations.EntityType,
			Object:      alarmID,
		},
	}); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	return nil
}

func (s *service) UpdateAlarm(ctx context.Context, session authn.Session, alarm Alarm) (Alarm, error) {
	alarm.UpdatedAt = time.Now()
	alarm.UpdatedBy = session.UserID

	for i := range alarm.Comments {
		id, err := s.idp.ID()
		if err != nil {
			return Alarm{}, err
		}
		alarm.Comments[i].ID = id
		alarm.Comments[i].AlarmID = alarm.ID
		alarm.Comments[i].DomainID = session.DomainID
		alarm.Comments[i].UserID = session.UserID
		alarm.Comments[i].CreatedAt = time.Now()
	}

	return s.repo.UpdateAlarm(ctx, alarm)
}
