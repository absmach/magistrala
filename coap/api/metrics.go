// +build !test

package api

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/nats"
)

var _ coap.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     coap.Service
}

// MetricsMiddleware instruments adapter by tracking request count and latency.
func MetricsMiddleware(svc coap.Service, counter metrics.Counter, latency metrics.Histogram) coap.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) Publish(msg mainflux.RawMessage) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Publish(msg)
}

func (mm *metricsMiddleware) Subscribe(chanID, clientID string, channel nats.Channel) error {
	return mm.svc.Subscribe(chanID, clientID, channel)
}

func (mm *metricsMiddleware) SetTimeout(clientID string, timer *time.Timer, duration int) (chan bool, error) {
	return mm.svc.SetTimeout(clientID, timer, duration)
}

func (mm *metricsMiddleware) RemoveTimeout(clientID string) {
	mm.svc.RemoveTimeout(clientID)
}

func (mm *metricsMiddleware) Unsubscribe(clientID string) {
	mm.svc.Unsubscribe(clientID)
}
