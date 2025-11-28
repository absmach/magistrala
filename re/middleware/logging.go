// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/messaging"
)

var _ re.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    re.Service
}

func LoggingMiddleware(svc re.Service, logger *slog.Logger) re.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (res re.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.String("rule_name", r.Name),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Add rule failed", args...)
			return
		}
		lm.logger.Info("Add rule completed successfully", args...)
	}(time.Now())
	return lm.svc.AddRule(ctx, session, r)
}

func (lm *loggingMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (res re.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("rule",
				slog.String("id", res.ID),
				slog.String("name", res.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("View rule failed", args...)
			return
		}
		lm.logger.Info("View rule completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewRule(ctx, session, id)
}

func (lm *loggingMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (res re.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("rule",
				slog.String("id", r.ID),
				slog.String("name", r.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update rule failed", args...)
			return
		}
		lm.logger.Info("Update rule completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateRule(ctx, session, r)
}

func (lm *loggingMiddleware) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (res re.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("rule",
				slog.String("id", r.ID),
				slog.String("name", r.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update rule failed", args...)
			return
		}
		lm.logger.Info("Update rule tags completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateRuleTags(ctx, session, r)
}

func (lm *loggingMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (res re.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("rule",
				slog.String("id", r.ID),
				slog.Any("schedule", r.Schedule),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update rule schedule failed", args...)
			return
		}
		lm.logger.Info("Update rule schedule completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateRuleSchedule(ctx, session, r)
}

func (lm *loggingMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (pg re.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("page",
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("total", pg.Total),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List rules failed", args...)
			return
		}
		lm.logger.Info("List rules completed successfully", args...)
	}(time.Now())
	return lm.svc.ListRules(ctx, session, pm)
}

func (lm *loggingMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.String("rule_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Remove rule failed", args...)
			return
		}
		lm.logger.Info("Remove rule completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveRule(ctx, session, id)
}

func (lm *loggingMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (res re.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("rule",
				slog.String("id", res.ID),
				slog.String("name", res.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Enable rule failed", args...)
			return
		}
		lm.logger.Info("Enable rule completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableRule(ctx, session, id)
}

func (lm *loggingMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (res re.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("rule",
				slog.String("id", res.ID),
				slog.String("name", res.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disable rule failed", args...)
			return
		}
		lm.logger.Info("Disable rule completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableRule(ctx, session, id)
}

func (lm *loggingMiddleware) StartScheduler(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Start scheduler failed", args...)
			return
		}
		lm.logger.Info("Start scheduler completed successfully", args...)
	}(time.Now())
	return lm.svc.StartScheduler(ctx)
}

func (lm *loggingMiddleware) Handle(msg *messaging.Message) (err error) {
	defer func(begin time.Time) {
		// Log only failure since the handlers are executed async and will always
		// return nil error. The rest of the loggin is performed in main.go error loop.
		if err != nil {
			args := []any{
				slog.String("duration", time.Since(begin).String()),
			}
			if msg != nil {
				args = append(args,
					slog.String("channel", msg.Channel),
					slog.String("payload_size", fmt.Sprintf("%d", len(msg.Payload))),
				)
			}
			lm.logger.Warn("Message consumption completed", args...)
		}
	}(time.Now())

	err = lm.svc.Handle(msg)
	return
}

func (lm *loggingMiddleware) Cancel() error {
	return lm.svc.Cancel()
}
