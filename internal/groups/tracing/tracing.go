// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala/pkg/authn"
	entityRolesTracing "github.com/absmach/magistrala/pkg/entityroles/tracing"
	"github.com/absmach/magistrala/pkg/groups"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ groups.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	gsvc   groups.Service
	entityRolesTracing.RolesSvcTracingMiddleware
}

// New returns a new group service with tracing capabilities.
func New(gsvc groups.Service, tracer trace.Tracer) groups.Service {
	t := entityRolesTracing.NewRolesSvcTracingMiddleware("group", gsvc, tracer)
	return &tracingMiddleware{tracer, gsvc, t}
}

// CreateGroup traces the "CreateGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) CreateGroup(ctx context.Context, session authn.Session, g groups.Group) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_create_group")
	defer span.End()

	return tm.gsvc.CreateGroup(ctx, session, g)
}

// ViewGroup traces the "ViewGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ViewGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.ViewGroup(ctx, session, id)
}

// ListGroups traces the "ListGroups" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) ListGroups(ctx context.Context, session authn.Session, pm groups.PageMeta) (groups.Page, error) {
	attr := []attribute.KeyValue{
		attribute.String("name", pm.Name),
		attribute.String("tag", pm.Tag),
		attribute.String("status", pm.Status.String()),
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
	}
	for k, v := range pm.Metadata {
		attr = append(attr, attribute.String(k, fmt.Sprintf("%v", v)))
	}
	ctx, span := tm.tracer.Start(ctx, "svc_list_groups", trace.WithAttributes(attr...))
	defer span.End()

	return tm.gsvc.ListGroups(ctx, session, pm)
}

// UpdateGroup traces the "UpdateGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) UpdateGroup(ctx context.Context, session authn.Session, g groups.Group) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_group")
	defer span.End()

	return tm.gsvc.UpdateGroup(ctx, session, g)
}

// EnableGroup traces the "EnableGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) EnableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.EnableGroup(ctx, session, id)
}

// DisableGroup traces the "DisableGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) DisableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.DisableGroup(ctx, session, id)
}

func (tm *tracingMiddleware) RetrieveGroupHierarchy(ctx context.Context, session authn.Session, id string, hm groups.HierarchyPageMeta) (groups.HierarchyPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_group_hierarchy",
		trace.WithAttributes(
			attribute.String("id", id),
			attribute.Int64("level", int64(hm.Level)),
			attribute.Int64("direction", hm.Direction),
			attribute.Bool("tree", hm.Tree),
		))
	defer span.End()

	return tm.gsvc.RetrieveGroupHierarchy(ctx, session, id, hm)
}

func (tm *tracingMiddleware) AddParentGroup(ctx context.Context, session authn.Session, id, parentID string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_add_parent_group",
		trace.WithAttributes(
			attribute.String("id", id),
			attribute.String("parent_id", parentID),
		))
	defer span.End()
	return tm.gsvc.AddParentGroup(ctx, session, id, parentID)
}

func (tm *tracingMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_remove_parent_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.gsvc.RemoveParentGroup(ctx, session, id)
}

func (tm *tracingMiddleware) ViewParentGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_parent_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.ViewParentGroup(ctx, session, id)
}

func (tm *tracingMiddleware) AddChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_add_children_groups",
		trace.WithAttributes(
			attribute.String("id", id),
			attribute.StringSlice("children_group_ids", childrenGroupIDs),
		))

	defer span.End()
	return tm.gsvc.AddChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (tm *tracingMiddleware) RemoveChildrenGroups(ctx context.Context, session authn.Session, id string, childrenGroupIDs []string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_remove_children_groups",
		trace.WithAttributes(
			attribute.String("id", id),
			attribute.StringSlice("children_group_ids", childrenGroupIDs),
		))
	defer span.End()
	return tm.gsvc.RemoveChildrenGroups(ctx, session, id, childrenGroupIDs)
}

func (tm *tracingMiddleware) RemoveAllChildrenGroups(ctx context.Context, session authn.Session, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_remove_all_children_groups", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.gsvc.RemoveAllChildrenGroups(ctx, session, id)
}

func (tm *tracingMiddleware) ListChildrenGroups(ctx context.Context, session authn.Session, id string, pm groups.PageMeta) (groups.Page, error) {
	attr := []attribute.KeyValue{
		attribute.String("id", id),
		attribute.String("name", pm.Name),
		attribute.String("tag", pm.Tag),
		attribute.String("status", pm.Status.String()),
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
	}
	for k, v := range pm.Metadata {
		attr = append(attr, attribute.String(k, fmt.Sprintf("%v", v)))
	}
	ctx, span := tm.tracer.Start(ctx, "svc_list_children_groups", trace.WithAttributes(attr...))
	defer span.End()
	return tm.gsvc.ListChildrenGroups(ctx, session, id, pm)
}

// DeleteGroup traces the "DeleteGroup" operation of the wrapped groups.Service.
func (tm *tracingMiddleware) DeleteGroup(ctx context.Context, session authn.Session, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_delete_group", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.gsvc.DeleteGroup(ctx, session, id)
}
