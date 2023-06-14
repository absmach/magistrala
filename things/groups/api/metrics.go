// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/things/groups"
)

var _ groups.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     groups.Service
}

// MetricsMiddleware returns a new metrics middleware wrapper.
func MetricsMiddleware(svc groups.Service, counter metrics.Counter, latency metrics.Histogram) groups.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateGroups(ctx context.Context, token string, g ...mfgroups.Group) ([]mfgroups.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_channel").Add(1)
		ms.latency.With("method", "create_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateGroups(ctx, token, g...)
}

func (ms *metricsMiddleware) UpdateGroup(ctx context.Context, token string, group mfgroups.Group) (rGroup mfgroups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel").Add(1)
		ms.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateGroup(ctx, token, group)
}

func (ms *metricsMiddleware) ViewGroup(ctx context.Context, token, id string) (g mfgroups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_channel").Add(1)
		ms.latency.With("method", "view_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewGroup(ctx, token, id)
}

func (ms *metricsMiddleware) ListGroups(ctx context.Context, token string, gp mfgroups.GroupsPage) (cg mfgroups.GroupsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels").Add(1)
		ms.latency.With("method", "list_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListGroups(ctx, token, gp)
}

func (ms *metricsMiddleware) EnableGroup(ctx context.Context, token string, id string) (g mfgroups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_channel").Add(1)
		ms.latency.With("method", "enable_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableGroup(ctx, token, id)
}

func (ms *metricsMiddleware) DisableGroup(ctx context.Context, token string, id string) (g mfgroups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_channel").Add(1)
		ms.latency.With("method", "disable_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableGroup(ctx, token, id)
}

func (ms *metricsMiddleware) ListMemberships(ctx context.Context, token, clientID string, gp mfgroups.GroupsPage) (mp mfgroups.MembershipsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels_by_thing").Add(1)
		ms.latency.With("method", "list_channels_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListMemberships(ctx, token, clientID, gp)
}
