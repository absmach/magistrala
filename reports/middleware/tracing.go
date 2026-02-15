// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/reports"
	"github.com/absmach/supermq/pkg/authn"
	rolemw "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	smqTracing "github.com/absmach/supermq/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    reports.Service
	rolemw.RoleManagerTracing
}

var _ reports.Service = (*tracingMiddleware)(nil)

func NewTracingMiddleware(tracer trace.Tracer, svc reports.Service) reports.Service {
	return &tracingMiddleware{
		tracer:             tracer,
		svc:                svc,
		RoleManagerTracing: rolemw.NewTracing("reports", svc, tracer),
	}
}

func (tm *tracingMiddleware) AddReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "add_report_config", trace.WithAttributes(
		attribute.String("name", cfg.Name),
		attribute.String("domain_id", cfg.DomainID),
	))
	defer span.End()

	return tm.svc.AddReportConfig(ctx, session, cfg)
}

func (tm *tracingMiddleware) ViewReportConfig(ctx context.Context, session authn.Session, id string, withRoles bool) (reports.ReportConfig, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "view_report_config", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.ViewReportConfig(ctx, session, id, withRoles)
}

func (tm *tracingMiddleware) UpdateReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "update_report_config", trace.WithAttributes(
		attribute.String("id", cfg.ID),
	))
	defer span.End()

	return tm.svc.UpdateReportConfig(ctx, session, cfg)
}

func (tm *tracingMiddleware) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "update_report_schedule", trace.WithAttributes(
		attribute.String("id", cfg.ID),
	))
	defer span.End()

	return tm.svc.UpdateReportSchedule(ctx, session, cfg)
}

func (tm *tracingMiddleware) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "remove_report_config", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.RemoveReportConfig(ctx, session, id)
}

func (tm *tracingMiddleware) ListReportsConfig(ctx context.Context, session authn.Session, pm reports.PageMeta) (reports.ReportConfigPage, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "list_reports_config", trace.WithAttributes(
		attribute.Int("offset", int(pm.Offset)),
		attribute.Int("limit", int(pm.Limit)),
	))
	defer span.End()

	return tm.svc.ListReportsConfig(ctx, session, pm)
}

func (tm *tracingMiddleware) EnableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "enable_report_config", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.EnableReportConfig(ctx, session, id)
}

func (tm *tracingMiddleware) DisableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "disable_report_config", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.DisableReportConfig(ctx, session, id)
}

func (tm *tracingMiddleware) UpdateReportTemplate(ctx context.Context, session authn.Session, cfg reports.ReportConfig) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "update_report_template", trace.WithAttributes(
		attribute.String("id", cfg.ID),
	))
	defer span.End()

	return tm.svc.UpdateReportTemplate(ctx, session, cfg)
}

func (tm *tracingMiddleware) ViewReportTemplate(ctx context.Context, session authn.Session, id string) (reports.ReportTemplate, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "view_report_template", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.ViewReportTemplate(ctx, session, id)
}

func (tm *tracingMiddleware) DeleteReportTemplate(ctx context.Context, session authn.Session, id string) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "delete_report_template", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.DeleteReportTemplate(ctx, session, id)
}

func (tm *tracingMiddleware) GenerateReport(ctx context.Context, session authn.Session, config reports.ReportConfig, action reports.ReportAction) (reports.ReportPage, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "generate_report", trace.WithAttributes(
		attribute.String("config_id", config.ID),
		attribute.String("action", string(action)),
	))
	defer span.End()

	return tm.svc.GenerateReport(ctx, session, config, action)
}

func (tm *tracingMiddleware) StartScheduler(ctx context.Context) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "start_scheduler")
	defer span.End()

	return tm.svc.StartScheduler(ctx)
}
