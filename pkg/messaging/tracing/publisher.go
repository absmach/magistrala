package tracing

import (
	"context"

	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/opentracing/opentracing-go"
)

// traced ops.
const publishOP = "publish_op"

var _ messaging.Publisher = (*publisherMiddleware)(nil)

type publisherMiddleware struct {
	publisher messaging.Publisher
	tracer    opentracing.Tracer
}

// New creates new messaging publisher tracing middleware.
func New(tracer opentracing.Tracer, publisher messaging.Publisher) messaging.Publisher {
	return &publisherMiddleware{
		publisher: publisher,
		tracer:    tracer,
	}
}

// Publish traces nats publish operations.
func (pm *publisherMiddleware) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	span, ctx := createSpan(ctx, publishOP, topic, msg.Subtopic, msg.Publisher, pm.tracer)
	defer span.Finish()
	return pm.publisher.Publish(ctx, topic, msg)
}

// Close nats trace publisher middleware
func (pm *publisherMiddleware) Close() error {
	return pm.publisher.Close()
}

func createSpan(ctx context.Context, operation, topic, subTopic, thingID string, tracer opentracing.Tracer) (opentracing.Span, context.Context) {
	span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, operation)
	switch operation {
	case publishOP:
		span.SetTag("publisher", thingID)
	default:
		span.SetTag("subscriber", thingID)
	}
	span.SetTag("topic", topic)
	if subTopic != "" {
		span.SetTag("sub-topic", subTopic)
	}
	return span, ctx
}
