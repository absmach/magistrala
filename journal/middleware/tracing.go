// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/journal"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	smqTracing "github.com/absmach/supermq/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ journal.Service = (*tracing)(nil)

type tracing struct {
	tracer trace.Tracer
	svc    journal.Service
}

func Tracing(svc journal.Service, tracer trace.Tracer) journal.Service {
	return &tracing{tracer, svc}
}

func (tm *tracing) Save(ctx context.Context, j journal.Journal) error {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "save", trace.WithAttributes(
		attribute.String("occurred_at", j.OccurredAt.String()),
		attribute.String("operation", j.Operation),
	))
	defer span.End()

	return tm.svc.Save(ctx, j)
}

func (tm *tracing) RetrieveAll(ctx context.Context, session smqauthn.Session, page journal.Page) (resp journal.JournalsPage, err error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "retrieve_all", trace.WithAttributes(
		attribute.Int64("offset", int64(page.Offset)),
		attribute.Int64("limit", int64(page.Limit)),
		attribute.Int64("total", int64(resp.Total)),
		attribute.String("entity_type", page.EntityType.String()),
		attribute.String("operation", page.Operation),
	))
	defer span.End()

	return tm.svc.RetrieveAll(ctx, session, page)
}

func (tm *tracing) RetrieveClientTelemetry(ctx context.Context, session smqauthn.Session, clientID string) (j journal.ClientTelemetry, err error) {
	ctx, span := smqTracing.StartSpan(ctx, tm.tracer, "retrieve", trace.WithAttributes(
		attribute.String("client_id", clientID),
		attribute.String("domain_id", session.DomainID),
	))
	defer span.End()

	return tm.svc.RetrieveClientTelemetry(ctx, session, clientID)
}
