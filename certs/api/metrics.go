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

func (ms *metricsMiddleware) IssueCert(ctx context.Context, token, thingID, name, ttl string) (certs.Cert, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "issue_cert").Add(1)
		ms.latency.With("method", "issue_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.IssueCert(ctx, token, thingID, name, ttl)
}

func (ms *metricsMiddleware) ListCerts(ctx context.Context, token, certID, thingID, serial, name string, status certs.Status, offset, limit uint64) (certs.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_certs").Add(1)
		ms.latency.With("method", "list_certs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListCerts(ctx, token, certID, thingID, serial, name, status, offset, limit)
}

func (ms *metricsMiddleware) ViewCert(ctx context.Context, token, serialID string) (certs.Cert, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_cert").Add(1)
		ms.latency.With("method", "view_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewCert(ctx, token, serialID)
}

func (ms *metricsMiddleware) RevokeCert(ctx context.Context, token, certID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke_cert").Add(1)
		ms.latency.With("method", "revoke_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RevokeCert(ctx, token, certID)
}

func (ms *metricsMiddleware) RenewCert(ctx context.Context, token, certID string) (certs.Cert, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "renew_cert").Add(1)
		ms.latency.With("method", "renew_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RenewCert(ctx, token, certID)
}

func (ms *metricsMiddleware) RemoveCert(ctx context.Context, token, certID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_cert").Add(1)
		ms.latency.With("method", "remove_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveCert(ctx, token, certID)
}

func (ms *metricsMiddleware) RevokeThingCerts(ctx context.Context, token, thingID string, limit int64) (uint64, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke_thing_cert").Add(1)
		ms.latency.With("method", "revoke_thing_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RevokeThingCerts(ctx, token, thingID, limit)
}

func (ms *metricsMiddleware) RenewThingCerts(ctx context.Context, token, thingID string, limit int64) (uint64, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "renew_cert").Add(1)
		ms.latency.With("method", "renew_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RenewThingCerts(ctx, token, thingID, limit)
}

func (ms *metricsMiddleware) RemoveThingCerts(ctx context.Context, token, thingID string, limit int64) (uint64, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_cert").Add(1)
		ms.latency.With("method", "remove_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveThingCerts(ctx, token, thingID, limit)
}
