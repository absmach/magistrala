// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/things/policies"
)

var _ policies.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     policies.Service
}

// MetricsMiddleware returns a new metrics middleware wrapper.
func MetricsMiddleware(svc policies.Service, counter metrics.Counter, latency metrics.Histogram) policies.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) AddPolicy(ctx context.Context, token string, external bool, p policies.Policy) (policy policies.Policy, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_policy").Add(1)
		ms.latency.With("method", "add_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.AddPolicy(ctx, token, external, p)
}

func (ms *metricsMiddleware) UpdatePolicy(ctx context.Context, token string, p policies.Policy) (policy policies.Policy, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_policy").Add(1)
		ms.latency.With("method", "update_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdatePolicy(ctx, token, p)
}

func (ms *metricsMiddleware) ListPolicies(ctx context.Context, token string, p policies.Page) (policypage policies.PolicyPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_policies").Add(1)
		ms.latency.With("method", "list_policies").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListPolicies(ctx, token, p)
}

func (ms *metricsMiddleware) DeletePolicy(ctx context.Context, token string, p policies.Policy) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_policy").Add(1)
		ms.latency.With("method", "delete_policy").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeletePolicy(ctx, token, p)
}

func (ms *metricsMiddleware) Authorize(ctx context.Context, ar policies.AccessRequest) (policy policies.Policy, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "authorize").Add(1)
		ms.latency.With("method", "authorize").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Authorize(ctx, ar)
}
