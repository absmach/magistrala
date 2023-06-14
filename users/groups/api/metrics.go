// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/users/groups"
)

var _ groups.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     groups.Service
}

// MetricsMiddleware instruments policies service by tracking request count and latency.
func MetricsMiddleware(svc groups.Service, counter metrics.Counter, latency metrics.Histogram) groups.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

// CreateGroup instruments CreateGroup method with metrics.
func (ms *metricsMiddleware) CreateGroup(ctx context.Context, token string, g mfgroups.Group) (mfgroups.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_group").Add(1)
		ms.latency.With("method", "create_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateGroup(ctx, token, g)
}

// UpdateGroup instruments UpdateGroup method with metrics.
func (ms *metricsMiddleware) UpdateGroup(ctx context.Context, token string, group mfgroups.Group) (rGroup mfgroups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_group").Add(1)
		ms.latency.With("method", "update_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateGroup(ctx, token, group)
}

// ViewGroup instruments ViewGroup method with metrics.
func (ms *metricsMiddleware) ViewGroup(ctx context.Context, token, id string) (g mfgroups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group").Add(1)
		ms.latency.With("method", "view_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewGroup(ctx, token, id)
}

// ListGroups instruments ListGroups method with metrics.
func (ms *metricsMiddleware) ListGroups(ctx context.Context, token string, gp mfgroups.GroupsPage) (cg mfgroups.GroupsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_groups").Add(1)
		ms.latency.With("method", "list_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListGroups(ctx, token, gp)
}

// EnableGroup instruments EnableGroup method with metrics.
func (ms *metricsMiddleware) EnableGroup(ctx context.Context, token string, id string) (g mfgroups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_group").Add(1)
		ms.latency.With("method", "enable_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableGroup(ctx, token, id)
}

// DisableGroup instruments DisableGroup method with metrics.
func (ms *metricsMiddleware) DisableGroup(ctx context.Context, token string, id string) (g mfgroups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_group").Add(1)
		ms.latency.With("method", "disable_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableGroup(ctx, token, id)
}

// ListMemberships instruments ListMemberships method with metrics.
func (ms *metricsMiddleware) ListMemberships(ctx context.Context, token, clientID string, gp mfgroups.GroupsPage) (mp mfgroups.MembershipsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_memberships").Add(1)
		ms.latency.With("method", "list_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListMemberships(ctx, token, clientID, gp)
}
