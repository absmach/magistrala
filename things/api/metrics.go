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

func (ms *metricsMiddleware) AddThing(key string, thing things.Thing) (things.Thing, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_thing").Add(1)
		ms.latency.With("method", "add_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AddThing(key, thing)
}

func (ms *metricsMiddleware) UpdateThing(key string, thing things.Thing) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_thing").Add(1)
		ms.latency.With("method", "update_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateThing(key, thing)
}

func (ms *metricsMiddleware) ViewThing(key, id string) (things.Thing, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_thing").Add(1)
		ms.latency.With("method", "view_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewThing(key, id)
}

func (ms *metricsMiddleware) ListThings(key string, offset, limit uint64) ([]things.Thing, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_things").Add(1)
		ms.latency.With("method", "list_things").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThings(key, offset, limit)
}

func (ms *metricsMiddleware) RemoveThing(key, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_thing").Add(1)
		ms.latency.With("method", "remove_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveThing(key, id)
}

func (ms *metricsMiddleware) CreateChannel(key string, channel things.Channel) (things.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_channel").Add(1)
		ms.latency.With("method", "create_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateChannel(key, channel)
}

func (ms *metricsMiddleware) UpdateChannel(key string, channel things.Channel) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel").Add(1)
		ms.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateChannel(key, channel)
}

func (ms *metricsMiddleware) ViewChannel(key, id string) (things.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_channel").Add(1)
		ms.latency.With("method", "view_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewChannel(key, id)
}

func (ms *metricsMiddleware) ListChannels(key string, offset, limit uint64) ([]things.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels").Add(1)
		ms.latency.With("method", "list_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListChannels(key, offset, limit)
}

func (ms *metricsMiddleware) RemoveChannel(key, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_channel").Add(1)
		ms.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveChannel(key, id)
}

func (ms *metricsMiddleware) Connect(key, chanID, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "connect").Add(1)
		ms.latency.With("method", "connect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Connect(key, chanID, thingID)
}

func (ms *metricsMiddleware) Disconnect(key, chanID, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "disconnect").Add(1)
		ms.latency.With("method", "disconnect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Disconnect(key, chanID, thingID)
}

func (ms *metricsMiddleware) CanAccess(id, key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "can_access").Add(1)
		ms.latency.With("method", "can_access").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CanAccess(id, key)
}

func (ms *metricsMiddleware) Identify(key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identify(key)
}
