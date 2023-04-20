package tracing

import (
	"context"

	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/opentracing/opentracing-go"
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
	tracer opentracing.Tracer
}

// NewPubSub creates a new pubsub middleware that traces pubsub operations.
func NewPubSub(tracer opentracing.Tracer, pubsub messaging.PubSub) messaging.PubSub {
	return &pubsubMiddleware{
		publisherMiddleware: publisherMiddleware{
			publisher: pubsub,
			tracer:    tracer,
		},
		pubsub: pubsub,
		tracer: tracer,
	}
}

// Subscribe creates a new subscription and traces the operation.
func (pm *pubsubMiddleware) Subscribe(ctx context.Context, id string, topic string, handler messaging.MessageHandler) error {
	span := createSpan(ctx, subscribeOP, topic, "", id, pm.tracer)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	h := &traceHandler{
		handler: handler,
		tracer:  pm.tracer,
		ctx:     ctx,
	}
	return pm.pubsub.Subscribe(ctx, id, topic, h)
}

// Unsubscribe removes an existing subscription and traces the operation.
func (pm *pubsubMiddleware) Unsubscribe(ctx context.Context, id string, topic string) error {
	span := createSpan(ctx, unsubscribeOp, topic, "", id, pm.tracer)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	return pm.pubsub.Unsubscribe(ctx, id, topic)
}

// traceHandler is used to trace the message handling operation
type traceHandler struct {
	handler messaging.MessageHandler
	tracer  opentracing.Tracer
	ctx     context.Context
	topic   string
}

// Handle instruments the message handling operation
func (h *traceHandler) Handle(msg *messaging.Message) error {
	span := createSpan(h.ctx, handleOp, h.topic, msg.Subtopic, msg.Publisher, h.tracer)
	defer span.Finish()
	return h.handler.Handle(msg)
}

// Cancel cancels the message handling operation
func (h *traceHandler) Cancel() error {
	return h.handler.Cancel()
}
