// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/reports"
	"github.com/absmach/supermq/pkg/authn"
	rolemw "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/go-kit/kit/metrics"
)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	service reports.Service
	rolemw.RoleManagerMetricsMiddleware
}

var _ reports.Service = (*metricsMiddleware)(nil)

func NewMetricsMiddleware(counter metrics.Counter, latency metrics.Histogram, service reports.Service) reports.Service {
	return &metricsMiddleware{
		counter:                      counter,
		latency:                      latency,
		service:                      service,
		RoleManagerMetricsMiddleware: rolemw.NewMetrics("reports", service, counter, latency),
	}
}

func (mm *metricsMiddleware) AddReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "add_report_config").Add(1)
		mm.latency.With("method", "add_report_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.AddReportConfig(ctx, session, cfg)
}

func (mm *metricsMiddleware) ViewReportConfig(ctx context.Context, session authn.Session, id string, withRoles bool) (reports.ReportConfig, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_report_config").Add(1)
		mm.latency.With("method", "view_report_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ViewReportConfig(ctx, session, id, withRoles)
}

func (mm *metricsMiddleware) UpdateReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_report_config").Add(1)
		mm.latency.With("method", "update_report_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.UpdateReportConfig(ctx, session, cfg)
}

func (mm *metricsMiddleware) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_report_schedule").Add(1)
		mm.latency.With("method", "update_report_schedule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.UpdateReportSchedule(ctx, session, cfg)
}

func (mm *metricsMiddleware) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_report_config").Add(1)
		mm.latency.With("method", "remove_report_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.RemoveReportConfig(ctx, session, id)
}

func (mm *metricsMiddleware) ListReportsConfig(ctx context.Context, session authn.Session, pm reports.PageMeta) (reports.ReportConfigPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_reports_config").Add(1)
		mm.latency.With("method", "list_reports_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ListReportsConfig(ctx, session, pm)
}

func (mm *metricsMiddleware) EnableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "enable_report_config").Add(1)
		mm.latency.With("method", "enable_report_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.EnableReportConfig(ctx, session, id)
}

func (mm *metricsMiddleware) DisableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "disable_report_config").Add(1)
		mm.latency.With("method", "disable_report_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.DisableReportConfig(ctx, session, id)
}

func (mm *metricsMiddleware) UpdateReportTemplate(ctx context.Context, session authn.Session, cfg reports.ReportConfig) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_report_template").Add(1)
		mm.latency.With("method", "update_report_template").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.UpdateReportTemplate(ctx, session, cfg)
}

func (mm *metricsMiddleware) ViewReportTemplate(ctx context.Context, session authn.Session, id string) (reports.ReportTemplate, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_report_template").Add(1)
		mm.latency.With("method", "view_report_template").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ViewReportTemplate(ctx, session, id)
}

func (mm *metricsMiddleware) DeleteReportTemplate(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_report_template").Add(1)
		mm.latency.With("method", "delete_report_template").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.DeleteReportTemplate(ctx, session, id)
}

func (mm *metricsMiddleware) GenerateReport(ctx context.Context, session authn.Session, config reports.ReportConfig, action reports.ReportAction) (reports.ReportPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "generate_report").Add(1)
		mm.latency.With("method", "generate_report").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.GenerateReport(ctx, session, config, action)
}

func (mm *metricsMiddleware) StartScheduler(ctx context.Context) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "start_scheduler").Add(1)
		mm.latency.With("method", "start_scheduler").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.StartScheduler(ctx)
}
