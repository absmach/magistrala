// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/emailer"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/ticker"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
)

type re struct {
	repo       Repository
	runInfo    chan pkglog.RunInfo
	idp        supermq.IDProvider
	rePubSub   messaging.PubSub
	writersPub messaging.Publisher
	alarmsPub  messaging.Publisher
	ticker     ticker.Ticker
	email      emailer.Emailer
	readers    grpcReadersV1.ReadersServiceClient
	roles.ProvisionManageService
}

func NewService(repo Repository, runInfo chan pkglog.RunInfo, policy policies.Service, idp supermq.IDProvider, rePubSub messaging.PubSub, writersPub, alarmsPub messaging.Publisher, tck ticker.Ticker, emailer emailer.Emailer, readers grpcReadersV1.ReadersServiceClient, availableActions []roles.Action, builtInRoles map[roles.BuiltInRoleName][]roles.Action) (Service, error) {
	rpms, err := roles.NewProvisionManageService(mgPolicies.RuleType, repo, policy, idp, availableActions, builtInRoles)
	if err != nil {
		return nil, err
	}
	return &re{
		repo:                   repo,
		idp:                    idp,
		runInfo:                runInfo,
		rePubSub:               rePubSub,
		writersPub:             writersPub,
		alarmsPub:              alarmsPub,
		ticker:                 tck,
		email:                  emailer,
		readers:                readers,
		ProvisionManageService: rpms,
	}, nil
}

func (re *re) AddRule(ctx context.Context, session authn.Session, r Rule) (retRule Rule, retErr error) {
	id, err := re.idp.ID()
	if err != nil {
		return Rule{}, err
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.ID = id
	r.CreatedBy = session.UserID
	r.DomainID = session.DomainID
	r.Status = EnabledStatus

	if !r.Schedule.StartDateTime.IsZero() {
		r.Schedule.StartDateTime = now
	}
	r.Schedule.Time = r.Schedule.StartDateTime

	rule, err := re.repo.AddRule(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	defer func() {
		if retErr != nil {
			if errRollBack := re.repo.RemoveRule(ctx, rule.ID); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(svcerr.ErrRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		BuiltInRoleAdmin: {roles.Member(session.UserID)},
	}

	optionalPolicies := []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     session.DomainID,
			Relation:    policies.DomainRelation,
			ObjectType:  mgPolicies.RuleType,
			Object:      rule.ID,
		},
	}

	_, err = re.AddNewEntitiesRoles(ctx, session.DomainID, session.UserID, []string{rule.ID}, optionalPolicies, newBuiltInRoleMembers)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return rule, nil
}

func (re *re) ViewRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	rule, err := re.repo.ViewRule(ctx, id)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return rule, nil
}

func (re *re) UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	r.UpdatedAt = time.Now().UTC()
	r.UpdatedBy = session.UserID
	rule, err := re.repo.UpdateRule(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return rule, nil
}

func (re *re) UpdateRuleTags(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	r.UpdatedAt = time.Now().UTC()
	r.UpdatedBy = session.UserID
	rule, err := re.repo.UpdateRuleTags(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return rule, nil
}

func (re *re) UpdateRuleSchedule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	r.UpdatedAt = time.Now().UTC()
	r.UpdatedBy = session.UserID
	rule, err := re.repo.UpdateRuleSchedule(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return rule, nil
}

func (re *re) ListRules(ctx context.Context, session authn.Session, pm PageMeta) (Page, error) {
	pm.Domain = session.DomainID
	page, err := re.repo.ListRules(ctx, pm)
	if err != nil {
		return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return page, nil
}

func (re *re) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	if err := re.repo.RemoveRule(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (re *re) EnableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	status, err := ToStatus(Enabled)
	if err != nil {
		return Rule{}, err
	}
	r := Rule{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: session.UserID,
		Status:    status,
	}
	rule, err := re.repo.UpdateRuleStatus(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return rule, nil
}

func (re *re) DisableRule(ctx context.Context, session authn.Session, id string) (Rule, error) {
	status, err := ToStatus(Disabled)
	if err != nil {
		return Rule{}, err
	}
	r := Rule{
		ID:        id,
		UpdatedAt: time.Now().UTC(),
		UpdatedBy: session.UserID,
		Status:    status,
	}
	rule, err := re.repo.UpdateRuleStatus(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return rule, nil
}

func (re *re) Cancel() error {
	return nil
}
