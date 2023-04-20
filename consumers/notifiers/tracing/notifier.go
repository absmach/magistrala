package tracing

import (
	notifiers "github.com/mainflux/mainflux/consumers/notifiers"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

const notifierOP = "notifier_op"

var _ notifiers.Notifier = (*serviceMiddleware)(nil)

type serviceMiddleware struct {
	svc    notifiers.Notifier
	tracer opentracing.Tracer
}

// NewNotifier creates a new notifier tracing middleware service.
func NewNotifier(tracer opentracing.Tracer, svc notifiers.Notifier) notifiers.Notifier {
	return &serviceMiddleware{
		svc:    svc,
		tracer: tracer,
	}
}

// Notify traces notify operations.
func (sm *serviceMiddleware) Notify(from string, to []string, msg *messaging.Message) error {
	span := sm.tracer.StartSpan(notifierOP, ext.SpanKindConsumer)
	ext.MessageBusDestination.Set(span, msg.Subtopic)
	defer span.Finish()
	return sm.svc.Notify(from, to, msg)
}
