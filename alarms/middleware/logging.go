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

func (lm *loggingMiddleware) CreateAlarm(ctx context.Context, alarm alarms.Alarm) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("alarm",
				slog.String("rule_id", alarm.RuleID),
				slog.String("domain_id", alarm.DomainID),
				slog.String("channel_id", alarm.ChannelID),
				slog.String("client_id", alarm.ClientID),
				slog.String("subtopic", alarm.Subtopic),
				slog.String("measurement", alarm.Measurement),
				slog.String("value", alarm.Value),
				slog.String("unit", alarm.Unit),
				slog.Uint64("status", uint64(alarm.Status)),
				slog.Uint64("severity", uint64(alarm.Severity)),
				slog.String("threshold", alarm.Threshold),
				slog.String("cause", alarm.Cause),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Create alarm failed", args...)
			return
		}
		if alarm.ID != "" {
			lm.logger.Info("Create alarm completed successfully", args...)
		}
	}(time.Now())

	return lm.service.CreateAlarm(ctx, alarm)
}

func (lm *loggingMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (dba alarms.Alarm, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("alarm",
				slog.String("id", dba.ID),
				slog.String("rule_id", dba.RuleID),
				slog.String("domain_id", dba.DomainID),
				slog.String("channel_id", dba.ChannelID),
				slog.String("client_id", dba.ClientID),
				slog.String("subtopic", dba.Subtopic),
				slog.String("measurement", dba.Measurement),
				slog.String("value", dba.Value),
				slog.String("unit", dba.Unit),
				slog.String("status", dba.Status.String()),
				slog.Uint64("severity", uint64(dba.Severity)),
				slog.String("threshold", dba.Threshold),
				slog.String("cause", dba.Cause),
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
			slog.String("rule_id", pm.RuleID),
			slog.String("domain_id", pm.DomainID),
			slog.String("channel_id", pm.ChannelID),
			slog.String("client_id", pm.ClientID),
			slog.String("subtopic", pm.Subtopic),
			slog.String("status", pm.Status.String()),
			slog.Uint64("severity", uint64(pm.Severity)),
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
