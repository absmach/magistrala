// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/ws"
	"go.opentelemetry.io/otel/trace"
)

var _ ws.Service = (*tracingMiddleware)(nil)

const (
	publishOP     = "publish_op"
	subscribeOP   = "subscribe_op"
	unsubscribeOP = "unsubscribe_op"
)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    ws.Service
}

// TracingMiddleware returns a new websocket service with tracing capabilities.
func TracingMiddleware(tracer trace.Tracer, svc ws.Service) ws.Service {
	return &tracingMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

// Subscribe traces the "Subscribe" operation of the wrapped ws.Service.
func (tm *tracingMiddleware) Subscribe(ctx context.Context, thingKey, chanID, subtopic string, client *ws.Client) error {
	ctx, span := tm.tracer.Start(ctx, subscribeOP)
	defer span.End()

	return tm.svc.Subscribe(ctx, thingKey, chanID, subtopic, client)
}
