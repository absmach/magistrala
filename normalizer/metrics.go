package normalizer

import (
	"time"

	"github.com/go-kit/kit/metrics"
	nats "github.com/nats-io/go-nats"
)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	ef      eventFlow
}

func newMetricsMiddleware(ef eventFlow, counter metrics.Counter, latency metrics.Histogram) *metricsMiddleware {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		ef:      ef,
	}
}

func (mm *metricsMiddleware) handleMessage(msg *nats.Msg) {
	defer func(begin time.Time) {
		mm.counter.With("method", "handleMessage").Add(1)
		mm.latency.With("method", "handleMessage").Observe(time.Since(begin).Seconds())
	}(time.Now())
	mm.ef.handleMsg(msg)
}
