// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/channels"
	entityRolesAPI "github.com/absmach/magistrala/pkg/entityroles/api"
	"github.com/go-kit/kit/metrics"
)

var _ channels.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     channels.Service
	entityRolesAPI.RolesSvcMetricsMiddleware
}

// MetricsMiddleware returns a new metrics middleware wrapper.
func MetricsMiddleware(svc channels.Service, counter metrics.Counter, latency metrics.Histogram) channels.Service {
	rolesSvcMetricsMiddleware := entityRolesAPI.NewRolesSvcMetricsMiddleware("channels", svc, counter, latency)
	return &metricsMiddleware{
		counter:                   counter,
		latency:                   latency,
		svc:                       svc,
		RolesSvcMetricsMiddleware: rolesSvcMetricsMiddleware,
	}
}

func (ms *metricsMiddleware) CreateChannels(ctx context.Context, token string, chs ...channels.Channel) ([]channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "register_channels").Add(1)
		ms.latency.With("method", "register_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateChannels(ctx, token, chs...)
}

func (ms *metricsMiddleware) ViewChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_channel").Add(1)
		ms.latency.With("method", "view_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewChannel(ctx, token, id)
}

func (ms *metricsMiddleware) ListChannels(ctx context.Context, token string, pm channels.PageMetadata) (channels.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels").Add(1)
		ms.latency.With("method", "list_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListChannels(ctx, token, pm)
}

func (ms *metricsMiddleware) ListChannelsByThing(ctx context.Context, token, thingID string, pm channels.PageMetadata) (channels.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels_by_thing").Add(1)
		ms.latency.With("method", "list_channels_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListChannelsByThing(ctx, token, thingID, pm)
}

func (ms *metricsMiddleware) UpdateChannel(ctx context.Context, token string, channel channels.Channel) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel").Add(1)
		ms.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateChannel(ctx, token, channel)
}

func (ms *metricsMiddleware) UpdateChannelTags(ctx context.Context, token string, channel channels.Channel) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel_tags").Add(1)
		ms.latency.With("method", "update_channel_tags").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateChannelTags(ctx, token, channel)
}

func (ms *metricsMiddleware) EnableChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_channel").Add(1)
		ms.latency.With("method", "enable_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableChannel(ctx, token, id)
}

func (ms *metricsMiddleware) DisableChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_channel").Add(1)
		ms.latency.With("method", "disable_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableChannel(ctx, token, id)
}

func (ms *metricsMiddleware) RemoveChannel(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_channel").Add(1)
		ms.latency.With("method", "delete_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveChannel(ctx, token, id)
}

func (ms *metricsMiddleware) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "connect").Add(1)
		ms.latency.With("method", "connect").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Connect(ctx, token, chIDs, thIDs)
}
func (ms *metricsMiddleware) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "disconnect").Add(1)
		ms.latency.With("method", "disconnect").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Disconnect(ctx, token, chIDs, thIDs)
}
