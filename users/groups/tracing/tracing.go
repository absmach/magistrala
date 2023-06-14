// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/users/groups"
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
func (tm *tracingMiddleware) CreateGroup(ctx context.Context, token string, g mfgroups.Group) (mfgroups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_create_group")
	defer span.End()

	return tm.gsvc.CreateGroup(ctx, token, g)
}

// ViewGroup traces the "ViewGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ViewGroup(ctx context.Context, token string, id string) (mfgroups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.ViewGroup(ctx, token, id)
}

// ListGroups traces the "ListGroups" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ListGroups(ctx context.Context, token string, gm mfgroups.GroupsPage) (mfgroups.GroupsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_groups")
	defer span.End()

	return tm.gsvc.ListGroups(ctx, token, gm)
}

// ListMemberships traces the "ListMemberships" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ListMemberships(ctx context.Context, token, clientID string, gm mfgroups.GroupsPage) (mfgroups.MembershipsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_memberships", trace.WithAttributes(attribute.String("clientID", clientID)))
	defer span.End()

	return tm.gsvc.ListMemberships(ctx, token, clientID, gm)
}

// UpdateGroup traces the "UpdateGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) UpdateGroup(ctx context.Context, token string, g mfgroups.Group) (mfgroups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_group")
	defer span.End()

	return tm.gsvc.UpdateGroup(ctx, token, g)
}

// EnableGroup traces the "EnableGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) EnableGroup(ctx context.Context, token, id string) (mfgroups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.EnableGroup(ctx, token, id)
}

// DisableGroup traces the "DisableGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) DisableGroup(ctx context.Context, token, id string) (mfgroups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.DisableGroup(ctx, token, id)
}
