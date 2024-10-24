// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/pkg/authn"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
	"github.com/go-kit/kit/metrics"
)

var _ domains.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     domains.Service
	rmMW.RoleManagerMetricsMiddleware
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc domains.Service, counter metrics.Counter, latency metrics.Histogram) domains.Service {
	rmmw := rmMW.NewRoleManagerMetricsMiddleware("domains", svc, counter, latency)

	return &metricsMiddleware{
		counter:                      counter,
		latency:                      latency,
		svc:                          svc,
		RoleManagerMetricsMiddleware: rmmw,
	}
}

func (ms *metricsMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (domains.Domain, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_domain").Add(1)
		ms.latency.With("method", "create_domain").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.CreateDomain(ctx, session, d)
}

func (ms *metricsMiddleware) RetrieveDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "retrieve_domain").Add(1)
		ms.latency.With("method", "retrieve_domain").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.RetrieveDomain(ctx, session, id)
}

func (ms *metricsMiddleware) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_domain").Add(1)
		ms.latency.With("method", "update_domain").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.UpdateDomain(ctx, session, id, d)
}

func (ms *metricsMiddleware) EnableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "enable_domain").Add(1)
		ms.latency.With("method", "enable_domain").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.EnableDomain(ctx, session, id)
}

func (ms *metricsMiddleware) DisableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "disable_domain").Add(1)
		ms.latency.With("method", "disable_domain").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DisableDomain(ctx, session, id)
}

func (ms *metricsMiddleware) FreezeDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "freeze_domain").Add(1)
		ms.latency.With("method", "freeze_domain").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.FreezeDomain(ctx, session, id)
}

func (ms *metricsMiddleware) ListDomains(ctx context.Context, session authn.Session, page domains.Page) (domains.DomainsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_domains").Add(1)
		ms.latency.With("method", "list_domains").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.ListDomains(ctx, session, page)
}

func (ms *metricsMiddleware) DeleteUserFromDomains(ctx context.Context, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_user_from_domains").Add(1)
		ms.latency.With("method", "delete_user_from_domains").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return ms.svc.DeleteUserFromDomains(ctx, id)
}
