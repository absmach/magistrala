// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/mainflux/mainflux/mqtt"
	"github.com/mainflux/mainflux/pkg/messaging"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const forwardOP = "forward_op"

var _ mqtt.Forwarder = (*forwarderMiddleware)(nil)

type forwarderMiddleware struct {
	topic     string
	forwarder mqtt.Forwarder
	tracer    trace.Tracer
}

// New creates new mqtt forwarder tracing middleware.
func New(tracer trace.Tracer, forwarder mqtt.Forwarder, topic string) mqtt.Forwarder {
	return &forwarderMiddleware{
		forwarder: forwarder,
		tracer:    tracer,
		topic:     topic,
	}
}

// Forward traces mqtt forward operations.
func (fm *forwarderMiddleware) Forward(ctx context.Context, id string, sub messaging.Subscriber, pub messaging.Publisher) error {
	ctx, span := fm.tracer.Start(ctx,
		forwardOP,
		trace.WithAttributes(
			attribute.String("topic", fm.topic),
			attribute.String("subscriber", id),
		),
	)
	defer span.End()

	return fm.forwarder.Forward(ctx, id, sub, pub)
}
