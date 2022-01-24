// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/certs"
)

var _ certs.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     certs.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc certs.Service, counter metrics.Counter, latency metrics.Histogram) certs.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) IssueCert(ctx context.Context, token, thingID string, ttl string, keyBits int, keyType string) (certs.Cert, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "issue_cert").Add(1)
		ms.latency.With("method", "issue_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.IssueCert(ctx, token, thingID, ttl, keyBits, keyType)
}

func (ms *metricsMiddleware) ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (certs.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_certs").Add(1)
		ms.latency.With("method", "list_certs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListCerts(ctx, token, thingID, offset, limit)
}

func (ms *metricsMiddleware) ListSerials(ctx context.Context, token, thingID string, offset, limit uint64) (certs.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_serials").Add(1)
		ms.latency.With("method", "list_serials").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListSerials(ctx, token, thingID, offset, limit)
}

func (ms *metricsMiddleware) ViewCert(ctx context.Context, token, serialID string) (certs.Cert, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_cert").Add(1)
		ms.latency.With("method", "view_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewCert(ctx, token, serialID)
}

func (ms *metricsMiddleware) RevokeCert(ctx context.Context, token, thingID string) (certs.Revoke, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke_cert").Add(1)
		ms.latency.With("method", "revoke_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RevokeCert(ctx, token, thingID)
}
