// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package handler

import (
	"context"
	"time"

	"github.com/absmach/mproxy/pkg/session"
	"github.com/go-kit/kit/metrics"
)

var _ session.Handler = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     session.Handler
}

// MetricsMiddleware instruments adapter by tracking request count and latency.
func MetricsMiddleware(svc session.Handler, counter metrics.Counter, latency metrics.Histogram) session.Handler {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

// AuthConnect implements session.Handler.
func (mm *metricsMiddleware) AuthConnect(ctx context.Context) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.AuthConnect(ctx)
}

// AuthPublish implements session.Handler.
func (mm *metricsMiddleware) AuthPublish(ctx context.Context, topic *string, payload *[]byte) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.AuthPublish(ctx, topic, payload)
}

// AuthSubscribe implements session.Handler.
func (*metricsMiddleware) AuthSubscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Connect implements session.Handler.
func (*metricsMiddleware) Connect(ctx context.Context) error {
	return nil
}

// Disconnect implements session.Handler.
func (*metricsMiddleware) Disconnect(ctx context.Context) error {
	return nil
}

// Publish instruments Publish method with metrics.
func (mm *metricsMiddleware) Publish(ctx context.Context, topic *string, payload *[]byte) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Publish(ctx, topic, payload)
}

// Subscribe implements session.Handler.
func (*metricsMiddleware) Subscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Unsubscribe implements session.Handler.
func (*metricsMiddleware) Unsubscribe(ctx context.Context, topics *[]string) error {
	return nil
}
