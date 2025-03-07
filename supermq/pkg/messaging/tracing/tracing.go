// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package tracing

import (
	"context"
	"fmt"

	"github.com/absmach/supermq/pkg/server"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var defaultAttributes = []attribute.KeyValue{
	attribute.Bool("messaging.destination.anonymous", false),
	attribute.String("messaging.destination.template", "channels/{channelID}/messages/*"),
	attribute.Bool("messaging.destination.temporary", true),
	attribute.String("network.transport", "tcp"),
	attribute.String("network.type", "ipv4"),
}

func CreateSpan(ctx context.Context, operation, clientID, topic, subTopic string, msgSize int, cfg server.Config, spanKind trace.SpanKind, tracer trace.Tracer) (context.Context, trace.Span) {
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
	}

	if msgSize > 0 {
		kvOpts = append(kvOpts, attribute.Int("messaging.message.payload_size_bytes", msgSize))
	}

	kvOpts = append(kvOpts, defaultAttributes...)

	return tracer.Start(ctx, spanName, trace.WithAttributes(kvOpts...), trace.WithSpanKind(spanKind))
}
