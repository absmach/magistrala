// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/coap"
	"github.com/absmach/magistrala/pkg/messaging"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ coap.Service = (*tracingServiceMiddleware)(nil)

// Operation names for tracing CoAP operations.
const (
	publishOP           = "publish_op"
	subscribeOP         = "subscribe_op"
	unsubscribeOP       = "unsubscribe_op"
	disconnectHandlerOp = "disconnect_handler_op"
)

// tracingServiceMiddleware is a middleware implementation for tracing CoAP service operations using OpenTelemetry.
type tracingServiceMiddleware struct {
	tracer trace.Tracer
	svc    coap.Service
}

// New creates a new instance of TracingServiceMiddleware that wraps an existing CoAP service with tracing capabilities.
func New(tracer trace.Tracer, svc coap.Service) coap.Service {
	return &tracingServiceMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

// Publish traces a CoAP publish operation.
func (tm *tracingServiceMiddleware) Publish(ctx context.Context, key string, msg *messaging.Message) error {
	ctx, span := tm.tracer.Start(ctx, publishOP)
	defer span.End()
	return tm.svc.Publish(ctx, key, msg)
}

// Subscribe traces a CoAP subscribe operation.
func (tm *tracingServiceMiddleware) Subscribe(ctx context.Context, key, chanID, subtopic string, c coap.Client) error {
	ctx, span := tm.tracer.Start(ctx, subscribeOP, trace.WithAttributes(
		attribute.String("channel_id", chanID),
		attribute.String("subtopic", subtopic),
	))
	defer span.End()
	return tm.svc.Subscribe(ctx, key, chanID, subtopic, c)
}

// Unsubscribe traces a CoAP unsubscribe operation.
func (tm *tracingServiceMiddleware) Unsubscribe(ctx context.Context, key, chanID, subptopic, token string) error {
	ctx, span := tm.tracer.Start(ctx, unsubscribeOP, trace.WithAttributes(
		attribute.String("channel_id", chanID),
		attribute.String("subtopic", subptopic),
	))
	defer span.End()
	return tm.svc.Unsubscribe(ctx, key, chanID, subptopic, token)
}

// DisconnectHandler traces a CoAP disconnect operation.
func (tm *tracingServiceMiddleware) DisconnectHandler(ctx context.Context, chanID, subptopic, token string) error {
	ctx, span := tm.tracer.Start(ctx, disconnectHandlerOp, trace.WithAttributes(
		attribute.String("channel_id", chanID),
		attribute.String("subtopic", subptopic),
	))
	defer span.End()
	return tm.svc.DisconnectHandler(ctx, chanID, subptopic, token)
}
