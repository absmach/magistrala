// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/pkg/messaging"
	"go.opentelemetry.io/otel/trace"
)

var _ http.Service = (*serviceMiddleware)(nil)

const publishOP = "publish"

// serviceMiddleware implements the http.Service interface, providing a middleware layer for tracing HTTP requests.
// It creates a new span for each request and sets it as the active span in the OpenTelemetry context.
type serviceMiddleware struct {
	tracer trace.Tracer
	svc    http.Service
}

// New creates a new instance of the http.Service interface with tracing middleware.
func New(tracer trace.Tracer, svc http.Service) http.Service {
	return &serviceMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

// Publish traces HTTP publish operations.
// It starts a new span as a child of the incoming span (if there is one) and sets it as the active span in the context.
func (sm *serviceMiddleware) Publish(ctx context.Context, token string, msg *messaging.Message) error {
	ctx, span := sm.tracer.Start(ctx, publishOP)
	defer span.End()
	return sm.svc.Publish(ctx, token, msg)
}
