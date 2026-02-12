// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/messaging"
	rolemw "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/go-kit/kit/metrics"
)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	service re.Service
	rolemw.RoleManagerMetricsMiddleware
}

var _ re.Service = (*metricsMiddleware)(nil)

func NewMetricsMiddleware(counter metrics.Counter, latency metrics.Histogram, service re.Service) re.Service {
	return &metricsMiddleware{
		counter:                      counter,
		latency:                      latency,
		service:                      service,
		RoleManagerMetricsMiddleware: rolemw.NewMetrics("re", service, counter, latency),
	}
}

func (mm *metricsMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "add_rule").Add(1)
		mm.latency.With("method", "add_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.AddRule(ctx, session, r)
}

func (mm *metricsMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_rule").Add(1)
		mm.latency.With("method", "view_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ViewRule(ctx, session, id)
}

func (mm *metricsMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_rule").Add(1)
		mm.latency.With("method", "update_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.UpdateRule(ctx, session, r)
}

func (mm *metricsMiddleware) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_rule_tags").Add(1)
		mm.latency.With("method", "update_rule_tags").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.UpdateRuleTags(ctx, session, r)
}

func (mm *metricsMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_rule_schedule").Add(1)
		mm.latency.With("method", "update_rule_schedule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.UpdateRuleSchedule(ctx, session, r)
}

func (mm *metricsMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_rules").Add(1)
		mm.latency.With("method", "list_rules").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ListRules(ctx, session, pm)
}

func (mm *metricsMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_rule").Add(1)
		mm.latency.With("method", "remove_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.RemoveRule(ctx, session, id)
}

func (mm *metricsMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "enable_rule").Add(1)
		mm.latency.With("method", "enable_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.EnableRule(ctx, session, id)
}

func (mm *metricsMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "disable_rule").Add(1)
		mm.latency.With("method", "disable_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.DisableRule(ctx, session, id)
}

func (mm *metricsMiddleware) Handle(msg *messaging.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "handle").Add(1)
		mm.latency.With("method", "handle").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.Handle(msg)
}

func (mm *metricsMiddleware) StartScheduler(ctx context.Context) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "start_scheduler").Add(1)
		mm.latency.With("method", "start_scheduler").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.StartScheduler(ctx)
}

func (mm *metricsMiddleware) Cancel() error {
	return mm.service.Cancel()
}
