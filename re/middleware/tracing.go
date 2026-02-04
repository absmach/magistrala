// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/messaging"
	rolemw "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	smqTracing "github.com/absmach/supermq/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    re.Service
	rolemw.RoleManagerTracing
}

var _ re.Service = (*tracingMiddleware)(nil)

func NewTracingMiddleware(tracer trace.Tracer, svc re.Service) re.Service {
	return &tracingMiddleware{
		tracer:             tracer,
		svc:                svc,
		RoleManagerTracing: rolemw.NewTracing("re", svc, tracer),
	}
}

func (tm *tracingMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "add_rule", trace.WithAttributes(
		attribute.String("name", r.Name),
		attribute.String("domain_id", r.DomainID),
	))
	defer span.End()

	return tm.svc.AddRule(ctx, session, r)
}

func (tm *tracingMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "view_rule", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.ViewRule(ctx, session, id)
}

func (tm *tracingMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "update_rule", trace.WithAttributes(
		attribute.String("id", r.ID),
	))
	defer span.End()

	return tm.svc.UpdateRule(ctx, session, r)
}

func (tm *tracingMiddleware) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "update_rule_tags", trace.WithAttributes(
		attribute.String("id", r.ID),
	))
	defer span.End()

	return tm.svc.UpdateRuleTags(ctx, session, r)
}

func (tm *tracingMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "update_rule_schedule", trace.WithAttributes(
		attribute.String("id", r.ID),
	))
	defer span.End()

	return tm.svc.UpdateRuleSchedule(ctx, session, r)
}

func (tm *tracingMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "list_rules", trace.WithAttributes(
		attribute.Int("offset", int(pm.Offset)),
		attribute.Int("limit", int(pm.Limit)),
	))
	defer span.End()

	return tm.svc.ListRules(ctx, session, pm)
}

func (tm *tracingMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "remove_rule", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.RemoveRule(ctx, session, id)
}

func (tm *tracingMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "enable_rule", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.EnableRule(ctx, session, id)
}

func (tm *tracingMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "disable_rule", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.DisableRule(ctx, session, id)
}

func (tm *tracingMiddleware) Handle(msg *messaging.Message) error {
	_, span := smqTracing.StartSpan(context.Background(), tm.tracer, "handle", trace.WithAttributes(
		attribute.String("channel", msg.Channel),
		attribute.String("subtopic", msg.Subtopic),
	))
	defer span.End()

	return tm.svc.Handle(msg)
}

func (tm *tracingMiddleware) StartScheduler(ctx context.Context) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "start_scheduler")
	defer span.End()

	return tm.svc.StartScheduler(ctx)
}

func (tm *tracingMiddleware) Cancel() error {
	return tm.svc.Cancel()
}
