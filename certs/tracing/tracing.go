// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/certs"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ certs.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    certs.Service
}

// New returns a new certs service with tracing capabilities.
func New(svc certs.Service, tracer trace.Tracer) certs.Service {
	return &tracingMiddleware{tracer, svc}
}

// IssueCert traces the "IssueCert" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) IssueCert(ctx context.Context, domainID, token, clientID, ttl string) (certs.Cert, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_create_group", trace.WithAttributes(
		attribute.String("client_id", clientID),
		attribute.String("ttl", ttl),
	))
	defer span.End()

	return tm.svc.IssueCert(ctx, domainID, token, clientID, ttl)
}

// ListCerts traces the "ListCerts" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) ListCerts(ctx context.Context, clientID string, pm certs.PageMetadata) (certs.CertPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_certs", trace.WithAttributes(
		attribute.String("client_id", clientID),
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
	))
	defer span.End()

	return tm.svc.ListCerts(ctx, clientID, pm)
}

// ListSerials traces the "ListSerials" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) ListSerials(ctx context.Context, clientID string, pm certs.PageMetadata) (certs.CertPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_serials", trace.WithAttributes(
		attribute.String("client_id", clientID),
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
	))
	defer span.End()

	return tm.svc.ListSerials(ctx, clientID, pm)
}

// ViewCert traces the "ViewCert" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) ViewCert(ctx context.Context, serialID string) (certs.Cert, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_cert", trace.WithAttributes(
		attribute.String("serial_id", serialID),
	))
	defer span.End()

	return tm.svc.ViewCert(ctx, serialID)
}

// RevokeCert traces the "RevokeCert" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) RevokeCert(ctx context.Context, domainID, token, serialID string) (certs.Revoke, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_revoke_cert", trace.WithAttributes(
		attribute.String("serial_id", serialID),
	))
	defer span.End()

	return tm.svc.RevokeCert(ctx, domainID, token, serialID)
}
