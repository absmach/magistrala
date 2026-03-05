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

func (mm *metricsMiddleware) CreateAlarm(ctx context.Context, alarm alarms.Alarm) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "create_alarm").Add(1)
		mm.latency.With("method", "create_alarm").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.CreateAlarm(ctx, alarm)
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
