// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/groups"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ groups.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	gsvc   groups.Service
}

// New returns a new group service with tracing capabilities.
func New(gsvc groups.Service, tracer trace.Tracer) groups.Service {
	return &tracingMiddleware{tracer, gsvc}
}

// CreateGroup traces the "CreateGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) CreateGroup(ctx context.Context, session auth.Session, kind string, g groups.Group) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_create_group")
	defer span.End()

	return tm.gsvc.CreateGroup(ctx, session, kind, g)
}

// ViewGroup traces the "ViewGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ViewGroup(ctx context.Context, id string) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.ViewGroup(ctx, id)
}

// ViewGroupPerms traces the "ViewGroupPerms" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ViewGroupPerms(ctx context.Context, session auth.Session, id string) ([]string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.ViewGroupPerms(ctx, session, id)
}

// ListGroups traces the "ListGroups" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ListGroups(ctx context.Context, session auth.Session, memberKind, memberID string, gm groups.Page) (groups.Page, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_groups")
	defer span.End()

	return tm.gsvc.ListGroups(ctx, session, memberKind, memberID, gm)
}

// ListMembers traces the "ListMembers" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ListMembers(ctx context.Context, groupID, permission, memberKind string) (groups.MembersPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_members", trace.WithAttributes(attribute.String("groupID", groupID)))
	defer span.End()

	return tm.gsvc.ListMembers(ctx, groupID, permission, memberKind)
}

// UpdateGroup traces the "UpdateGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) UpdateGroup(ctx context.Context, session auth.Session, g groups.Group) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_group")
	defer span.End()

	return tm.gsvc.UpdateGroup(ctx, session, g)
}

// EnableGroup traces the "EnableGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) EnableGroup(ctx context.Context, session auth.Session, id string) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.EnableGroup(ctx, session, id)
}

// DisableGroup traces the "DisableGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) DisableGroup(ctx context.Context, session auth.Session, id string) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.DisableGroup(ctx, session, id)
}

// Assign traces the "Assign" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) Assign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_assign", trace.WithAttributes(attribute.String("id", groupID)))
	defer span.End()

	return tm.gsvc.Assign(ctx, session, groupID, relation, memberKind, memberIDs...)
}

// Unassign traces the "Unassign" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) Unassign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_unassign", trace.WithAttributes(attribute.String("id", groupID)))
	defer span.End()

	return tm.gsvc.Unassign(ctx, session, groupID, relation, memberKind, memberIDs...)
}

// DeleteGroup traces the "DeleteGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) DeleteGroup(ctx context.Context, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_delete_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.DeleteGroup(ctx, id)
}
