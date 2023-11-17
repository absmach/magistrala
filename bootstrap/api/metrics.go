// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/go-kit/kit/metrics"
)

var _ bootstrap.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     bootstrap.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc bootstrap.Service, counter metrics.Counter, latency metrics.Histogram) bootstrap.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

// Add instruments Add method with metrics.
func (mm *metricsMiddleware) Add(ctx context.Context, token string, cfg bootstrap.Config) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "add").Add(1)
		mm.latency.With("method", "add").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Add(ctx, token, cfg)
}

// View instruments View method with metrics.
func (mm *metricsMiddleware) View(ctx context.Context, token, id string) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view").Add(1)
		mm.latency.With("method", "view").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.View(ctx, token, id)
}

// Update instruments Update method with metrics.
func (mm *metricsMiddleware) Update(ctx context.Context, token string, cfg bootstrap.Config) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update").Add(1)
		mm.latency.With("method", "update").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Update(ctx, token, cfg)
}

// UpdateCert instruments UpdateCert method with metrics.
func (mm *metricsMiddleware) UpdateCert(ctx context.Context, token, thingKey, clientCert, clientKey, caCert string) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_cert").Add(1)
		mm.latency.With("method", "update_cert").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateCert(ctx, token, thingKey, clientCert, clientKey, caCert)
}

// UpdateConnections instruments UpdateConnections method with metrics.
func (mm *metricsMiddleware) UpdateConnections(ctx context.Context, token, id string, connections []string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_connections").Add(1)
		mm.latency.With("method", "update_connections").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateConnections(ctx, token, id, connections)
}

// List instruments List method with metrics.
func (mm *metricsMiddleware) List(ctx context.Context, token string, filter bootstrap.Filter, offset, limit uint64) (saved bootstrap.ConfigsPage, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list").Add(1)
		mm.latency.With("method", "list").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.List(ctx, token, filter, offset, limit)
}

// Remove instruments Remove method with metrics.
func (mm *metricsMiddleware) Remove(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove").Add(1)
		mm.latency.With("method", "remove").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Remove(ctx, token, id)
}

// Bootstrap instruments Bootstrap method with metrics.
func (mm *metricsMiddleware) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "bootstrap").Add(1)
		mm.latency.With("method", "bootstrap").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.Bootstrap(ctx, externalKey, externalID, secure)
}

// ChangeState instruments ChangeState method with metrics.
func (mm *metricsMiddleware) ChangeState(ctx context.Context, token, id string, state bootstrap.State) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "change_state").Add(1)
		mm.latency.With("method", "change_state").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ChangeState(ctx, token, id, state)
}

// UpdateChannelHandler instruments UpdateChannelHandler method with metrics.
func (mm *metricsMiddleware) UpdateChannelHandler(ctx context.Context, channel bootstrap.Channel) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "update_channel").Add(1)
		mm.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.UpdateChannelHandler(ctx, channel)
}

// RemoveConfigHandler instruments RemoveConfigHandler method with metrics.
func (mm *metricsMiddleware) RemoveConfigHandler(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_config").Add(1)
		mm.latency.With("method", "remove_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveConfigHandler(ctx, id)
}

// RemoveChannelHandler instruments RemoveChannelHandler method with metrics.
func (mm *metricsMiddleware) RemoveChannelHandler(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "remove_channel").Add(1)
		mm.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RemoveChannelHandler(ctx, id)
}

// DisconnectThingHandler instruments DisconnectThingHandler method with metrics.
func (mm *metricsMiddleware) DisconnectThingHandler(ctx context.Context, channelID, thingID string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "disconnect_thing_handler").Add(1)
		mm.latency.With("method", "disconnect_thing_handler").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DisconnectThingHandler(ctx, channelID, thingID)
}
