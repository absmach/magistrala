// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/authn"
)

var _ authn.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     authn.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc authn.Service, counter metrics.Counter, latency metrics.Histogram) authn.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) Issue(ctx context.Context, token string, key authn.Key) (authn.Key, string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "issue").Add(1)
		ms.latency.With("method", "issue").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Issue(ctx, token, key)
}

func (ms *metricsMiddleware) Revoke(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke").Add(1)
		ms.latency.With("method", "revoke").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Revoke(ctx, token, id)
}

func (ms *metricsMiddleware) Retrieve(ctx context.Context, token, id string) (authn.Key, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "retrieve").Add(1)
		ms.latency.With("method", "retrieve").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Retrieve(ctx, token, id)
}

func (ms *metricsMiddleware) Identify(ctx context.Context, token string) (authn.Identity, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identify(ctx, token)
}
