//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/things"
)

var _ things.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     things.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc things.Service, counter metrics.Counter, latency metrics.Histogram) things.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) AddThing(ctx context.Context, token string, thing things.Thing) (things.Thing, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_thing").Add(1)
		ms.latency.With("method", "add_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AddThing(ctx, token, thing)
}

func (ms *metricsMiddleware) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_thing").Add(1)
		ms.latency.With("method", "update_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateThing(ctx, token, thing)
}

func (ms *metricsMiddleware) UpdateKey(ctx context.Context, token, id, key string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_key").Add(1)
		ms.latency.With("method", "update_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateKey(ctx, token, id, key)
}

func (ms *metricsMiddleware) ViewThing(ctx context.Context, token, id string) (things.Thing, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_thing").Add(1)
		ms.latency.With("method", "view_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewThing(ctx, token, id)
}

func (ms *metricsMiddleware) ListThings(ctx context.Context, token string, offset, limit uint64, name string) (things.ThingsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_things").Add(1)
		ms.latency.With("method", "list_things").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThings(ctx, token, offset, limit, name)
}

func (ms *metricsMiddleware) ListThingsByChannel(ctx context.Context, token, id string, offset, limit uint64) (things.ThingsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_things_by_channel").Add(1)
		ms.latency.With("method", "list_things_by_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThingsByChannel(ctx, token, id, offset, limit)
}

func (ms *metricsMiddleware) RemoveThing(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_thing").Add(1)
		ms.latency.With("method", "remove_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveThing(ctx, token, id)
}

func (ms *metricsMiddleware) CreateChannel(ctx context.Context, token string, channel things.Channel) (things.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_channel").Add(1)
		ms.latency.With("method", "create_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateChannel(ctx, token, channel)
}

func (ms *metricsMiddleware) UpdateChannel(ctx context.Context, token string, channel things.Channel) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel").Add(1)
		ms.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateChannel(ctx, token, channel)
}

func (ms *metricsMiddleware) ViewChannel(ctx context.Context, token, id string) (things.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_channel").Add(1)
		ms.latency.With("method", "view_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewChannel(ctx, token, id)
}

func (ms *metricsMiddleware) ListChannels(ctx context.Context, token string, offset, limit uint64, name string) (things.ChannelsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels").Add(1)
		ms.latency.With("method", "list_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListChannels(ctx, token, offset, limit, name)
}

func (ms *metricsMiddleware) ListChannelsByThing(ctx context.Context, token, id string, offset, limit uint64) (things.ChannelsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels_by_thing").Add(1)
		ms.latency.With("method", "list_channels_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListChannelsByThing(ctx, token, id, offset, limit)
}

func (ms *metricsMiddleware) RemoveChannel(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_channel").Add(1)
		ms.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveChannel(ctx, token, id)
}

func (ms *metricsMiddleware) Connect(ctx context.Context, token, chanID, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "connect").Add(1)
		ms.latency.With("method", "connect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Connect(ctx, token, chanID, thingID)
}

func (ms *metricsMiddleware) Disconnect(ctx context.Context, token, chanID, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "disconnect").Add(1)
		ms.latency.With("method", "disconnect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Disconnect(ctx, token, chanID, thingID)
}

func (ms *metricsMiddleware) CanAccess(ctx context.Context, id, key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "can_access").Add(1)
		ms.latency.With("method", "can_access").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CanAccess(ctx, id, key)
}

func (ms *metricsMiddleware) CanAccessByID(ctx context.Context, chanID, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "can_access_by_id").Add(1)
		ms.latency.With("method", "can_access_by_id").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CanAccessByID(ctx, chanID, thingID)
}

func (ms *metricsMiddleware) Identify(ctx context.Context, key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identify(ctx, key)
}
