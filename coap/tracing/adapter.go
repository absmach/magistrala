package tracing

import (
	"context"

	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/opentracing/opentracing-go"
)

var _ coap.Service = (*tracingServiceMiddleware)(nil)

// Operation names for tracing CoAP operations.
const (
	publishOP     = "publish_op"
	subscribeOP   = "subscirbe_op"
	unsubscribeOP = "unsubscribe_op"
)

// tracingServiceMiddleware is a middleware implementation for tracing CoAP service operations using OpenTracing.
type tracingServiceMiddleware struct {
	tracer opentracing.Tracer
	svc    coap.Service
}

// New creates a new instance of TracingServiceMiddleware that wraps an existing CoAP service with tracing capabilities.
func New(tracer opentracing.Tracer, svc coap.Service) coap.Service {
	return &tracingServiceMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

// Publish traces a CoAP publish operation.
func (tm *tracingServiceMiddleware) Publish(ctx context.Context, key string, msg *messaging.Message) error {
	span := tm.createSpan(ctx, publishOP)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	return tm.svc.Publish(ctx, key, msg)
}

// Subscribe traces a CoAP subscribe operation.
func (tm *tracingServiceMiddleware) Subscribe(ctx context.Context, key string, chanID string, subtopic string, c coap.Client) error {
	span := tm.createSpan(ctx, subscribeOP)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	return tm.svc.Subscribe(ctx, key, chanID, subtopic, c)
}

// Unsubscribe traces a CoAP unsubscribe operation.
func (tm *tracingServiceMiddleware) Unsubscribe(ctx context.Context, key string, chanID string, subptopic string, token string) error {
	span := tm.createSpan(ctx, unsubscribeOP)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	return tm.svc.Unsubscribe(ctx, key, chanID, subptopic, token)
}

// createSpan creates an OpenTracing span with an operation name and an optional parent span.
func (tm *tracingServiceMiddleware) createSpan(ctx context.Context, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tm.tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}
	return tm.tracer.StartSpan(opName)
}
