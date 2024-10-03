// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/auth"
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
func (tm *tracingMiddleware) RegisterClient(ctx context.Context, session auth.Session, client mgclients.Client, selfRegister bool) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_register_client", trace.WithAttributes(attribute.String("identity", client.Credentials.Identity)))
	defer span.End()

	return tm.svc.RegisterClient(ctx, session, client, selfRegister)
}

// IssueToken traces the "IssueToken" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) IssueToken(ctx context.Context, identity, secret, domainID string) (*magistrala.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_issue_token", trace.WithAttributes(attribute.String("identity", identity)))
	defer span.End()

	return tm.svc.IssueToken(ctx, identity, secret, domainID)
}

// RefreshToken traces the "RefreshToken" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) RefreshToken(ctx context.Context, session auth.Session, refreshToken, domainID string) (*magistrala.Token, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_refresh_token", trace.WithAttributes(attribute.String("refresh_token", refreshToken)))
	defer span.End()

	return tm.svc.RefreshToken(ctx, session, refreshToken, domainID)
}

// ViewClient traces the "ViewClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ViewClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.ViewClient(ctx, session, id)
}

// ListClients traces the "ListClients" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ListClients(ctx context.Context, session auth.Session, pm mgclients.Page) (mgclients.ClientsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_clients", trace.WithAttributes(
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
		attribute.String("direction", pm.Dir),
		attribute.String("order", pm.Order),
	))

	defer span.End()

	return tm.svc.ListClients(ctx, session, pm)
}

// SearchUsers traces the "SearchUsers" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) SearchUsers(ctx context.Context, pm mgclients.Page) (mgclients.ClientsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_search_clients", trace.WithAttributes(
		attribute.Int64("offset", int64(pm.Offset)),
		attribute.Int64("limit", int64(pm.Limit)),
		attribute.String("direction", pm.Dir),
		attribute.String("order", pm.Order),
	))
	defer span.End()

	return tm.svc.SearchUsers(ctx, pm)
}

// UpdateClient traces the "UpdateClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClient(ctx context.Context, session auth.Session, cli mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_name_and_metadata", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.String("name", cli.Name),
	))
	defer span.End()

	return tm.svc.UpdateClient(ctx, session, cli)
}

// UpdateClientTags traces the "UpdateClientTags" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientTags(ctx context.Context, session auth.Session, cli mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_tags", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateClientTags(ctx, session, cli)
}

// UpdateClientIdentity traces the "UpdateClientIdentity" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientIdentity(ctx context.Context, session auth.Session, id, identity string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_identity", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("identity", identity),
	))
	defer span.End()

	return tm.svc.UpdateClientIdentity(ctx, session, id, identity)
}

// UpdateClientSecret traces the "UpdateClientSecret" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientSecret(ctx context.Context, session auth.Session, oldSecret, newSecret string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_secret")
	defer span.End()

	return tm.svc.UpdateClientSecret(ctx, session, oldSecret, newSecret)
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
func (tm *tracingMiddleware) ResetSecret(ctx context.Context, session auth.Session, secret string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_reset_secret")
	defer span.End()

	return tm.svc.ResetSecret(ctx, session, secret)
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
func (tm *tracingMiddleware) ViewProfile(ctx context.Context, session auth.Session) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_profile")
	defer span.End()

	return tm.svc.ViewProfile(ctx, session)
}

// UpdateClientRole traces the "UpdateClientRole" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) UpdateClientRole(ctx context.Context, session auth.Session, cli mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client_role", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateClientRole(ctx, session, cli)
}

// EnableClient traces the "EnableClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) EnableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.EnableClient(ctx, session, id)
}

// DisableClient traces the "DisableClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) DisableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.DisableClient(ctx, session, id)
}

// ListMembers traces the "ListMembers" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) ListMembers(ctx context.Context, session auth.Session, objectKind, objectID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_members", trace.WithAttributes(attribute.String("object_kind", objectKind)), trace.WithAttributes(attribute.String("object_id", objectID)))
	defer span.End()

	return tm.svc.ListMembers(ctx, session, objectKind, objectID, pm)
}

// Identify traces the "Identify" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) Identify(ctx context.Context, session auth.Session) (string, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_identify", trace.WithAttributes(attribute.String("user_id", session.UserID)))
	defer span.End()

	return tm.svc.Identify(ctx, session)
}

// OAuthCallback traces the "OAuthCallback" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) OAuthCallback(ctx context.Context, client mgclients.Client) (mgclients.Client, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_oauth_callback", trace.WithAttributes(
		attribute.String("client_id", client.ID),
	))
	defer span.End()

	return tm.svc.OAuthCallback(ctx, client)
}

// DeleteClient traces the "DeleteClient" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) DeleteClient(ctx context.Context, session auth.Session, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_delete_client", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.DeleteClient(ctx, session, id)
}

// OAuthAddClientPolicy traces the "OAuthAddClientPolicy" operation of the wrapped clients.Service.
func (tm *tracingMiddleware) OAuthAddClientPolicy(ctx context.Context, client mgclients.Client) error {
	ctx, span := tm.tracer.Start(ctx, "svc_add_client_policy", trace.WithAttributes(
		attribute.String("id", client.ID),
	))
	defer span.End()

	return tm.svc.OAuthAddClientPolicy(ctx, client)
}
