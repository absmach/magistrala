// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/pkg/messaging"
)

var _ http.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     http.Service
}

// MetricsMiddleware instruments adapter by tracking request count and latency.
func MetricsMiddleware(svc http.Service, counter metrics.Counter, latency metrics.Histogram) http.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) Publish(ctx context.Context, token string, msg messaging.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Publish(ctx, token, msg)
}
