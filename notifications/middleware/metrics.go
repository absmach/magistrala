// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/notifications"
	"github.com/go-kit/kit/metrics"
)

var _ notifications.Notifier = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter  metrics.Counter
	latency  metrics.Histogram
	notifier notifications.Notifier
}

// NewMetrics returns new notifier with metrics middleware.
func NewMetrics(notifier notifications.Notifier, counter metrics.Counter, latency metrics.Histogram) notifications.Notifier {
	return &metricsMiddleware{
		counter:  counter,
		latency:  latency,
		notifier: notifier,
	}
}

func (mm *metricsMiddleware) Notify(ctx context.Context, n notifications.Notification) error {
	defer func(begin time.Time) {
		methodName := notificationTypeToMethodName(n.Type)
		mm.counter.With("method", methodName).Add(1)
		mm.latency.With("method", methodName).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.notifier.Notify(ctx, n)
}
