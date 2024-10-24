// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
	"github.com/go-kit/kit/metrics"
)

var _ channels.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     channels.Service
	rmMW.RoleManagerMetricsMiddleware
}

// MetricsMiddleware returns a new metrics middleware wrapper.
func MetricsMiddleware(svc channels.Service, counter metrics.Counter, latency metrics.Histogram) channels.Service {
	return &metricsMiddleware{
		counter:                      counter,
		latency:                      latency,
		svc:                          svc,
		RoleManagerMetricsMiddleware: rmMW.NewRoleManagerMetricsMiddleware("channels", svc, counter, latency),
	}
}

func (ms *metricsMiddleware) CreateChannels(ctx context.Context, session authn.Session, chs ...channels.Channel) ([]channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "register_channels").Add(1)
		ms.latency.With("method", "register_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateChannels(ctx, session, chs...)
}

func (ms *metricsMiddleware) ViewChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_channel").Add(1)
		ms.latency.With("method", "view_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewChannel(ctx, session, id)
}

func (ms *metricsMiddleware) ListChannels(ctx context.Context, session authn.Session, pm channels.PageMetadata) (channels.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels").Add(1)
		ms.latency.With("method", "list_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListChannels(ctx, session, pm)
}

func (ms *metricsMiddleware) ListChannelsByClient(ctx context.Context, session authn.Session, clientID string, pm channels.PageMetadata) (channels.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels_by_client").Add(1)
		ms.latency.With("method", "list_channels_by_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListChannelsByClient(ctx, session, clientID, pm)
}

func (ms *metricsMiddleware) UpdateChannel(ctx context.Context, session authn.Session, channel channels.Channel) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel").Add(1)
		ms.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateChannel(ctx, session, channel)
}

func (ms *metricsMiddleware) UpdateChannelTags(ctx context.Context, session authn.Session, channel channels.Channel) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel_tags").Add(1)
		ms.latency.With("method", "update_channel_tags").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateChannelTags(ctx, session, channel)
}

func (ms *metricsMiddleware) EnableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_channel").Add(1)
		ms.latency.With("method", "enable_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableChannel(ctx, session, id)
}

func (ms *metricsMiddleware) DisableChannel(ctx context.Context, session authn.Session, id string) (channels.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_channel").Add(1)
		ms.latency.With("method", "disable_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableChannel(ctx, session, id)
}

func (ms *metricsMiddleware) RemoveChannel(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_channel").Add(1)
		ms.latency.With("method", "delete_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveChannel(ctx, session, id)
}

func (ms *metricsMiddleware) Connect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "connect").Add(1)
		ms.latency.With("method", "connect").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Connect(ctx, session, chIDs, thIDs, connTypes)
}

func (ms *metricsMiddleware) Disconnect(ctx context.Context, session authn.Session, chIDs, thIDs []string, connTypes []connections.ConnType) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "disconnect").Add(1)
		ms.latency.With("method", "disconnect").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Disconnect(ctx, session, chIDs, thIDs, connTypes)
}

func (ms *metricsMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "set_parent_group").Add(1)
		ms.latency.With("method", "set_parent_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (ms *metricsMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_parent_group").Add(1)
		ms.latency.With("method", "remove_parent_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveParentGroup(ctx, session, id)
}
