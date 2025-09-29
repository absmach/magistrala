// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	grpcReadersV1 "github.com/absmach/magistrala/api/grpc/readers/v1"
	"github.com/absmach/magistrala/pkg/emailer"
	pkglog "github.com/absmach/magistrala/pkg/logger"
	"github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/pkg/ticker"
	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
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
	workerMgr  *WorkerManager
}

func NewService(repo Repository, runInfo chan pkglog.RunInfo, idp supermq.IDProvider, rePubSub messaging.PubSub, writersPub, alarmsPub messaging.Publisher, tck ticker.Ticker, emailer emailer.Emailer, readers grpcReadersV1.ReadersServiceClient) Service {
	reEngine := &re{
		repo:       repo,
		idp:        idp,
		runInfo:    runInfo,
		rePubSub:   rePubSub,
		writersPub: writersPub,
		alarmsPub:  alarmsPub,
		ticker:     tck,
		email:      emailer,
		readers:    readers,
	}
	return reEngine
}

func shouldCreateWorker(rule Rule) bool {
	if rule.Status != EnabledStatus {
		return false
	}

	if rule.Schedule.Recurring == schedule.None {
		return true
	}

	now := time.Now().UTC()
	dueTime := rule.Schedule.Time

	if dueTime.IsZero() || dueTime.Before(now) {
		return true
	}

	return dueTime.Sub(now) <= time.Hour
}

func (re *re) AddRule(ctx context.Context, session authn.Session, r Rule) (Rule, error) {
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
	r.LastRunStatus = NeverRunStatus
	r.ExecutionCount = 0

	if !r.Schedule.StartDateTime.IsZero() {
		r.Schedule.StartDateTime = now
	}
	r.Schedule.Time = r.Schedule.StartDateTime

	rule, err := re.repo.AddRule(ctx, r)
	if err != nil {
		return Rule{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if shouldCreateWorker(rule) {
		re.workerMgr.AddWorker(ctx, rule)
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

	if shouldCreateWorker(rule) {
		re.workerMgr.UpdateWorker(ctx, rule)
	} else {
		re.workerMgr.RemoveWorker(rule.ID)
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

	if shouldCreateWorker(rule) {
		re.workerMgr.UpdateWorker(ctx, rule)
	} else {
		re.workerMgr.RemoveWorker(rule.ID)
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

	re.workerMgr.RemoveWorker(id)

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

	if shouldCreateWorker(rule) {
		re.workerMgr.AddWorker(ctx, rule)
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

	re.workerMgr.RemoveWorker(id)

	return rule, nil
}

func (re *re) Cancel() error {
	return re.workerMgr.StopAll()
}

// updateRuleExecutionStatus updates the execution status of a rule
func (re *re) updateRuleExecutionStatus(ctx context.Context, ruleID string, status ExecutionStatus, errorMessage string) {
	now := time.Now().UTC()
	rule := Rule{
		ID:                  ruleID,
		LastRunStatus:       status,
		LastRunTime:         &now,
		LastRunErrorMessage: errorMessage,
	}

	if status == SuccessStatus || status == PartialSuccessStatus {
		currentRule, err := re.repo.ViewRule(ctx, ruleID)
		if err == nil {
			rule.ExecutionCount = currentRule.ExecutionCount + 1
		}
	}

	if err := re.repo.UpdateRuleExecutionStatus(ctx, rule); err != nil {
		re.runInfo <- pkglog.RunInfo{
			Level:   slog.LevelWarn,
			Message: fmt.Sprintf("failed to update rule execution status: %s", err),
			Details: []slog.Attr{
				slog.String("rule_id", ruleID),
				slog.String("status", status.String()),
			},
		}
	}
}
