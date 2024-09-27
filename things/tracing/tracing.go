// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/things"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ things.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    things.Service
}

// New returns a new group service with tracing capabilities.
func New(svc things.Service, tracer trace.Tracer) things.Service {
	return &tracingMiddleware{tracer, svc}
}

// CreateThings traces the "CreateThings" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) CreateThings(ctx context.Context, session auth.Session, clis ...mgclients.Client) ([]mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_create_client")
	defer span.End()

	return tm.svc.CreateThings(ctx, session, clis...)
}

// ViewClient traces the "ViewClient" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ViewClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.svc.ViewClient(ctx, session, id)
}

// ViewClientPerms traces the "ViewClientPerms" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ViewClientPerms(ctx context.Context, session auth.Session, id string) ([]string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_client_permissions", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.svc.ViewClientPerms(ctx, session, id)
}

// ListClients traces the "ListClients" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ListClients(ctx context.Context, session auth.Session, reqUserID string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_clients")
	defer span.End()
	return tm.svc.ListClients(ctx, session, reqUserID, pm)
}

// UpdateClient traces the "UpdateClient" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) UpdateClient(ctx context.Context, session auth.Session, cli mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_name_and_metadata", trace.WithAttributes(attribute.String("id", cli.ID)))
	defer span.End()

	return tm.svc.UpdateClient(ctx, session, cli)
}

// UpdateClientTags traces the "UpdateClientTags" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) UpdateClientTags(ctx context.Context, session auth.Session, cli mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_tags", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateClientTags(ctx, session, cli)
}

// UpdateClientSecret traces the "UpdateClientSecret" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) UpdateClientSecret(ctx context.Context, session auth.Session, oldSecret, newSecret string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_secret")
	defer span.End()

	return tm.svc.UpdateClientSecret(ctx, session, oldSecret, newSecret)
}

// EnableClient traces the "EnableClient" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) EnableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.EnableClient(ctx, session, id)
}

// DisableClient traces the "DisableClient" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) DisableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.DisableClient(ctx, session, id)
}

// ListClientsByGroup traces the "ListClientsByGroup" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ListClientsByGroup(ctx context.Context, session auth.Session, groupID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_things_by_channel", trace.WithAttributes(attribute.String("groupID", groupID)))
	defer span.End()

	return tm.svc.ListClientsByGroup(ctx, session, groupID, pm)
}

// ListMemberships traces the "ListMemberships" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) Identify(ctx context.Context, key string) (string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_identify", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	return tm.svc.Identify(ctx, key)
}

// Share traces the "Share" operation of the wrapped things.Service.
func (tm *tracingMiddleware) Share(ctx context.Context, session auth.Session, id, relation string, userids ...string) error {
	ctx, span := tm.tracer.Start(ctx, "share", trace.WithAttributes(attribute.String("id", id), attribute.String("relation", relation), attribute.StringSlice("user_ids", userids)))
	defer span.End()
	return tm.svc.Share(ctx, session, id, relation, userids...)
}

// Unshare traces the "Unshare" operation of the wrapped things.Service.
func (tm *tracingMiddleware) Unshare(ctx context.Context, session auth.Session, id, relation string, userids ...string) error {
	ctx, span := tm.tracer.Start(ctx, "unshare", trace.WithAttributes(attribute.String("id", id), attribute.String("relation", relation), attribute.StringSlice("user_ids", userids)))
	defer span.End()
	return tm.svc.Unshare(ctx, session, id, relation, userids...)
}

// DeleteClient traces the "DeleteClient" operation of the wrapped things.Service.
func (tm *tracingMiddleware) DeleteClient(ctx context.Context, session auth.Session, id string) error {
	ctx, span := tm.tracer.Start(ctx, "delete_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.svc.DeleteClient(ctx, session, id)
}
