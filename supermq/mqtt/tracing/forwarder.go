// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"fmt"

	"github.com/absmach/supermq/mqtt"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/server"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const forwardOP = "process"

var _ mqtt.Forwarder = (*forwarderMiddleware)(nil)

type forwarderMiddleware struct {
	topic     string
	forwarder mqtt.Forwarder
	tracer    trace.Tracer
	host      server.Config
}

// New creates new mqtt forwarder tracing middleware.
func New(config server.Config, tracer trace.Tracer, forwarder mqtt.Forwarder, topic string) mqtt.Forwarder {
	return &forwarderMiddleware{
		forwarder: forwarder,
		tracer:    tracer,
		topic:     topic,
		host:      config,
	}
}

// Forward traces mqtt forward operations.
func (fm *forwarderMiddleware) Forward(ctx context.Context, id string, sub messaging.Subscriber, pub messaging.Publisher) error {
	subject := fmt.Sprintf("channels.%s.messages", fm.topic)
	spanName := fmt.Sprintf("%s %s", subject, forwardOP)

	ctx, span := fm.tracer.Start(ctx,
		spanName,
		trace.WithAttributes(
			attribute.String("messaging.system", "mqtt"),
			attribute.Bool("messaging.destination.anonymous", false),
			attribute.String("messaging.destination.template", "channels/{channelID}/messages/*"),
			attribute.Bool("messaging.destination.temporary", true),
			attribute.String("network.protocol.name", "mqtt"),
			attribute.String("network.protocol.version", "3.1.1"),
			attribute.String("network.transport", "tcp"),
			attribute.String("network.type", "ipv4"),
			attribute.String("messaging.operation", forwardOP),
			attribute.String("messaging.client_id", id),
			attribute.String("server.address", fm.host.Host),
			attribute.String("server.socket.port", fm.host.Port),
		),
	)
	defer span.End()

	return fm.forwarder.Forward(ctx, id, sub, pub)
}
