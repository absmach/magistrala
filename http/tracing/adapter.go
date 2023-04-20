package tracing

import (
	"context"

	"github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/opentracing/opentracing-go"
)

var _ http.Service = (*serviceMiddleware)(nil)

const publishOP = "publishOP"

// serviceMiddleware implements the http.Service interface, providing a middleware layer for tracing HTTP requests.
// It creates a new span for each request and sets it as the active span in the OpenTracing context.
type serviceMiddleware struct {
	tracer opentracing.Tracer
	svc    http.Service
}

// New creates a new instance of the http.Service interface with tracing middleware.
func New(tracer opentracing.Tracer, svc http.Service) http.Service {
	return &serviceMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

// Publish traces HTTP publish operations.
// It starts a new span as a child of the incoming span (if there is one) and sets it as the active span in the context.
func (sm *serviceMiddleware) Publish(ctx context.Context, token string, msg *messaging.Message) error {
	var spanCtx opentracing.SpanContext = nil
	if httpSpan := opentracing.SpanFromContext(ctx); httpSpan != nil {
		spanCtx = httpSpan.Context()
	}
	span := sm.tracer.StartSpan(publishOP, opentracing.ChildOf(spanCtx))
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	return sm.svc.Publish(ctx, token, msg)
}
