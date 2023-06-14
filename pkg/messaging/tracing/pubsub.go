// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package tracing

import (
	"context"

	"github.com/mainflux/mainflux/pkg/messaging"
	"go.opentelemetry.io/otel/trace"
)

// Constants to define different operations to be traced.
const (
	subscribeOP   = "subscribe_op"
	unsubscribeOp = "unsubscribe_op"
	handleOp      = "handle_op"
)

var _ messaging.PubSub = (*pubsubMiddleware)(nil)

type pubsubMiddleware struct {
	publisherMiddleware
	pubsub messaging.PubSub
}

// NewPubSub creates a new pubsub middleware that traces pubsub operations.
func NewPubSub(tracer trace.Tracer, pubsub messaging.PubSub) messaging.PubSub {
	return &pubsubMiddleware{
		publisherMiddleware: publisherMiddleware{
			publisher: pubsub,
			tracer:    tracer,
		},
		pubsub: pubsub,
	}
}

// Subscribe creates a new subscription and traces the operation.
func (pm *pubsubMiddleware) Subscribe(ctx context.Context, id string, topic string, handler messaging.MessageHandler) error {
	ctx, span := createSpan(ctx, subscribeOP, topic, "", id, pm.tracer)
	defer span.End()
	h := &traceHandler{
		handler: handler,
		tracer:  pm.tracer,
		ctx:     ctx,
	}
	return pm.pubsub.Subscribe(ctx, id, topic, h)
}

// Unsubscribe removes an existing subscription and traces the operation.
func (pm *pubsubMiddleware) Unsubscribe(ctx context.Context, id string, topic string) error {
	ctx, span := createSpan(ctx, unsubscribeOp, topic, "", id, pm.tracer)
	defer span.End()
	return pm.pubsub.Unsubscribe(ctx, id, topic)
}

// TraceHandler is used to trace the message handling operation.
type traceHandler struct {
	handler messaging.MessageHandler
	tracer  trace.Tracer
	ctx     context.Context
	topic   string
}

// Handle instruments the message handling operation.
func (h *traceHandler) Handle(msg *messaging.Message) error {
	_, span := createSpan(h.ctx, handleOp, h.topic, msg.Subtopic, msg.Publisher, h.tracer)
	defer span.End()
	return h.handler.Handle(msg)
}

// Cancel cancels the message handling operation.
func (h *traceHandler) Cancel() error {
	return h.handler.Cancel()
}
