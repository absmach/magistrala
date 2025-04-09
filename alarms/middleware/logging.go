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

func (lm *loggingMiddleware) CreateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (dba alarms.Alarm, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("alarm",
				slog.String("id", dba.ID),
				slog.String("rule_id", dba.RuleID),
				slog.String("measurement", dba.Measurement),
				slog.String("value", dba.Value),
				slog.String("unit", dba.Unit),
				slog.String("cause", dba.Cause),
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
				slog.String("measurement", dba.Measurement),
				slog.String("value", dba.Value),
				slog.String("unit", dba.Unit),
				slog.String("cause", dba.Cause),
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
