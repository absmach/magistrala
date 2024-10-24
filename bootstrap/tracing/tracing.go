// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ bootstrap.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    bootstrap.Service
}

// New returns a new bootstrap service with tracing capabilities.
func New(svc bootstrap.Service, tracer trace.Tracer) bootstrap.Service {
	return &tracingMiddleware{tracer, svc}
}

// Add traces the "Add" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) Add(ctx context.Context, session mgauthn.Session, token string, cfg bootstrap.Config) (bootstrap.Config, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_register_user", trace.WithAttributes(
		attribute.String("client_id", cfg.ClientID),
		attribute.String("domain_id ", cfg.DomainID),
		attribute.String("name", cfg.Name),
		attribute.String("external_id", cfg.ExternalID),
		attribute.String("content", cfg.Content),
		attribute.String("state", cfg.State.String()),
	))
	defer span.End()

	return tm.svc.Add(ctx, session, token, cfg)
}

// View traces the "View" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) View(ctx context.Context, session mgauthn.Session, id string) (bootstrap.Config, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_user", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.View(ctx, session, id)
}

// Update traces the "Update" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) Update(ctx context.Context, session mgauthn.Session, cfg bootstrap.Config) error {
	ctx, span := tm.tracer.Start(ctx, "svc_update_user", trace.WithAttributes(
		attribute.String("name", cfg.Name),
		attribute.String("content", cfg.Content),
		attribute.String("client_id", cfg.ClientID),
		attribute.String("domain_id ", cfg.DomainID),
	))
	defer span.End()

	return tm.svc.Update(ctx, session, cfg)
}

// UpdateCert traces the "UpdateCert" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) UpdateCert(ctx context.Context, session mgauthn.Session, clientID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_cert", trace.WithAttributes(
		attribute.String("client_id", clientID),
	))
	defer span.End()

	return tm.svc.UpdateCert(ctx, session, clientID, clientCert, clientKey, caCert)
}

// UpdateConnections traces the "UpdateConnections" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) UpdateConnections(ctx context.Context, session mgauthn.Session, token, id string, connections []string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_update_connections", trace.WithAttributes(
		attribute.String("id", id),
		attribute.StringSlice("connections", connections),
	))
	defer span.End()

	return tm.svc.UpdateConnections(ctx, session, token, id, connections)
}

// List traces the "List" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) List(ctx context.Context, session mgauthn.Session, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_users", trace.WithAttributes(
		attribute.Int64("offset", int64(offset)),
		attribute.Int64("limit", int64(limit)),
	))
	defer span.End()

	return tm.svc.List(ctx, session, filter, offset, limit)
}

// Remove traces the "Remove" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) Remove(ctx context.Context, session mgauthn.Session, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_remove_user", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.Remove(ctx, session, id)
}

// Bootstrap traces the "Bootstrap" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (bootstrap.Config, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_bootstrap_user", trace.WithAttributes(
		attribute.String("external_key", externalKey),
		attribute.String("external_id", externalID),
		attribute.Bool("secure", secure),
	))
	defer span.End()

	return tm.svc.Bootstrap(ctx, externalKey, externalID, secure)
}

// ChangeState traces the "ChangeState" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) ChangeState(ctx context.Context, session mgauthn.Session, token, id string, state bootstrap.State) error {
	ctx, span := tm.tracer.Start(ctx, "svc_change_state", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("state", state.String()),
	))
	defer span.End()

	return tm.svc.ChangeState(ctx, session, token, id, state)
}

// UpdateChannelHandler traces the "UpdateChannelHandler" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) UpdateChannelHandler(ctx context.Context, channel bootstrap.Channel) error {
	ctx, span := tm.tracer.Start(ctx, "svc_update_channel_handler", trace.WithAttributes(
		attribute.String("id", channel.ID),
		attribute.String("name", channel.Name),
		attribute.String("description", channel.Description),
	))
	defer span.End()

	return tm.svc.UpdateChannelHandler(ctx, channel)
}

// RemoveConfigHandler traces the "RemoveConfigHandler" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) RemoveConfigHandler(ctx context.Context, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_remove_config_handler", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.RemoveConfigHandler(ctx, id)
}

// RemoveChannelHandler traces the "RemoveChannelHandler" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) RemoveChannelHandler(ctx context.Context, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_remove_channel_handler", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.RemoveChannelHandler(ctx, id)
}

// ConnectClientHandler traces the "ConnectClientHandler" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) ConnectClientHandler(ctx context.Context, channelID, clientID string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_connect_client_handler", trace.WithAttributes(
		attribute.String("channel_id", channelID),
		attribute.String("client_id", clientID),
	))
	defer span.End()

	return tm.svc.ConnectClientHandler(ctx, channelID, clientID)
}

// DisconnectClientHandler traces the "DisconnectClientHandler" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) DisconnectClientHandler(ctx context.Context, channelID, clientID string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_disconnect_client_handler", trace.WithAttributes(
		attribute.String("channel_id", channelID),
		attribute.String("client_id", clientID),
	))
	defer span.End()

	return tm.svc.DisconnectClientHandler(ctx, channelID, clientID)
}
