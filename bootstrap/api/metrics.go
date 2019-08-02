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
	"github.com/mainflux/mainflux/bootstrap"
)

var _ bootstrap.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     bootstrap.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc bootstrap.Service, counter metrics.Counter, latency metrics.Histogram) bootstrap.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsMiddleware) Add(key string, cfg bootstrap.Config) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "add").Add(1)
		mm.latency.With("method", "add").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Add(key, cfg)
}

func (mm *metricsMiddleware) View(id, key string) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view").Add(1)
		mm.latency.With("method", "view").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.View(id, key)
}

func (mm *metricsMiddleware) Update(key string, cfg bootstrap.Config) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update").Add(1)
		mm.latency.With("method", "update").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Update(key, cfg)
}

func (mm *metricsMiddleware) UpdateCert(key, thingKey, clientCert, clientKey, caCert string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_cert").Add(1)
		mm.latency.With("method", "update_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateCert(key, thingKey, clientCert, clientKey, caCert)
}

func (mm *metricsMiddleware) UpdateConnections(key, id string, connections []string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_connections").Add(1)
		mm.latency.With("method", "update_connections").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateConnections(key, id, connections)
}

func (mm *metricsMiddleware) List(key string, filter bootstrap.Filter, offset, limit uint64) (saved bootstrap.ConfigsPage, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list").Add(1)
		mm.latency.With("method", "list").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.List(key, filter, offset, limit)
}

func (mm *metricsMiddleware) Remove(id, key string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove").Add(1)
		mm.latency.With("method", "remove").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Remove(id, key)
}

func (mm *metricsMiddleware) Bootstrap(externalKey, externalID string, secure bool) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "bootstrap").Add(1)
		mm.latency.With("method", "bootstrap").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Bootstrap(externalKey, externalID, secure)
}

func (mm *metricsMiddleware) ChangeState(id, key string, state bootstrap.State) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "change_state").Add(1)
		mm.latency.With("method", "change_state").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ChangeState(id, key, state)
}

func (mm *metricsMiddleware) UpdateChannelHandler(channel bootstrap.Channel) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_channel").Add(1)
		mm.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateChannelHandler(channel)
}

func (mm *metricsMiddleware) RemoveConfigHandler(id string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_config").Add(1)
		mm.latency.With("method", "remove_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveConfigHandler(id)
}

func (mm *metricsMiddleware) RemoveChannelHandler(id string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_channel").Add(1)
		mm.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveChannelHandler(id)
}

func (mm *metricsMiddleware) DisconnectThingHandler(channelID, thingID string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "disconnect_thing_handler").Add(1)
		mm.latency.With("method", "disconnect_thing_handler").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DisconnectThingHandler(channelID, thingID)
}
