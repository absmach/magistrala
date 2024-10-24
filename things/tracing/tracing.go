// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
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

// CreateClients traces the "CreateClients" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) CreateClients(ctx context.Context, session authn.Session, cli ...things.Client) ([]things.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_create_client")
	defer span.End()

	return tm.svc.CreateClients(ctx, session, cli...)
}

// View traces the "View" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) View(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.svc.View(ctx, session, id)
}

// ViewPerms traces the "ViewPerms" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ViewPerms(ctx context.Context, session authn.Session, id string) ([]string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_client_permissions", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.svc.ViewPerms(ctx, session, id)
}

// ListClients traces the "ListClients" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm things.Page) (things.ClientsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_clients")
	defer span.End()
	return tm.svc.ListClients(ctx, session, reqUserID, pm)
}

// Update traces the "Update" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) Update(ctx context.Context, session authn.Session, cli things.Client) (things.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client", trace.WithAttributes(attribute.String("id", cli.ID)))
	defer span.End()

	return tm.svc.Update(ctx, session, cli)
}

// UpdateTags traces the "UpdateTags" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) UpdateTags(ctx context.Context, session authn.Session, cli things.Client) (things.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_tags", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateTags(ctx, session, cli)
}

// UpdateSecret traces the "UpdateSecret" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (things.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_secret")
	defer span.End()

	return tm.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
}

// Enable traces the "Enable" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) Enable(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.Enable(ctx, session, id)
}

// Disable traces the "Disable" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) Disable(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.Disable(ctx, session, id)
}

// ListClientsByGroup traces the "ListClientsByGroup" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ListClientsByGroup(ctx context.Context, session authn.Session, groupID string, pm things.Page) (things.MembersPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_clients_by_channel", trace.WithAttributes(attribute.String("groupID", groupID)))
	defer span.End()

	return tm.svc.ListClientsByGroup(ctx, session, groupID, pm)
}

// ListMemberships traces the "ListMemberships" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) Identify(ctx context.Context, key string) (string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_identify", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	return tm.svc.Identify(ctx, key)
}

// Authorize traces the "Authorize" operation of the wrapped things.Service.
func (tm *tracingMiddleware) Authorize(ctx context.Context, req things.AuthzReq) (string, error) {
	ctx, span := tm.tracer.Start(ctx, "connect", trace.WithAttributes(attribute.String("thingKey", req.ClientKey), attribute.String("channelID", req.ChannelID)))
	defer span.End()

	return tm.svc.Authorize(ctx, req)
}

// Share traces the "Share" operation of the wrapped things.Service.
func (tm *tracingMiddleware) Share(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	ctx, span := tm.tracer.Start(ctx, "share", trace.WithAttributes(attribute.String("id", id), attribute.String("relation", relation), attribute.StringSlice("user_ids", userids)))
	defer span.End()
	return tm.svc.Share(ctx, session, id, relation, userids...)
}

// Unshare traces the "Unshare" operation of the wrapped things.Service.
func (tm *tracingMiddleware) Unshare(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	ctx, span := tm.tracer.Start(ctx, "unshare", trace.WithAttributes(attribute.String("id", id), attribute.String("relation", relation), attribute.StringSlice("user_ids", userids)))
	defer span.End()
	return tm.svc.Unshare(ctx, session, id, relation, userids...)
}

// Delete traces the "Delete" operation of the wrapped things.Service.
func (tm *tracingMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	ctx, span := tm.tracer.Start(ctx, "delete_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.svc.Delete(ctx, session, id)
}
