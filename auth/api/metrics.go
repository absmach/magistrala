// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/internal/groups"
)

var _ auth.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     auth.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc auth.Service, counter metrics.Counter, latency metrics.Histogram) auth.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) Issue(ctx context.Context, token string, key auth.Key) (auth.Key, string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "issue_key").Add(1)
		ms.latency.With("method", "issue_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Issue(ctx, token, key)
}

func (ms *metricsMiddleware) Revoke(ctx context.Context, token, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "revoke_key").Add(1)
		ms.latency.With("method", "revoke_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Revoke(ctx, token, id)
}

func (ms *metricsMiddleware) RetrieveKey(ctx context.Context, token, id string) (auth.Key, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "retrieve_key").Add(1)
		ms.latency.With("method", "retrieve_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RetrieveKey(ctx, token, id)
}

func (ms *metricsMiddleware) Identify(ctx context.Context, token string) (auth.Identity, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identify(ctx, token)
}

func (ms *metricsMiddleware) Authorize(ctx context.Context, token, sub, obj, act string) (auth bool, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "authorize").Add(1)
		ms.latency.With("method", "authorize").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Authorize(ctx, token, sub, obj, act)
}

func (ms *metricsMiddleware) CreateGroup(ctx context.Context, token string, g groups.Group) (id string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_group").Add(1)
		ms.latency.With("method", "create_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateGroup(ctx, token, g)
}

func (ms *metricsMiddleware) UpdateGroup(ctx context.Context, token string, g groups.Group) (gr groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_group").Add(1)
		ms.latency.With("method", "update_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateGroup(ctx, token, g)
}

func (ms *metricsMiddleware) RemoveGroup(ctx context.Context, token string, id string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_group").Add(1)
		ms.latency.With("method", "remove_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveGroup(ctx, token, id)
}

func (ms *metricsMiddleware) ViewGroup(ctx context.Context, token, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group").Add(1)
		ms.latency.With("method", "view_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewGroup(ctx, token, id)
}

func (ms *metricsMiddleware) ListGroups(ctx context.Context, token string, level uint64, gm groups.Metadata) (gp groups.GroupPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_groups").Add(1)
		ms.latency.With("method", "list_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListGroups(ctx, token, level, gm)
}

func (ms *metricsMiddleware) ListParents(ctx context.Context, token, childID string, level uint64, gm groups.Metadata) (gp groups.GroupPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "parents").Add(1)
		ms.latency.With("method", "parents").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListParents(ctx, token, childID, level, gm)
}

func (ms *metricsMiddleware) ListChildren(ctx context.Context, token, parentID string, level uint64, gm groups.Metadata) (gp groups.GroupPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_children").Add(1)
		ms.latency.With("method", "list_children").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListChildren(ctx, token, parentID, level, gm)
}

func (ms *metricsMiddleware) ListMembers(ctx context.Context, token, groupID string, offset, limit uint64, gm groups.Metadata) (gp groups.MemberPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_members").Add(1)
		ms.latency.With("method", "list_members").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListMembers(ctx, token, groupID, offset, limit, gm)
}

func (ms *metricsMiddleware) ListMemberships(ctx context.Context, token, groupID string, offset, limit uint64, gm groups.Metadata) (gp groups.GroupPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_memberships").Add(1)
		ms.latency.With("method", "list_memberships").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListMemberships(ctx, token, groupID, offset, limit, gm)
}

func (ms *metricsMiddleware) Assign(ctx context.Context, token, memberID, groupID string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign").Add(1)
		ms.latency.With("method", "assign").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Assign(ctx, token, memberID, groupID)
}

func (ms *metricsMiddleware) Unassign(ctx context.Context, token, memberID, groupID string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign").Add(1)
		ms.latency.With("method", "unassign").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Unassign(ctx, token, memberID, groupID)
}
