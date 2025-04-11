// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/pkg/authn"
	smqTracing "github.com/absmach/supermq/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    alarms.Service
}

var _ alarms.Service = (*tracingMiddleware)(nil)

func NewTracingMiddleware(tracer trace.Tracer, svc alarms.Service) alarms.Service {
	return &tracingMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

func (tm *tracingMiddleware) CreateAlarm(ctx context.Context, alarm alarms.Alarm) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "create_alarm", trace.WithAttributes(
		attribute.String("rule_id", alarm.RuleID),
		attribute.String("measurement", alarm.Measurement),
		attribute.String("value", alarm.Value),
		attribute.String("unit", alarm.Unit),
		attribute.String("cause", alarm.Cause),
		attribute.String("status", alarm.Status.String()),
	))
	defer span.End()

	return tm.svc.CreateAlarm(ctx, alarm)
}

func (tm *tracingMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (alarms.Alarm, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "update_alarm", trace.WithAttributes(
		attribute.String("rule_id", alarm.RuleID),
		attribute.String("measurement", alarm.Measurement),
		attribute.String("value", alarm.Value),
		attribute.String("unit", alarm.Unit),
		attribute.String("cause", alarm.Cause),
		attribute.String("status", alarm.Status.String()),
	))
	defer span.End()

	return tm.svc.UpdateAlarm(ctx, session, alarm)
}

func (tm *tracingMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (alarms.Alarm, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "get_alarm", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.ViewAlarm(ctx, session, id)
}

func (tm *tracingMiddleware) ListAlarms(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "list_alarms", trace.WithAttributes(
		attribute.Int("offset", int(pm.Offset)),
		attribute.Int("limit", int(pm.Limit)),
	))
	defer span.End()

	return tm.svc.ListAlarms(ctx, session, pm)
}

func (tm *tracingMiddleware) DeleteAlarm(ctx context.Context, session authn.Session, id string) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "delete_alarm", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.DeleteAlarm(ctx, session, id)
}
