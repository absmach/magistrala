//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package api

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/readers"
)

var _ readers.MessageRepository = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     readers.MessageRepository
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc readers.MessageRepository, counter metrics.Counter, latency metrics.Histogram) readers.MessageRepository {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) ReadAll(chanID string, offset, limit uint64, query map[string]string) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "read_all").Add(1)
		mm.latency.With("method", "read_all").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ReadAll(chanID, offset, limit, query)
}
