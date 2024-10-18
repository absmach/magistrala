// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/authn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ invitations.Service = (*tracing)(nil)

type tracing struct {
	tracer trace.Tracer
	svc    invitations.Service
}

func Tracing(svc invitations.Service, tracer trace.Tracer) invitations.Service {
	return &tracing{tracer, svc}
}

func (tm *tracing) SendInvitation(ctx context.Context, session authn.Session, invitation invitations.Invitation) (err error) {
	ctx, span := tm.tracer.Start(ctx, "send_invitation", trace.WithAttributes(
		attribute.String("domain_id", invitation.DomainID),
		attribute.String("user_id", invitation.UserID),
	))
	defer span.End()

	return tm.svc.SendInvitation(ctx, session, invitation)
}

func (tm *tracing) ViewInvitation(ctx context.Context, session authn.Session, userID, domain string) (invitation invitations.Invitation, err error) {
	ctx, span := tm.tracer.Start(ctx, "view_invitation", trace.WithAttributes(
		attribute.String("user_id", userID),
		attribute.String("domain_id", domain),
	))
	defer span.End()

	return tm.svc.ViewInvitation(ctx, session, userID, domain)
}

func (tm *tracing) ListInvitations(ctx context.Context, session authn.Session, page invitations.Page) (invs invitations.InvitationPage, err error) {
	ctx, span := tm.tracer.Start(ctx, "list_invitations", trace.WithAttributes(
		attribute.Int("limit", int(page.Limit)),
		attribute.Int("offset", int(page.Offset)),
		attribute.String("user_id", page.UserID),
		attribute.String("domain_id", page.DomainID),
		attribute.String("invited_by", page.InvitedBy),
	))
	defer span.End()

	return tm.svc.ListInvitations(ctx, session, page)
}

func (tm *tracing) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	ctx, span := tm.tracer.Start(ctx, "accept_invitation", trace.WithAttributes(
		attribute.String("domain_id", domainID),
	))
	defer span.End()

	return tm.svc.AcceptInvitation(ctx, session, domainID)
}

func (tm *tracing) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	ctx, span := tm.tracer.Start(ctx, "reject_invitation", trace.WithAttributes(
		attribute.String("domain_id", domainID),
	))
	defer span.End()

	return tm.svc.RejectInvitation(ctx, session, domainID)
}

func (tm *tracing) DeleteInvitation(ctx context.Context, session authn.Session, userID, domainID string) (err error) {
	ctx, span := tm.tracer.Start(ctx, "delete_invitation", trace.WithAttributes(
		attribute.String("user_id", userID),
		attribute.String("domain_id", domainID),
	))
	defer span.End()

	return tm.svc.DeleteInvitation(ctx, session, userID, domainID)
}
