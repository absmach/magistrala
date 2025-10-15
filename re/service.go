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

func (re *re) AbortRuleExecution(ctx context.Context, session authn.Session, id string) error {
	rule, err := re.repo.ViewRule(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if rule.LastRunStatus != InProgressStatus && rule.LastRunStatus != QueuedStatus {
		return errors.Wrap(errors.New(fmt.Sprintf("cannot abort rule with status '%s': rule must be in 'in_progress' or 'queued' status",
			rule.LastRunStatus.String())), svcerr.ErrMalformedEntity)
	}

	if re.workerMgr != nil {
		// Also check if worker actually exists and is running
		workerStatus := re.workerMgr.GetWorkerStatus(id)
		if workerStatus == nil {
			return errors.Wrap(errors.New("no active worker found for this rule"), svcerr.ErrNotFound)
		}

		running, ok := workerStatus["running"].(bool)
		if !ok || !running {
			return errors.Wrap(errors.New("cannot abort: worker is not currently running"), svcerr.ErrMalformedEntity)
		}

		queueLen, _ := workerStatus["queue_length"].(int)
		if queueLen == 0 && rule.LastRunStatus != InProgressStatus {
			return errors.Wrap(errors.New("cannot abort: no execution in progress"), svcerr.ErrMalformedEntity)
		}

		re.workerMgr.AbortRule(id)
	}

	return nil
}

func (re *re) GetRuleExecutionStatus(ctx context.Context, session authn.Session, id string) (RuleExecutionStatus, error) {
	rule, err := re.repo.ViewRule(ctx, id)
	if err != nil {
		return RuleExecutionStatus{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	status := RuleExecutionStatus{
		Rule:          rule,
		WorkerRunning: false,
		QueueLength:   0,
	}

	if re.workerMgr != nil {
		workerStatus := re.workerMgr.GetWorkerStatus(id)
		if workerStatus != nil {
			if running, ok := workerStatus["running"].(bool); ok {
				status.WorkerRunning = running
			}
			if queueLen, ok := workerStatus["queue_length"].(int); ok {
				status.QueueLength = queueLen
			}
		}
	}

	return status, nil
}

func (re *re) updateRuleExecutionStatus(ctx context.Context, ruleID string, status ExecutionStatus, err error) {
	now := time.Now().UTC()
	rule := Rule{
		ID:            ruleID,
		LastRunStatus: status,
		LastRunTime:   &now,
	}

	if err != nil {
		rule.LastRunErrorMessage = err.Error()
	}

	// Debug: Log the status being set
	fmt.Printf("[DEBUG] updateRuleExecutionStatus: rule_id=%s, status=%s, has_error=%v\n", ruleID, status.String(), err != nil)

	// Always fetch the current rule to get the current execution count
	currentRule, viewErr := re.repo.ViewRule(ctx, ruleID)
	if viewErr != nil {
		fmt.Printf("[WARN] Failed to retrieve current rule: rule_id=%s, error=%v\n", ruleID, viewErr)
		// If we can't fetch the rule, set count based on status
		switch status {
		case SuccessStatus, PartialSuccessStatus, FailureStatus:
			rule.ExecutionCount = 1
		default:
			rule.ExecutionCount = 0
		}
	} else {
		// Start with current count
		rule.ExecutionCount = currentRule.ExecutionCount

		// Add 1 if completed successfully, add 0 otherwise (preserve count)
		switch status {
		case SuccessStatus, PartialSuccessStatus, FailureStatus:
			rule.ExecutionCount = currentRule.ExecutionCount + 1
			fmt.Printf("[DEBUG] Incremented execution count: rule_id=%s, old_count=%d, new_count=%d\n", ruleID, currentRule.ExecutionCount, rule.ExecutionCount)
		default:
			fmt.Printf("[DEBUG] Preserving execution count: rule_id=%s, status=%s, count=%d\n", ruleID, status.String(), rule.ExecutionCount)
		}
	}

	fmt.Printf("[DEBUG] About to update rule execution status in database: rule_id=%s, status=%s, execution_count=%d\n", ruleID, status.String(), rule.ExecutionCount)

	if err := re.repo.UpdateRuleExecutionStatus(ctx, rule); err != nil {
		fmt.Printf("[ERROR] Failed to update rule execution status in database: rule_id=%s, error=%v\n", ruleID, err)

		re.runInfo <- pkglog.RunInfo{
			Level:   slog.LevelWarn,
			Message: fmt.Sprintf("failed to update rule execution status: %s", err),
			Details: []slog.Attr{
				slog.String("rule_id", ruleID),
				slog.String("status", status.String()),
			},
		}
	} else {
		fmt.Printf("[DEBUG] Successfully updated rule execution status in database: rule_id=%s, status=%s, execution_count=%d\n", ruleID, status.String(), rule.ExecutionCount)
	}
}
