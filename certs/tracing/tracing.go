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
func (tm *tracingMiddleware) IssueCert(ctx context.Context, token, thingID, ttl string) (certs.Cert, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_create_group", trace.WithAttributes(
		attribute.String("thing_id", thingID),
		attribute.String("ttl", ttl),
	))
	defer span.End()

	return tm.svc.IssueCert(ctx, token, thingID, ttl)
}

// ListCerts traces the "ListCerts" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (certs.Page, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_certs", trace.WithAttributes(
		attribute.String("thing_id", thingID),
		attribute.Int64("offset", int64(offset)),
		attribute.Int64("limit", int64(limit)),
	))
	defer span.End()

	return tm.svc.ListCerts(ctx, token, thingID, offset, limit)
}

// ListSerials traces the "ListSerials" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) ListSerials(ctx context.Context, token, thingID string, offset, limit uint64) (certs.Page, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_serials", trace.WithAttributes(
		attribute.String("thing_id", thingID),
		attribute.Int64("offset", int64(offset)),
		attribute.Int64("limit", int64(limit)),
	))
	defer span.End()

	return tm.svc.ListSerials(ctx, token, thingID, offset, limit)
}

// ViewCert traces the "ViewCert" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) ViewCert(ctx context.Context, token, serialID string) (certs.Cert, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_cert", trace.WithAttributes(
		attribute.String("serial_id", serialID),
	))
	defer span.End()

	return tm.svc.ViewCert(ctx, token, serialID)
}

// RevokeCert traces the "RevokeCert" operation of the wrapped certs.Service.
func (tm *tracingMiddleware) RevokeCert(ctx context.Context, token, serialID string) (certs.Revoke, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_revoke_cert", trace.WithAttributes(
		attribute.String("serial_id", serialID),
	))
	defer span.End()

	return tm.svc.RevokeCert(ctx, token, serialID)
}
