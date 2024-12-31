// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/absmach/supermq/consumers"
	"github.com/go-kit/kit/metrics"
)

var _ consumers.BlockingConsumer = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter  metrics.Counter
	latency  metrics.Histogram
	consumer consumers.BlockingConsumer
}

// MetricsMiddleware returns new message repository
// with Save method wrapped to expose metrics.
func MetricsMiddleware(consumer consumers.BlockingConsumer, counter metrics.Counter, latency metrics.Histogram) consumers.BlockingConsumer {
	return &metricsMiddleware{
		counter:  counter,
		latency:  latency,
		consumer: consumer,
	}
}

// ConsumeBlocking instruments ConsumeBlocking method with metrics.
func (mm *metricsMiddleware) ConsumeBlocking(ctx context.Context, msgs interface{}) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "consume").Add(1)
		mm.latency.With("method", "consume").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.consumer.ConsumeBlocking(ctx, msgs)
}
