// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/emailer"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/ticker"
)

var (
	ErrGoroutinesNotAllowed = errors.New("goroutines are not allowed in Go scripts")
	ErrPanicNotAllowed      = errors.New("panic is not allowed in Go scripts")
)

type re struct {
	repo       Repository
	runInfo    chan pkglog.RunInfo
	idp        magistrala.IDProvider
	rePubSub   messaging.PubSub
	writersPub messaging.Publisher
	alarmsPub  messaging.Publisher
	ticker     ticker.Ticker
	email      emailer.Emailer
	readers    grpcReadersV1.ReadersServiceClient
}

func NewService(repo Repository, runInfo chan pkglog.RunInfo, idp magistrala.IDProvider, rePubSub messaging.PubSub, writersPub, alarmsPub messaging.Publisher, tck ticker.Ticker, emailer emailer.Emailer, readers grpcReadersV1.ReadersServiceClient) (Service, error) {
	return &re{
		repo:       repo,
		idp:        idp,
		runInfo:    runInfo,
		rePubSub:   rePubSub,
		writersPub: writersPub,
		alarmsPub:  alarmsPub,
		ticker:     tck,
		email:      emailer,
		readers:    readers,
	}, nil
}

func (re *re) AddRule(ctx context.Context, session authn.Session, r Rule) (retRule Rule, retErr error) {
	if r.Logic.Type == GoType && goKeywordRegex.MatchString(r.Logic.Value) {
		return Rule{}, errors.Wrap(svcerr.ErrMalformedEntity, ErrGoroutinesNotAllowed)
	}
	if r.Logic.Type == GoType && panicRegex.MatchString(r.Logic.Value) {
		return Rule{}, errors.Wrap(svcerr.ErrMalformedEntity, ErrPanicNotAllowed)
	}

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

	return rule, nil
}

func (re *re) ViewRule(ctx context.Context, session authn.Session, id string, withRoles bool) (Rule, error) {
	rule, err := re.repo.ViewRule(ctx, id)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return rule, nil
}

func (re *re) UpdateRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
	if r.Logic.Type == GoType && goKeywordRegex.MatchString(r.Logic.Value) {
		return Rule{}, errors.Wrap(svcerr.ErrMalformedEntity, ErrGoroutinesNotAllowed)
	}
	if r.Logic.Type == GoType && panicRegex.MatchString(r.Logic.Value) {
		return Rule{}, errors.Wrap(svcerr.ErrMalformedEntity, ErrPanicNotAllowed)
	}

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
	page, err := re.repo.ListAllRules(ctx, pm)
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
