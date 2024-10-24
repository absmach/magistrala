// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/authn"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
	"github.com/go-kit/kit/metrics"
)

var _ clients.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     clients.Service
	rmMW.RoleManagerMetricsMiddleware
}

// MetricsMiddleware returns a new metrics middleware wrapper.
func MetricsMiddleware(svc clients.Service, counter metrics.Counter, latency metrics.Histogram) clients.Service {
	return &metricsMiddleware{
		counter:                      counter,
		latency:                      latency,
		svc:                          svc,
		RoleManagerMetricsMiddleware: rmMW.NewRoleManagerMetricsMiddleware("clients", svc, counter, latency),
	}
}

func (ms *metricsMiddleware) CreateClients(ctx context.Context, session authn.Session, clients ...clients.Client) ([]clients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "register_clients").Add(1)
		ms.latency.With("method", "register_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateClients(ctx, session, clients...)
}

func (ms *metricsMiddleware) View(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_client").Add(1)
		ms.latency.With("method", "view_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.View(ctx, session, id)
}

func (ms *metricsMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm clients.Page) (clients.ClientsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients").Add(1)
		ms.latency.With("method", "list_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListClients(ctx, session, reqUserID, pm)
}

func (ms *metricsMiddleware) Update(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client").Add(1)
		ms.latency.With("method", "update_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Update(ctx, session, client)
}

func (ms *metricsMiddleware) UpdateTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_tags").Add(1)
		ms.latency.With("method", "update_client_tags").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateTags(ctx, session, client)
}

func (ms *metricsMiddleware) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (clients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_secret").Add(1)
		ms.latency.With("method", "update_client_secret").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
}

func (ms *metricsMiddleware) Enable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_client").Add(1)
		ms.latency.With("method", "enable_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Enable(ctx, session, id)
}

func (ms *metricsMiddleware) Disable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_client").Add(1)
		ms.latency.With("method", "disable_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Disable(ctx, session, id)
}

func (ms *metricsMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_client").Add(1)
		ms.latency.With("method", "delete_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Delete(ctx, session, id)
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
