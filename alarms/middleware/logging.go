// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5/middleware"
)

type loggingMiddleware struct {
	logger  *slog.Logger
	service alarms.Service
}

var _ alarms.Service = (*loggingMiddleware)(nil)

func NewLoggingMiddleware(logger *slog.Logger, service alarms.Service) alarms.Service {
	return &loggingMiddleware{
		logger:  logger,
		service: service,
	}
}

func (lm *loggingMiddleware) CreateRule(ctx context.Context, session authn.Session, rule alarms.Rule) (dbr alarms.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("rule",
				slog.String("id", dbr.ID),
				slog.String("name", dbr.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Create rule failed", args...)
			return
		}
		lm.logger.Info("Create rule completed successfully", args...)
	}(time.Now())

	return lm.service.CreateRule(ctx, session, rule)
}

func (lm *loggingMiddleware) UpdateRule(ctx context.Context, session authn.Session, rule alarms.Rule) (dbr alarms.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("rule",
				slog.String("id", dbr.ID),
				slog.String("name", dbr.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update rule failed", args...)
			return
		}
		lm.logger.Info("Update rule completed successfully", args...)
	}(time.Now())

	return lm.service.UpdateRule(ctx, session, rule)
}

func (lm *loggingMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (dbr alarms.Rule, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.String("id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View rule failed", args...)
			return
		}
		lm.logger.Info("View rule completed successfully", args...)
	}(time.Now())

	return lm.service.ViewRule(ctx, session, id)
}

func (lm *loggingMiddleware) ListRules(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (dbp alarms.RulesPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Int("offset", int(pm.Offset)),
			slog.Int("limit", int(pm.Limit)),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List rules failed", args...)
			return
		}
		lm.logger.Info("List rules completed successfully", args...)
	}(time.Now())

	return lm.service.ListRules(ctx, session, pm)
}

func (lm *loggingMiddleware) DeleteRule(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.String("id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete rule failed", args...)
			return
		}
		lm.logger.Info("Delete rule completed successfully", args...)
	}(time.Now())

	return lm.service.DeleteRule(ctx, session, id)
}

func (lm *loggingMiddleware) CreateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (dba alarms.Alarm, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("alarm",
				slog.String("id", dba.ID),
				slog.String("rule_id", dba.RuleID),
				slog.String("message", dba.Message),
				slog.String("status", dba.Status.String()),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Create alarm failed", args...)
			return
		}
		lm.logger.Info("Create alarm completed successfully", args...)
	}(time.Now())

	return lm.service.CreateAlarm(ctx, session, alarm)
}

func (lm *loggingMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (dba alarms.Alarm, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("alarm",
				slog.String("id", dba.ID),
				slog.String("rule_id", dba.RuleID),
				slog.String("message", dba.Message),
				slog.String("status", dba.Status.String()),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update alarm failed", args...)
			return
		}
		lm.logger.Info("Update alarm completed successfully", args...)
	}(time.Now())

	return lm.service.UpdateAlarm(ctx, session, alarm)
}

func (lm *loggingMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (dba alarms.Alarm, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.String("id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View alarm failed", args...)
			return
		}
		lm.logger.Info("View alarm completed successfully", args...)
	}(time.Now())

	return lm.service.ViewAlarm(ctx, session, id)
}

func (lm *loggingMiddleware) ListAlarms(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (dbp alarms.AlarmsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Int("offset", int(pm.Offset)),
			slog.Int("limit", int(pm.Limit)),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List alarms failed", args...)
			return
		}
		lm.logger.Info("List alarms completed successfully", args...)
	}(time.Now())

	return lm.service.ListAlarms(ctx, session, pm)
}

func (lm *loggingMiddleware) DeleteAlarm(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.String("id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete alarm failed", args...)
			return
		}
		lm.logger.Info("Delete alarm completed successfully", args...)
	}(time.Now())

	return lm.service.DeleteAlarm(ctx, session, id)
}

func (lm *loggingMiddleware) AssignAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("alarm",
				slog.String("id", alarm.ID),
				slog.String("assignee_id", alarm.AssigneeID),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Assign alarm failed", args...)
			return
		}
		lm.logger.Info("Assign alarm completed successfully", args...)
	}(time.Now())

	return lm.service.AssignAlarm(ctx, session, alarm)
}
