// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/authz"
)

var _ authz.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     authz.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc authz.Service, counter metrics.Counter, latency metrics.Histogram) authz.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) AddPolicy(ctx context.Context, token string, p authz.Policy) (bool, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_policy").Add(1)
		ms.latency.With("method", "add_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AddPolicy(ctx, token, p)
}

func (ms *metricsMiddleware) RemovePolicy(ctx context.Context, token string, p authz.Policy) (bool, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_policy").Add(1)
		ms.latency.With("method", "remove_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemovePolicy(ctx, token, p)
}

func (ms *metricsMiddleware) Authorize(ctx context.Context, p authz.Policy) (bool, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "authorize").Add(1)
		ms.latency.With("method", "authorize").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Authorize(ctx, p)
}
