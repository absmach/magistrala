// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala/consumers"
	"github.com/absmach/magistrala/internal/server"
	mgjson "github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	consumeBlockingOP = "retrieve_blocking" // This is not specified in the open telemetry spec.
	consumeAsyncOP    = "retrieve_async"    // This is not specified in the open telemetry spec.
)

var defaultAttributes = []attribute.KeyValue{
	attribute.String("messaging.system", "nats"),
	attribute.Bool("messaging.destination.anonymous", false),
	attribute.String("messaging.destination.template", "channels/{channelID}/messages/*"),
	attribute.Bool("messaging.destination.temporary", true),
	attribute.String("network.protocol.name", "nats"),
	attribute.String("network.protocol.version", "2.2.4"),
	attribute.String("network.transport", "tcp"),
	attribute.String("network.type", "ipv4"),
}

var (
	_ consumers.AsyncConsumer    = (*tracingMiddlewareAsync)(nil)
	_ consumers.BlockingConsumer = (*tracingMiddlewareBlock)(nil)
)

type tracingMiddlewareAsync struct {
	consumer consumers.AsyncConsumer
	tracer   trace.Tracer
	host     server.Config
}
type tracingMiddlewareBlock struct {
	consumer consumers.BlockingConsumer
	tracer   trace.Tracer
	host     server.Config
}

// NewAsync creates a new traced consumers.AsyncConsumer service.
func NewAsync(tracer trace.Tracer, consumerAsync consumers.AsyncConsumer, host server.Config) consumers.AsyncConsumer {
	return &tracingMiddlewareAsync{
		consumer: consumerAsync,
		tracer:   tracer,
		host:     host,
	}
}

// NewBlocking creates a new traced consumers.BlockingConsumer service.
func NewBlocking(tracer trace.Tracer, consumerBlock consumers.BlockingConsumer, host server.Config) consumers.BlockingConsumer {
	return &tracingMiddlewareBlock{
		consumer: consumerBlock,
		tracer:   tracer,
		host:     host,
	}
}

// ConsumeBlocking  traces consume operations for message/s consumed.
func (tm *tracingMiddlewareBlock) ConsumeBlocking(ctx context.Context, messages interface{}) error {
	var span trace.Span
	switch m := messages.(type) {
	case mgjson.Messages:
		if len(m.Data) > 0 {
			firstMsg := m.Data[0]
			ctx, span = createSpan(ctx, consumeBlockingOP, firstMsg.Publisher, firstMsg.Channel, firstMsg.Subtopic, len(m.Data), tm.host, trace.SpanKindConsumer, tm.tracer)
			defer span.End()
		}
	case []senml.Message:
		if len(m) > 0 {
			firstMsg := m[0]
			ctx, span = createSpan(ctx, consumeBlockingOP, firstMsg.Publisher, firstMsg.Channel, firstMsg.Subtopic, len(m), tm.host, trace.SpanKindConsumer, tm.tracer)
			defer span.End()
		}
	}
	return tm.consumer.ConsumeBlocking(ctx, messages)
}

// ConsumeAsync traces consume operations for message/s consumed.
func (tm *tracingMiddlewareAsync) ConsumeAsync(ctx context.Context, messages interface{}) {
	var span trace.Span
	switch m := messages.(type) {
	case mgjson.Messages:
		if len(m.Data) > 0 {
			firstMsg := m.Data[0]
			ctx, span = createSpan(ctx, consumeAsyncOP, firstMsg.Publisher, firstMsg.Channel, firstMsg.Subtopic, len(m.Data), tm.host, trace.SpanKindConsumer, tm.tracer)
			defer span.End()
		}
	case []senml.Message:
		if len(m) > 0 {
			firstMsg := m[0]
			ctx, span = createSpan(ctx, consumeAsyncOP, firstMsg.Publisher, firstMsg.Channel, firstMsg.Subtopic, len(m), tm.host, trace.SpanKindConsumer, tm.tracer)
			defer span.End()
		}
	}
	tm.consumer.ConsumeAsync(ctx, messages)
}

// Errors traces async consume errors.
func (tm *tracingMiddlewareAsync) Errors() <-chan error {
	return tm.consumer.Errors()
}

func createSpan(ctx context.Context, operation, clientID, topic, subTopic string, noMessages int, cfg server.Config, spanKind trace.SpanKind, tracer trace.Tracer) (context.Context, trace.Span) {
	subject := fmt.Sprintf("channels.%s.messages", topic)
	if subTopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subTopic)
	}
	spanName := fmt.Sprintf("%s %s", subject, operation)

	kvOpts := []attribute.KeyValue{
		attribute.String("messaging.operation", operation),
		attribute.String("messaging.client_id", clientID),
		attribute.String("messaging.destination.name", subject),
		attribute.String("server.address", cfg.Host),
		attribute.String("server.socket.port", cfg.Port),
		attribute.Int("messaging.batch.message_count", noMessages),
	}

	kvOpts = append(kvOpts, defaultAttributes...)

	return tracer.Start(ctx, spanName, trace.WithAttributes(kvOpts...), trace.WithSpanKind(spanKind))
}
