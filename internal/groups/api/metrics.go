// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/groups"
	"github.com/go-kit/kit/metrics"
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
func (ms *metricsMiddleware) CreateGroup(ctx context.Context, token, kind string, g groups.Group) (groups.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_group").Add(1)
		ms.latency.With("method", "create_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateGroup(ctx, token, kind, g)
}

// UpdateGroup instruments UpdateGroup method with metrics.
func (ms *metricsMiddleware) UpdateGroup(ctx context.Context, token string, group groups.Group) (rGroup groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_group").Add(1)
		ms.latency.With("method", "update_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateGroup(ctx, token, group)
}

// ViewGroup instruments ViewGroup method with metrics.
func (ms *metricsMiddleware) ViewGroup(ctx context.Context, token, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group").Add(1)
		ms.latency.With("method", "view_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewGroup(ctx, token, id)
}

// ViewGroupPerms instruments ViewGroup method with metrics.
func (ms *metricsMiddleware) ViewGroupPerms(ctx context.Context, token, id string) (p []string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group_perms").Add(1)
		ms.latency.With("method", "view_group_perms").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewGroupPerms(ctx, token, id)
}

// ListGroups instruments ListGroups method with metrics.
func (ms *metricsMiddleware) ListGroups(ctx context.Context, token, memberKind, memberID string, gp groups.Page) (cg groups.Page, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_groups").Add(1)
		ms.latency.With("method", "list_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListGroups(ctx, token, memberKind, memberID, gp)
}

// EnableGroup instruments EnableGroup method with metrics.
func (ms *metricsMiddleware) EnableGroup(ctx context.Context, token, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_group").Add(1)
		ms.latency.With("method", "enable_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableGroup(ctx, token, id)
}

// DisableGroup instruments DisableGroup method with metrics.
func (ms *metricsMiddleware) DisableGroup(ctx context.Context, token, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_group").Add(1)
		ms.latency.With("method", "disable_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableGroup(ctx, token, id)
}

// ListMembers instruments ListMembers method with metrics.
func (ms *metricsMiddleware) ListMembers(ctx context.Context, token, groupID, permission, memberKind string) (mp groups.MembersPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_memberships").Add(1)
		ms.latency.With("method", "list_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListMembers(ctx, token, groupID, permission, memberKind)
}

// Assign instruments Assign method with metrics.
func (ms *metricsMiddleware) Assign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign").Add(1)
		ms.latency.With("method", "assign").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Assign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

// Unassign instruments Unassign method with metrics.
func (ms *metricsMiddleware) Unassign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign").Add(1)
		ms.latency.With("method", "unassign").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Unassign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (ms *metricsMiddleware) DeleteGroup(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_group").Add(1)
		ms.latency.With("method", "delete_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeleteGroup(ctx, token, id)
}
