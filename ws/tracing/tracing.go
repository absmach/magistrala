// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/ws"
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

// New returns a new websocket service with tracing capabilities.
func New(tracer trace.Tracer, svc ws.Service) ws.Service {
	return &tracingMiddleware{
		tracer: tracer,
		svc:    svc,
	}
}

// Publish traces the "Publish" operation of the wrapped ws.Service.
func (tm *tracingMiddleware) Publish(ctx context.Context, thingKey string, msg *messaging.Message) error {
	ctx, span := tm.tracer.Start(ctx, publishOP)
	defer span.End()

	return tm.svc.Publish(ctx, thingKey, msg)
}

// Subscribe traces the "Subscribe" operation of the wrapped ws.Service.
func (tm *tracingMiddleware) Subscribe(ctx context.Context, thingKey string, chanID string, subtopic string, client *ws.Client) error {
	ctx, span := tm.tracer.Start(ctx, subscribeOP)
	defer span.End()

	return tm.svc.Subscribe(ctx, thingKey, chanID, subtopic, client)
}

// Unsubscribe traces the "Unsubscribe" operation of the wrapped ws.Service.
func (tm *tracingMiddleware) Unsubscribe(ctx context.Context, thingKey string, chanID string, subtopic string) error {
	ctx, span := tm.tracer.Start(ctx, unsubscribeOP)
	defer span.End()

	return tm.svc.Unsubscribe(ctx, thingKey, chanID, subtopic)
}
