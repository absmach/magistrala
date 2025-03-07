// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"context"
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/roles"
	rmMW "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
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

func (ms *metricsMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (domains.Domain, []roles.RoleProvision, error) {
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

func (mm *metricsMiddleware) SendInvitation(ctx context.Context, session authn.Session, invitation domains.Invitation) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "send_invitation").Add(1)
		mm.latency.With("method", "send_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.SendInvitation(ctx, session, invitation)
}

func (mm *metricsMiddleware) ViewInvitation(ctx context.Context, session authn.Session, userID, domainID string) (invitation domains.Invitation, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_invitation").Add(1)
		mm.latency.With("method", "view_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.ViewInvitation(ctx, session, userID, domainID)
}

func (mm *metricsMiddleware) ListInvitations(ctx context.Context, session authn.Session, pm domains.InvitationPageMeta) (invs domains.InvitationPage, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_invitations").Add(1)
		mm.latency.With("method", "list_invitations").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.ListInvitations(ctx, session, pm)
}

func (mm *metricsMiddleware) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "accept_invitation").Add(1)
		mm.latency.With("method", "accept_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.AcceptInvitation(ctx, session, domainID)
}

func (mm *metricsMiddleware) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "reject_invitation").Add(1)
		mm.latency.With("method", "reject_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.RejectInvitation(ctx, session, domainID)
}

func (mm *metricsMiddleware) DeleteInvitation(ctx context.Context, session authn.Session, userID, domainID string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_invitation").Add(1)
		mm.latency.With("method", "delete_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.DeleteInvitation(ctx, session, userID, domainID)
}
