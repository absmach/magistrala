// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/twins"
)

var _ twins.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     twins.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc twins.Service, counter metrics.Counter, latency metrics.Histogram) twins.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) AddTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) (saved twins.Twin, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_twin").Add(1)
		ms.latency.With("method", "add_twin").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AddTwin(ctx, token, twin, def)
}

func (ms *metricsMiddleware) UpdateTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_twin").Add(1)
		ms.latency.With("method", "update_twin").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateTwin(ctx, token, twin, def)
}

func (ms *metricsMiddleware) ViewTwin(ctx context.Context, token, twinID string) (tw twins.Twin, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_twin").Add(1)
		ms.latency.With("method", "view_twin").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewTwin(ctx, token, twinID)
}

func (ms *metricsMiddleware) ListTwins(ctx context.Context, token string, offset uint64, limit uint64, name string, metadata twins.Metadata) (page twins.Page, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_twins").Add(1)
		ms.latency.With("method", "list_twins").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListTwins(ctx, token, offset, limit, name, metadata)
}

func (ms *metricsMiddleware) SaveStates(msg *messaging.Message) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "save_states").Add(1)
		ms.latency.With("method", "save_states").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.SaveStates(msg)
}

func (ms *metricsMiddleware) ListStates(ctx context.Context, token string, offset uint64, limit uint64, twinID string) (st twins.StatesPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_states").Add(1)
		ms.latency.With("method", "list_states").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListStates(ctx, token, offset, limit, twinID)
}

func (ms *metricsMiddleware) RemoveTwin(ctx context.Context, token, twinID string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_twin").Add(1)
		ms.latency.With("method", "remove_twin").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveTwin(ctx, token, twinID)
}
