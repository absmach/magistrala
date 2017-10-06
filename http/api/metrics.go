package api

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/writer"
)

var _ http.Service = (*metricService)(nil)

type metricService struct {
	counter metrics.Counter
	latency metrics.Histogram
	http.Service
}

// NewMetricService instruments adapter by tracking request count and latency.
func NewMetricService(counter metrics.Counter, latency metrics.Histogram, s http.Service) http.Service {
	return &metricService{
		counter: counter,
		latency: latency,
		Service: s,
	}
}

func (ms *metricService) Publish(msg writer.RawMessage) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "publish").Add(1)
		ms.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.Publish(msg)
}
