// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/users"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ users.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    users.Service
}

// New returns a new group service with tracing capabilities.
func New(svc users.Service, tracer trace.Tracer) users.Service {
	return &tracingMiddleware{tracer, svc}
}

// RegisterClient traces the "RegisterClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) RegisterClient(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_register_client", trace.WithAttributes(attribute.String("identity", client.Credentials.Identity)))
	defer span.End()

	return tm.svc.RegisterClient(ctx, token, client)
}

// IssueToken traces the "IssueToken" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_issue_token", trace.WithAttributes(attribute.String("identity", identity)))
	defer span.End()

	return tm.svc.IssueToken(ctx, identity, secret, domainID)
}

// RefreshToken traces the "RefreshToken" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) RefreshToken(ctx context.Context, accessToken, domainID string) (*magistrala.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_refresh_token", trace.WithAttributes(attribute.String("access_token", accessToken)))
	defer span.End()

	return tm.svc.RefreshToken(ctx, accessToken, domainID)
}

// ViewClient traces the "ViewClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ViewClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.ViewClient(ctx, token, id)
}

// ListClients traces the "ListClients" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ListClients(ctx context.Context, token string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_clients", trace.WithAttributes(
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
		attribute.String("direction", pm.Dir),
		attribute.String("order", pm.Order),
	))

	defer span.End()

	return tm.svc.ListClients(ctx, token, pm)
}

// UpdateClient traces the "UpdateClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClient(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_name_and_metadata", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.String("name", cli.Name),
	))
	defer span.End()

	return tm.svc.UpdateClient(ctx, token, cli)
}

// UpdateClientTags traces the "UpdateClientTags" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientTags(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_tags", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateClientTags(ctx, token, cli)
}

// UpdateClientIdentity traces the "UpdateClientIdentity" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientIdentity(ctx context.Context, token, id, identity string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_identity", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("identity", identity),
	))
	defer span.End()

	return tm.svc.UpdateClientIdentity(ctx, token, id, identity)
}

// UpdateClientSecret traces the "UpdateClientSecret" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientSecret(ctx context.Context, token, oldSecret, newSecret string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_secret")
	defer span.End()

	return tm.svc.UpdateClientSecret(ctx, token, oldSecret, newSecret)
}

// GenerateResetToken traces the "GenerateResetToken" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) GenerateResetToken(ctx context.Context, email, host string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_generate_reset_token", trace.WithAttributes(
		attribute.String("email", email),
		attribute.String("host", host),
	))
	defer span.End()

	return tm.svc.GenerateResetToken(ctx, email, host)
}

// ResetSecret traces the "ResetSecret" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ResetSecret(ctx context.Context, token, secret string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_reset_secret")
	defer span.End()

	return tm.svc.ResetSecret(ctx, token, secret)
}

// SendPasswordReset traces the "SendPasswordReset" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) SendPasswordReset(ctx context.Context, host, email, user, token string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_send_password_reset", trace.WithAttributes(
		attribute.String("email", email),
		attribute.String("user", user),
	))
	defer span.End()

	return tm.svc.SendPasswordReset(ctx, host, email, user, token)
}

// ViewProfile traces the "ViewProfile" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ViewProfile(ctx context.Context, token string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_profile")
	defer span.End()

	return tm.svc.ViewProfile(ctx, token)
}

// UpdateClientRole traces the "UpdateClientRole" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientRole(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_role", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateClientRole(ctx, token, cli)
}

// EnableClient traces the "EnableClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) EnableClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.EnableClient(ctx, token, id)
}

// DisableClient traces the "DisableClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) DisableClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.DisableClient(ctx, token, id)
}

// ListMembers traces the "ListMembers" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ListMembers(ctx context.Context, token, objectKind, objectID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_members", trace.WithAttributes(attribute.String("object_kind", objectKind)), trace.WithAttributes(attribute.String("object_id", objectID)))
	defer span.End()

	return tm.svc.ListMembers(ctx, token, objectKind, objectID, pm)
}

// Identify traces the "Identify" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) Identify(ctx context.Context, token string) (string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_identify", trace.WithAttributes(attribute.String("token", token)))
	defer span.End()

	return tm.svc.Identify(ctx, token)
}
