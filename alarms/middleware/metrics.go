// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/go-kit/kit/metrics"
)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	service alarms.Service
}

var _ alarms.Service = (*metricsMiddleware)(nil)

func NewMetricsMiddleware(counter metrics.Counter, latency metrics.Histogram, service alarms.Service) alarms.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		service: service,
	}
}

func (mm *metricsMiddleware) CreateRule(ctx context.Context, session authn.Session, rule alarms.Rule) (alarms.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "create_rule").Add(1)
		mm.latency.With("method", "create_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.CreateRule(ctx, session, rule)
}

func (mm *metricsMiddleware) UpdateRule(ctx context.Context, session authn.Session, rule alarms.Rule) (alarms.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_rule").Add(1)
		mm.latency.With("method", "update_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.UpdateRule(ctx, session, rule)
}

func (mm *metricsMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (alarms.Rule, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "get_rule").Add(1)
		mm.latency.With("method", "get_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ViewRule(ctx, session, id)
}

func (mm *metricsMiddleware) ListRules(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (alarms.RulesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_rules").Add(1)
		mm.latency.With("method", "list_rules").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ListRules(ctx, session, pm)
}

func (mm *metricsMiddleware) DeleteRule(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_rule").Add(1)
		mm.latency.With("method", "delete_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.DeleteRule(ctx, session, id)
}

func (mm *metricsMiddleware) CreateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (alarms.Alarm, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "create_alarm").Add(1)
		mm.latency.With("method", "create_alarm").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.CreateAlarm(ctx, session, alarm)
}

func (mm *metricsMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (alarms.Alarm, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_alarm").Add(1)
		mm.latency.With("method", "update_alarm").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.UpdateAlarm(ctx, session, alarm)
}

func (mm *metricsMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (alarms.Alarm, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "get_alarm").Add(1)
		mm.latency.With("method", "get_alarm").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ViewAlarm(ctx, session, id)
}

func (mm *metricsMiddleware) ListAlarms(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_alarms").Add(1)
		mm.latency.With("method", "list_alarms").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ListAlarms(ctx, session, pm)
}

func (mm *metricsMiddleware) DeleteAlarm(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_alarm").Add(1)
		mm.latency.With("method", "delete_alarm").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.DeleteAlarm(ctx, session, id)
}

func (mm *metricsMiddleware) AssignAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "assign_alarm").Add(1)
		mm.latency.With("method", "assign_alarm").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.AssignAlarm(ctx, session, alarm)
}
