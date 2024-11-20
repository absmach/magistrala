// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/pkg/authn"
	rmTrace "github.com/absmach/magistrala/pkg/roles/rolemanager/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ domains.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    domains.Service
	rmTrace.RoleManagerTracing
}

// New returns a new group service with tracing capabilities.
func New(svc domains.Service, tracer trace.Tracer) domains.Service {
	return &tracingMiddleware{tracer, svc, rmTrace.NewRoleManagerTracing("domain", svc, tracer)}
}

func (tm *tracingMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (domains.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "create_domain", trace.WithAttributes(
		attribute.String("name", d.Name),
	))
	defer span.End()
	return tm.svc.CreateDomain(ctx, session, d)
}

func (tm *tracingMiddleware) RetrieveDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "view_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.RetrieveDomain(ctx, session, id)
}

func (tm *tracingMiddleware) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "update_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.UpdateDomain(ctx, session, id, d)
}

func (tm *tracingMiddleware) EnableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "enable_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.EnableDomain(ctx, session, id)
}

func (tm *tracingMiddleware) DisableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "disable_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.DisableDomain(ctx, session, id)
}

func (tm *tracingMiddleware) FreezeDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	ctx, span := tm.tracer.Start(ctx, "freeze_domain", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()
	return tm.svc.FreezeDomain(ctx, session, id)
}

func (tm *tracingMiddleware) ListDomains(ctx context.Context, session authn.Session, p domains.Page) (domains.DomainsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "list_domains")
	defer span.End()
	return tm.svc.ListDomains(ctx, session, p)
}

func (tm *tracingMiddleware) DeleteUserFromDomains(ctx context.Context, id string) error {
	ctx, span := tm.tracer.Start(ctx, "delete_user_from_domains")
	defer span.End()
	return tm.svc.DeleteUserFromDomains(ctx, id)
}
