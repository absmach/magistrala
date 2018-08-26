//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/normalizer"
)

var _ normalizer.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     normalizer.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc normalizer.Service, counter metrics.Counter, latency metrics.Histogram) normalizer.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) Normalize(msg mainflux.RawMessage) (normalizer.NormalizedData, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "normalize").Add(1)
		mm.latency.With("method", "normalize").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Normalize(msg)
}
