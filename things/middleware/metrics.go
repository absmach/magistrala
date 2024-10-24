// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/things"
	"github.com/go-kit/kit/metrics"
)

var _ things.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     things.Service
}

// MetricsMiddleware returns a new metrics middleware wrapper.
func MetricsMiddleware(svc things.Service, counter metrics.Counter, latency metrics.Histogram) things.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateClients(ctx context.Context, session authn.Session, things ...things.Client) ([]things.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "register_clients").Add(1)
		ms.latency.With("method", "register_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateClients(ctx, session, things...)
}

func (ms *metricsMiddleware) View(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_client").Add(1)
		ms.latency.With("method", "view_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.View(ctx, session, id)
}

func (ms *metricsMiddleware) ViewPerms(ctx context.Context, session authn.Session, id string) ([]string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_client_permissions").Add(1)
		ms.latency.With("method", "view_client_permissions").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ViewPerms(ctx, session, id)
}

func (ms *metricsMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm things.Page) (things.ClientsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients").Add(1)
		ms.latency.With("method", "list_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListClients(ctx, session, reqUserID, pm)
}

func (ms *metricsMiddleware) Update(ctx context.Context, session authn.Session, thing things.Client) (things.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client").Add(1)
		ms.latency.With("method", "update_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Update(ctx, session, thing)
}

func (ms *metricsMiddleware) UpdateTags(ctx context.Context, session authn.Session, thing things.Client) (things.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_tags").Add(1)
		ms.latency.With("method", "update_client_tags").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateTags(ctx, session, thing)
}

func (ms *metricsMiddleware) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (things.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client_secret").Add(1)
		ms.latency.With("method", "update_client_secret").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
}

func (ms *metricsMiddleware) Enable(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_client").Add(1)
		ms.latency.With("method", "enable_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Enable(ctx, session, id)
}

func (ms *metricsMiddleware) Disable(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_client").Add(1)
		ms.latency.With("method", "disable_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Disable(ctx, session, id)
}

func (ms *metricsMiddleware) ListClientsByGroup(ctx context.Context, session authn.Session, groupID string, pm things.Page) (mp things.MembersPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients_by_channel").Add(1)
		ms.latency.With("method", "list_clients_by_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListClientsByGroup(ctx, session, groupID, pm)
}

func (ms *metricsMiddleware) Identify(ctx context.Context, key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify_client").Add(1)
		ms.latency.With("method", "identify_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Identify(ctx, key)
}

func (ms *metricsMiddleware) Authorize(ctx context.Context, req things.AuthzReq) (id string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "authorize").Add(1)
		ms.latency.With("method", "authorize").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Authorize(ctx, req)
}

func (ms *metricsMiddleware) Share(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "share").Add(1)
		ms.latency.With("method", "share").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Share(ctx, session, id, relation, userids...)
}

func (ms *metricsMiddleware) Unshare(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "unshare").Add(1)
		ms.latency.With("method", "unshare").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Unshare(ctx, session, id, relation, userids...)
}

func (ms *metricsMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_client").Add(1)
		ms.latency.With("method", "delete_client").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.Delete(ctx, session, id)
}
