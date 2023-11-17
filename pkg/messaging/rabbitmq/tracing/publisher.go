// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package tracing

import (
	"context"

	"github.com/absmach/magistrala/internal/server"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/messaging/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Traced operations.
const publishOP = "publish"

var defaultAttributes = []attribute.KeyValue{
	attribute.String("messaging.system", "rabbitmq"),
	attribute.String("network.protocol.name", "amqp"),
	attribute.String("network.protocol.version", "3.9.20"),
	attribute.String("messaging.rabbitmq.destination.routing_key", "magistrala"),
}

var _ messaging.Publisher = (*publisherMiddleware)(nil)

type publisherMiddleware struct {
	publisher messaging.Publisher
	tracer    trace.Tracer
	host      server.Config
}

func NewPublisher(config server.Config, tracer trace.Tracer, publisher messaging.Publisher) messaging.Publisher {
	pub := &publisherMiddleware{
		publisher: publisher,
		tracer:    tracer,
		host:      config,
	}

	return pub
}

func (pm *publisherMiddleware) Publish(ctx context.Context, topic string, msg *messaging.Message) error {
	ctx, span := tracing.CreateSpan(ctx, publishOP, msg.Publisher, topic, msg.Subtopic, len(msg.Payload), pm.host, trace.SpanKindClient, pm.tracer)
	defer span.End()

	span.SetAttributes(defaultAttributes...)

	return pm.publisher.Publish(ctx, topic, msg)
}

func (pm *publisherMiddleware) Close() error {
	return pm.publisher.Close()
}
