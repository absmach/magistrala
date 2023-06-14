// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package tracing

import (
	"context"

	"github.com/mainflux/mainflux/pkg/messaging"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Traced operations.
const publishOP = "publish_op"

var _ messaging.Publisher = (*publisherMiddleware)(nil)

type publisherMiddleware struct {
	publisher messaging.Publisher
	tracer    trace.Tracer
}

// New creates new messaging publisher tracing middleware.
func New(tracer trace.Tracer, publisher messaging.Publisher) messaging.Publisher {
	return &publisherMiddleware{
		publisher: publisher,
		tracer:    tracer,
	}
}

// Publish traces NATS publish operations.
func (pm *publisherMiddleware) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	ctx, span := createSpan(ctx, publishOP, topic, msg.Subtopic, msg.Publisher, pm.tracer)
	defer span.End()
	return pm.publisher.Publish(ctx, topic, msg)
}

// Close NATS trace publisher middleware.
func (pm *publisherMiddleware) Close() error {
	return pm.publisher.Close()
}

func createSpan(ctx context.Context, operation, topic, subTopic, thingID string, tracer trace.Tracer) (context.Context, trace.Span) {
	kvOpts := []attribute.KeyValue{}
	switch operation {
	case publishOP:
		kvOpts = append(kvOpts, attribute.String("publisher", thingID))
	default:
		kvOpts = append(kvOpts, attribute.String("subscriber", thingID))
	}
	kvOpts = append(kvOpts, attribute.String("topic", topic))
	if subTopic != "" {
		kvOpts = append(kvOpts, attribute.String("subtopic", topic))
	}
	return tracer.Start(ctx, operation, trace.WithAttributes(kvOpts...))
}
