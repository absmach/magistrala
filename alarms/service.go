// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package alarms

import (
	"context"
	"time"

	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
)

type service struct {
	idp  supermq.IDProvider
	repo Repository
	roles.ProvisionManageService
}

var _ Service = (*service)(nil)

func NewService(policy policies.Service, idp supermq.IDProvider, repo Repository, availableActions []roles.Action, builtInRoles map[roles.BuiltInRoleName][]roles.Action) (Service, error) {
	rpms, err := roles.NewProvisionManageService(mgPolicies.AlarmType, repo, policy, idp, availableActions, builtInRoles)
	if err != nil {
		return nil, err
	}
	return &service{
		idp:                    idp,
		repo:                   repo,
		ProvisionManageService: rpms,
	}, nil
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

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		BuiltInRoleAdmin: {roles.Member(alarm.CreatedBy)},
	}

	optionalPolicies := []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     alarm.DomainID,
			Relation:    policies.DomainRelation,
			ObjectType:  mgPolicies.AlarmType,
			Object:      alarm.ID,
		},
	}

	_, err = s.AddNewEntitiesRoles(ctx, alarm.DomainID, alarm.CreatedBy, []string{alarm.ID}, optionalPolicies, newBuiltInRoleMembers)
	if err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return nil
}

func (s *service) ViewAlarm(ctx context.Context, session authn.Session, alarmID string, withRoles bool) (Alarm, error) {
	var alarm Alarm
	var err error
	switch withRoles {
	case true:
		alarm, err = s.repo.RetrieveByIDWithRoles(ctx, alarmID, session.UserID)
	default:
		alarm, err = s.repo.ViewAlarm(ctx, alarmID, session.DomainID)
	}
	return alarm, err
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
