// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/roles"
	rolemw "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/absmach/supermq/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ domains.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    domains.Service
	rolemw.RoleManagerTracing
}

// NewTracing returns a new domains service with tracing capabilities.
func NewTracing(svc domains.Service, tracer trace.Tracer) domains.Service {
	return &tracingMiddleware{tracer, svc, rolemw.NewTracing("domain", svc, tracer)}
}

func (tm *tracingMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (domains.Domain, []roles.RoleProvision, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "create_domain", trace.WithAttributes(
		attribute.String("name", d.Name),
	))
	defer span.End()
	return tm.svc.CreateDomain(ctx, session, d)
}

func (tm *tracingMiddleware) RetrieveDomain(ctx context.Context, session authn.Session, id string, withRoles bool) (domains.Domain, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "view_domain", trace.WithAttributes(
		attribute.String("id", id),
		attribute.Bool("with_roles", withRoles),
	))
	defer span.End()
	return tm.svc.RetrieveDomain(ctx, session, id, withRoles)
}

func (tm *tracingMiddleware) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "update_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.UpdateDomain(ctx, session, id, d)
}

func (tm *tracingMiddleware) EnableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "enable_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.EnableDomain(ctx, session, id)
}

func (tm *tracingMiddleware) DisableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "disable_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.DisableDomain(ctx, session, id)
}

func (tm *tracingMiddleware) FreezeDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "freeze_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.FreezeDomain(ctx, session, id)
}

func (tm *tracingMiddleware) ListDomains(ctx context.Context, session authn.Session, p domains.Page) (domains.DomainsPage, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "list_domains")
	defer span.End()
	return tm.svc.ListDomains(ctx, session, p)
}

func (tm *tracingMiddleware) DeleteDomain(ctx context.Context, session authn.Session, id string) (err error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "delete_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.DeleteDomain(ctx, session, id)
}

func (tm *tracingMiddleware) SendInvitation(ctx context.Context, session authn.Session, invitation domains.Invitation) (domains.Invitation, error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "send_invitation", trace.WithAttributes(
		attribute.String("domain_id", invitation.DomainID),
		attribute.String("invitee_user_id", invitation.InviteeUserID),
	))
	defer span.End()

	return tm.svc.SendInvitation(ctx, session, invitation)
}

func (tm *tracingMiddleware) ListInvitations(ctx context.Context, session authn.Session, pm domains.InvitationPageMeta) (invs domains.InvitationPage, err error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "list_invitations", trace.WithAttributes(
		attribute.Int("limit", int(pm.Limit)),
		attribute.Int("offset", int(pm.Offset)),
		attribute.String("invitee_user_id", pm.InviteeUserID),
		attribute.String("domain_id", pm.DomainID),
		attribute.String("invited_by", pm.InvitedBy),
	))
	defer span.End()

	return tm.svc.ListInvitations(ctx, session, pm)
}

func (tm *tracingMiddleware) ListDomainInvitations(ctx context.Context, session authn.Session, pm domains.InvitationPageMeta) (invs domains.InvitationPage, err error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "list_domain_invitations", trace.WithAttributes(
		attribute.Int("limit", int(pm.Limit)),
		attribute.Int("offset", int(pm.Offset)),
		attribute.String("domain_id", session.DomainID),
	))
	defer span.End()

	return tm.svc.ListDomainInvitations(ctx, session, pm)
}

func (tm *tracingMiddleware) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (inv domains.Invitation, err error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "accept_invitation", trace.WithAttributes(
		attribute.String("domain_id", domainID),
	))
	defer span.End()

	return tm.svc.AcceptInvitation(ctx, session, domainID)
}

func (tm *tracingMiddleware) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (inv domains.Invitation, err error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "reject_invitation", trace.WithAttributes(
		attribute.String("domain_id", domainID),
	))
	defer span.End()

	return tm.svc.RejectInvitation(ctx, session, domainID)
}

func (tm *tracingMiddleware) DeleteInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) (err error) {
	ctx, span := tracing.StartSpan(ctx, tm.tracer, "delete_invitation", trace.WithAttributes(
		attribute.String("invitee_user_id", inviteeUserID),
		attribute.String("domain_id", domainID),
	))
	defer span.End()

	return tm.svc.DeleteInvitation(ctx, session, inviteeUserID, domainID)
}
