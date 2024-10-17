// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/go-kit/kit/metrics"
)

var _ invitations.Service = (*metricsmw)(nil)

type metricsmw struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     invitations.Service
}

func Metrics(counter metrics.Counter, latency metrics.Histogram, svc invitations.Service) invitations.Service {
	return &metricsmw{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (mm *metricsmw) SendInvitation(ctx context.Context, session authn.Session, invitation invitations.Invitation) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "send_invitation").Add(1)
		mm.latency.With("method", "send_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.SendInvitation(ctx, session, invitation)
}

func (mm *metricsmw) ViewInvitation(ctx context.Context, session authn.Session, userID, domainID string) (invitation invitations.Invitation, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "view_invitation").Add(1)
		mm.latency.With("method", "view_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.ViewInvitation(ctx, session, userID, domainID)
}

func (mm *metricsmw) ListInvitations(ctx context.Context, session authn.Session, page invitations.Page) (invs invitations.InvitationPage, err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_invitations").Add(1)
		mm.latency.With("method", "list_invitations").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.ListInvitations(ctx, session, page)
}

func (mm *metricsmw) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "accept_invitation").Add(1)
		mm.latency.With("method", "accept_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.AcceptInvitation(ctx, session, domainID)
}

func (mm *metricsmw) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "reject_invitation").Add(1)
		mm.latency.With("method", "reject_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.RejectInvitation(ctx, session, domainID)
}

func (mm *metricsmw) DeleteInvitation(ctx context.Context, session authn.Session, userID, domainID string) (err error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_invitation").Add(1)
		mm.latency.With("method", "delete_invitation").Observe(time.Since(begin).Seconds())
	}(time.Now())
	return mm.svc.DeleteInvitation(ctx, session, userID, domainID)
}
