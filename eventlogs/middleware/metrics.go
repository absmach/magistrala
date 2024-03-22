// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/eventlogs"
	"github.com/go-kit/kit/metrics"
)

var _ eventlogs.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	service eventlogs.Service
}

// MetricsMiddleware returns new message repository
// with Save method wrapped to expose metrics.
func MetricsMiddleware(service eventlogs.Service, counter metrics.Counter, latency metrics.Histogram) eventlogs.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		service: service,
	}
}

func (mm *metricsMiddleware) ReadAll(ctx context.Context, token string, page eventlogs.Page) (eventlogs.EventsPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "read_all").Add(1)
		mm.latency.With("method", "read_all").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.service.ReadAll(ctx, token, page)
}
