// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/absmach/supermq/consumers/notifiers"
	"github.com/go-kit/kit/metrics"
)

var _ notifiers.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     notifiers.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc notifiers.Service, counter metrics.Counter, latency metrics.Histogram) notifiers.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

// CreateSubscription instruments CreateSubscription method with metrics.
func (ms *metricsMiddleware) CreateSubscription(ctx context.Context, token string, sub notifiers.Subscription) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_subscription").Add(1)
		ms.latency.With("method", "create_subscription").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateSubscription(ctx, token, sub)
}

// ViewSubscription instruments ViewSubscription method with metrics.
func (ms *metricsMiddleware) ViewSubscription(ctx context.Context, token, topic string) (notifiers.Subscription, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_subscription").Add(1)
		ms.latency.With("method", "view_subscription").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewSubscription(ctx, token, topic)
}

// ListSubscriptions instruments ListSubscriptions method with metrics.
func (ms *metricsMiddleware) ListSubscriptions(ctx context.Context, token string, pm notifiers.PageMetadata) (notifiers.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_subscriptions").Add(1)
		ms.latency.With("method", "list_subscriptions").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListSubscriptions(ctx, token, pm)
}

// RemoveSubscription instruments RemoveSubscription method with metrics.
func (ms *metricsMiddleware) RemoveSubscription(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_subscription").Add(1)
		ms.latency.With("method", "remove_subscription").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveSubscription(ctx, token, id)
}

// ConsumeBlocking instruments ConsumeBlocking method with metrics.
func (ms *metricsMiddleware) ConsumeBlocking(ctx context.Context, msg interface{}) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "consume").Add(1)
		ms.latency.With("method", "consume").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ConsumeBlocking(ctx, msg)
}
