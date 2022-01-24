// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/lora"
)

var _ lora.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     lora.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc lora.Service, counter metrics.Counter, latency metrics.Histogram) lora.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) CreateThing(ctx context.Context, thingID string, loraDevEUI string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "create_thing").Add(1)
		mm.latency.With("method", "create_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.CreateThing(ctx, thingID, loraDevEUI)
}

func (mm *metricsMiddleware) UpdateThing(ctx context.Context, thingID string, loraDevEUI string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_thing").Add(1)
		mm.latency.With("method", "update_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateThing(ctx, thingID, loraDevEUI)
}

func (mm *metricsMiddleware) RemoveThing(ctx context.Context, thingID string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_thing").Add(1)
		mm.latency.With("method", "remove_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveThing(ctx, thingID)
}

func (mm *metricsMiddleware) CreateChannel(ctx context.Context, chanID, loraApp string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "create_channel").Add(1)
		mm.latency.With("method", "create_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.CreateChannel(ctx, chanID, loraApp)
}

func (mm *metricsMiddleware) UpdateChannel(ctx context.Context, chanID, loraApp string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_channel").Add(1)
		mm.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateChannel(ctx, chanID, loraApp)
}

func (mm *metricsMiddleware) RemoveChannel(ctx context.Context, chanID string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_channel").Add(1)
		mm.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveChannel(ctx, chanID)
}

func (mm *metricsMiddleware) ConnectThing(ctx context.Context, chanID, thingID string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "connect_thing").Add(1)
		mm.latency.With("method", "connect_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ConnectThing(ctx, chanID, thingID)
}

func (mm *metricsMiddleware) DisconnectThing(ctx context.Context, chanID, thingID string) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "disconnect_thing").Add(1)
		mm.latency.With("method", "disconnect_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DisconnectThing(ctx, chanID, thingID)
}

func (mm *metricsMiddleware) Publish(ctx context.Context, msg lora.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "publish").Add(1)
		mm.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Publish(ctx, msg)
}
