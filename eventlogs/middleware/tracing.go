// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/eventlogs"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ eventlogs.Service = (*tracing)(nil)

type tracing struct {
	tracer trace.Tracer
	svc    eventlogs.Service
}

func Tracing(svc eventlogs.Service, tracer trace.Tracer) eventlogs.Service {
	return &tracing{tracer, svc}
}

func (tm *tracing) ReadAll(ctx context.Context, token string, page eventlogs.Page) (eventlogs.EventsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "read_all", trace.WithAttributes(
		attribute.String("id", page.ID),
		attribute.String("entity_type", page.EntityType),
	))
	defer span.End()

	return tm.svc.ReadAll(ctx, token, page)
}
