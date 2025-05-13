// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/reports"
	"github.com/absmach/supermq/pkg/authn"
)

var _ reports.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    reports.Service
}

func LoggingMiddleware(svc reports.Service, logger *slog.Logger) reports.Service {
	return &loggingMiddleware{logger, svc}
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

func (lm *loggingMiddleware) GenerateReport(ctx context.Context, session authn.Session, config reports.ReportConfig, action reports.ReportAction) (page reports.ReportPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Generate report failed", args...)
			return
		}
		lm.logger.Info("Generate report completed", args...)
	}(time.Now())

	return lm.svc.GenerateReport(ctx, session, config, action)
}

func (lm *loggingMiddleware) AddReportConfig(ctx context.Context, session authn.Session, config reports.ReportConfig) (res reports.ReportConfig, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.String("report_name", config.Name),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Add report config failed", args...)
			return
		}
		lm.logger.Info("Add report config completed successfully", args...)
	}(time.Now())
	return lm.svc.AddReportConfig(ctx, session, config)
}

func (lm *loggingMiddleware) ViewReportConfig(ctx context.Context, session authn.Session, id string) (res reports.ReportConfig, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("report_config",
				slog.String("id", res.ID),
				slog.String("name", res.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("View report config failed", args...)
			return
		}
		lm.logger.Info("View report config completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewReportConfig(ctx, session, id)
}

func (lm *loggingMiddleware) UpdateReportConfig(ctx context.Context, session authn.Session, config reports.ReportConfig) (res reports.ReportConfig, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("report_config",
				slog.String("id", config.ID),
				slog.String("name", config.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update report config failed", args...)
			return
		}
		lm.logger.Info("Update report config completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateReportConfig(ctx, session, config)
}

func (lm *loggingMiddleware) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (res reports.ReportConfig, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("report",
				slog.String("id", cfg.ID),
				slog.Any("schedule", cfg.Schedule),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update report schedule failed", args...)
			return
		}
		lm.logger.Info("Update report schedule completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateReportSchedule(ctx, session, cfg)
}

func (lm *loggingMiddleware) ListReportsConfig(ctx context.Context, session authn.Session, pm reports.PageMeta) (pg reports.ReportConfigPage, err error) {
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
			lm.logger.Warn("List reports config failed", args...)
			return
		}
		lm.logger.Info("List reports config completed successfully", args...)
	}(time.Now())
	return lm.svc.ListReportsConfig(ctx, session, pm)
}

func (lm *loggingMiddleware) DisableReportConfig(ctx context.Context, session authn.Session, id string) (res reports.ReportConfig, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("report_config",
				slog.String("id", res.ID),
				slog.String("name", res.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disable report config failed", args...)
			return
		}
		lm.logger.Info("Disable report config completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableReportConfig(ctx, session, id)
}

func (lm *loggingMiddleware) EnableReportConfig(ctx context.Context, session authn.Session, id string) (res reports.ReportConfig, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.Group("report_config",
				slog.String("id", res.ID),
				slog.String("name", res.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Enable report config failed", args...)
			return
		}
		lm.logger.Info("Enable report config completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableReportConfig(ctx, session, id)
}

func (lm *loggingMiddleware) RemoveReportConfig(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("domain_id", session.DomainID),
			slog.String("report_config_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Remove report config failed", args...)
			return
		}
		lm.logger.Info("Remove report config completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveReportConfig(ctx, session, id)
}
