// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/absmach/magistrala/certs"
	"github.com/go-kit/kit/metrics"
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

// IssueCert instruments IssueCert method with metrics.
func (ms *metricsMiddleware) IssueCert(ctx context.Context, domainID, token, clientID, ttl string) (certs.Cert, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "issue_cert").Add(1)
		ms.latency.With("method", "issue_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.IssueCert(ctx, domainID, token, clientID, ttl)
}

// ListCerts instruments ListCerts method with metrics.
func (ms *metricsMiddleware) ListCerts(ctx context.Context, clientID string, pm certs.PageMetadata) (certs.CertPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_certs").Add(1)
		ms.latency.With("method", "list_certs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListCerts(ctx, clientID, pm)
}

// ListSerials instruments ListSerials method with metrics.
func (ms *metricsMiddleware) ListSerials(ctx context.Context, clientID string, pm certs.PageMetadata) (certs.CertPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_serials").Add(1)
		ms.latency.With("method", "list_serials").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListSerials(ctx, clientID, pm)
}

// ViewCert instruments ViewCert method with metrics.
func (ms *metricsMiddleware) ViewCert(ctx context.Context, serialID string) (certs.Cert, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_cert").Add(1)
		ms.latency.With("method", "view_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewCert(ctx, serialID)
}

// RevokeCert instruments RevokeCert method with metrics.
func (ms *metricsMiddleware) RevokeCert(ctx context.Context, domainID, token, clientID string) (certs.Revoke, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke_cert").Add(1)
		ms.latency.With("method", "revoke_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RevokeCert(ctx, domainID, token, clientID)
}
