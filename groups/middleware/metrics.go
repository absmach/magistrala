// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/pkg/authn"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
	"github.com/go-kit/kit/metrics"
)

var _ groups.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     groups.Service
	rmMW.RoleManagerMetricsMiddleware
}

// MetricsMiddleware instruments policies service by tracking request count and latency.
func MetricsMiddleware(svc groups.Service, counter metrics.Counter, latency metrics.Histogram) groups.Service {
	rmm := rmMW.NewRoleManagerMetricsMiddleware("group", svc, counter, latency)
	return &metricsMiddleware{
		counter:                      counter,
		latency:                      latency,
		svc:                          svc,
		RoleManagerMetricsMiddleware: rmm,
	}
}

// CreateGroup instruments CreateGroup method with metrics.
func (ms *metricsMiddleware) CreateGroup(ctx context.Context, session authn.Session, g groups.Group) (groups.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_group").Add(1)
		ms.latency.With("method", "create_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateGroup(ctx, session, g)
}

// UpdateGroup instruments UpdateGroup method with metrics.
func (ms *metricsMiddleware) UpdateGroup(ctx context.Context, session authn.Session, group groups.Group) (rGroup groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_group").Add(1)
		ms.latency.With("method", "update_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateGroup(ctx, session, group)
}

// ViewGroup instruments ViewGroup method with metrics.
func (ms *metricsMiddleware) ViewGroup(ctx context.Context, session authn.Session, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group").Add(1)
		ms.latency.With("method", "view_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewGroup(ctx, session, id)
}

// ListGroups instruments ListGroups method with metrics.
func (ms *metricsMiddleware) ListGroups(ctx context.Context, session authn.Session, pm groups.PageMeta) (cg groups.Page, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_groups").Add(1)
		ms.latency.With("method", "list_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListGroups(ctx, session, pm)
}

func (ms *metricsMiddleware) ListUserGroups(ctx context.Context, session authn.Session, userID string, pm groups.PageMeta) (cg groups.Page, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_user_groups").Add(1)
		ms.latency.With("method", "list_user_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListUserGroups(ctx, session, userID, pm)
}

// EnableGroup instruments EnableGroup method with metrics.
func (ms *metricsMiddleware) EnableGroup(ctx context.Context, session authn.Session, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_group").Add(1)
		ms.latency.With("method", "enable_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableGroup(ctx, session, id)
}

// DisableGroup instruments DisableGroup method with metrics.
func (ms *metricsMiddleware) DisableGroup(ctx context.Context, session authn.Session, id string) (g groups.Group, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_group").Add(1)
		ms.latency.With("method", "disable_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableGroup(ctx, session, id)
}

func (ms *metricsMiddleware) DeleteGroup(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_group").Add(1)
		ms.latency.With("method", "delete_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeleteGroup(ctx, session, id)
}

func (ms *metricsMiddleware) RetrieveGroupHierarchy(ctx context.Context, session authn.Session, id string, hm groups.HierarchyPageMeta) (groups.HierarchyPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_parent_groups").Add(1)
		ms.latency.With("method", "list_parent_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RetrieveGroupHierarchy(ctx, session, id, hm)
}

func (ms *metricsMiddleware) AddParentGroup(ctx context.Context, session authn.Session, id, parentID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_parent_group").Add(1)
		ms.latency.With("method", "add_parent_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.AddParentGroup(ctx, session, id, parentID)
}

func (ms *metricsMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_parent_group").Add(1)
		ms.latency.With("method", "remove_parent_group").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveParentGroup(ctx, session, id)
}

func (ms *metricsMiddleware) AddChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_children_groups").Add(1)
		ms.latency.With("method", "add_children_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.AddChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (ms *metricsMiddleware) RemoveChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_children_groups").Add(1)
		ms.latency.With("method", "remove_children_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (ms *metricsMiddleware) RemoveAllChildrenGroups(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_all_children_groups").Add(1)
		ms.latency.With("method", "remove_all_children_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RemoveAllChildrenGroups(ctx, session, id)
}

func (ms *metricsMiddleware) ListChildrenGroups(ctx context.Context, session authn.Session, id string, startLevel, endLevel int64, pm groups.PageMeta) (groups.Page, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_children_groups").Add(1)
		ms.latency.With("method", "list_children_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListChildrenGroups(ctx, session, id, startLevel, endLevel, pm)
}
