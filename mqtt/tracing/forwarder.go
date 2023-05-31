package tracing

import (
	"context"

	"github.com/mainflux/mainflux/mqtt"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/opentracing/opentracing-go"
)

const forwardOP = "forward_op"

var _ mqtt.Forwarder = (*forwarderMiddleware)(nil)

type forwarderMiddleware struct {
	topic     string
	forwarder mqtt.Forwarder
	tracer    opentracing.Tracer
}

// New creates new mqtt forwarder tracing middleware.
func New(tracer opentracing.Tracer, forwarder mqtt.Forwarder, topic string) mqtt.Forwarder {
	return &forwarderMiddleware{
		forwarder: forwarder,
		tracer:    tracer,
		topic:     topic,
	}
}

// Forward traces mqtt forward operations
func (fm *forwarderMiddleware) Forward(ctx context.Context, id string, sub messaging.Subscriber, pub messaging.Publisher) error {
	span, ctx := opentracing.StartSpanFromContextWithTracer(ctx, fm.tracer, forwardOP)
	defer span.Finish()
	span.SetTag("subscriber", id)
	span.SetTag("topic", fm.topic)
	return fm.forwarder.Forward(ctx, id, sub, pub)
}
