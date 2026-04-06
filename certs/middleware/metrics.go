// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/pkg/authn"
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

func (mm *metricsMiddleware) RenewCert(ctx context.Context, session authn.Session, serialNumber string) (certs.Certificate, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "renew_certificate").Add(1)
		mm.latency.With("method", "renew_certificate").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.RenewCert(ctx, session, serialNumber)
}

func (mm *metricsMiddleware) IssueCert(ctx context.Context, session authn.Session, entityID, ttl string, ipAddrs []string, options certs.SubjectOptions) (certs.Certificate, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "issue_certificate").Add(1)
		mm.latency.With("method", "issue_certificate").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.IssueCert(ctx, session, entityID, ttl, ipAddrs, options)
}

func (mm *metricsMiddleware) ListCerts(ctx context.Context, session authn.Session, pm certs.PageMetadata) (certs.CertificatePage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_certificates").Add(1)
		mm.latency.With("method", "list_certificates").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.ListCerts(ctx, session, pm)
}

func (mm *metricsMiddleware) RevokeBySerial(ctx context.Context, session authn.Session, serialNumber string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "revoke_by_serial").Add(1)
		mm.latency.With("method", "revoke_by_serial").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.RevokeBySerial(ctx, session, serialNumber)
}

func (mm *metricsMiddleware) RevokeAll(ctx context.Context, session authn.Session, entityId string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "revoke_all").Add(1)
		mm.latency.With("method", "revoke_all").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.RevokeAll(ctx, session, entityId)
}

func (mm *metricsMiddleware) ViewCert(ctx context.Context, session authn.Session, serialNumber string) (certs.Certificate, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_certificate").Add(1)
		mm.latency.With("method", "view_certificate").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ViewCert(ctx, session, serialNumber)
}

func (mm *metricsMiddleware) OCSP(ctx context.Context, serialNumber string, ocspRequestDER []byte) ([]byte, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "ocsp").Add(1)
		mm.latency.With("method", "ocsp").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.OCSP(ctx, serialNumber, ocspRequestDER)
}

func (mm *metricsMiddleware) GetEntityID(ctx context.Context, serialNumber string) (string, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "get_entity_id").Add(1)
		mm.latency.With("method", "get_entity_id").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.GetEntityID(ctx, serialNumber)
}

func (mm *metricsMiddleware) GenerateCRL(ctx context.Context) ([]byte, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "generate_crl").Add(1)
		mm.latency.With("method", "generate_crl").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.GenerateCRL(ctx)
}

func (mm *metricsMiddleware) RetrieveCAChain(ctx context.Context) (certs.Certificate, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "retrieve_ca_chain").Add(1)
		mm.latency.With("method", "retrieve_ca_chain").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.RetrieveCAChain(ctx)
}

func (mm *metricsMiddleware) IssueFromCSR(ctx context.Context, session authn.Session, entityID, ttl string, csr certs.CSR) (certs.Certificate, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "issue_from_csr").Add(1)
		mm.latency.With("method", "issue_from_csr").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.IssueFromCSR(ctx, session, entityID, ttl, csr)
}

func (mm *metricsMiddleware) IssueFromCSRInternal(ctx context.Context, entityID, ttl string, csr certs.CSR) (certs.Certificate, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "issue_from_csr_internal").Add(1)
		mm.latency.With("method", "issue_from_csr_internal").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.IssueFromCSRInternal(ctx, entityID, ttl, csr)
}
