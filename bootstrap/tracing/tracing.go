// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
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
func (tm *tracingMiddleware) Add(ctx context.Context, token string, cfg bootstrap.Config) (bootstrap.Config, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_register_client", trace.WithAttributes(
		attribute.String("thing_id", cfg.ThingID),
		attribute.String("owner", cfg.Owner),
		attribute.String("name", cfg.Name),
		attribute.String("external_id", cfg.ExternalID),
		attribute.String("content", cfg.Content),
		attribute.String("state", cfg.State.String()),
	))
	defer span.End()

	return tm.svc.Add(ctx, token, cfg)
}

// View traces the "View" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) View(ctx context.Context, token, id string) (bootstrap.Config, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_client", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.View(ctx, token, id)
}

// Update traces the "Update" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) Update(ctx context.Context, token string, cfg bootstrap.Config) error {
	ctx, span := tm.tracer.Start(ctx, "svc_update_client", trace.WithAttributes(
		attribute.String("name", cfg.Name),
		attribute.String("content", cfg.Content),
		attribute.String("thing_id", cfg.ThingID),
		attribute.String("owner", cfg.Owner),
	))
	defer span.End()

	return tm.svc.Update(ctx, token, cfg)
}

// UpdateCert traces the "UpdateCert" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) UpdateCert(ctx context.Context, token, thingID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_cert", trace.WithAttributes(
		attribute.String("thing_id", thingID),
	))
	defer span.End()

	return tm.svc.UpdateCert(ctx, token, thingID, clientCert, clientKey, caCert)
}

// UpdateConnections traces the "UpdateConnections" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) UpdateConnections(ctx context.Context, token, id string, connections []string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_update_connections", trace.WithAttributes(
		attribute.String("id", id),
		attribute.StringSlice("connections", connections),
	))
	defer span.End()

	return tm.svc.UpdateConnections(ctx, token, id, connections)
}

// List traces the "List" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) List(ctx context.Context, token string, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_clients", trace.WithAttributes(
		attribute.Int64("offset", int64(offset)),
		attribute.Int64("limit", int64(limit)),
	))
	defer span.End()

	return tm.svc.List(ctx, token, filter, offset, limit)
}

// Remove traces the "Remove" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) Remove(ctx context.Context, token, id string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_remove_client", trace.WithAttributes(
		attribute.String("id", id),
	))
	defer span.End()

	return tm.svc.Remove(ctx, token, id)
}

// Bootstrap traces the "Bootstrap" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (bootstrap.Config, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_bootstrap_client", trace.WithAttributes(
		attribute.String("external_key", externalKey),
		attribute.String("external_id", externalID),
		attribute.Bool("secure", secure),
	))
	defer span.End()

	return tm.svc.Bootstrap(ctx, externalKey, externalID, secure)
}

// ChangeState traces the "ChangeState" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) ChangeState(ctx context.Context, token, id string, state bootstrap.State) error {
	ctx, span := tm.tracer.Start(ctx, "svc_change_state", trace.WithAttributes(
		attribute.String("id", id),
		attribute.String("state", state.String()),
	))
	defer span.End()

	return tm.svc.ChangeState(ctx, token, id, state)
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

// DisconnectThingHandler traces the "DisconnectThingHandler" operation of the wrapped bootstrap.Service.
func (tm *tracingMiddleware) DisconnectThingHandler(ctx context.Context, channelID, thingID string) error {
	ctx, span := tm.tracer.Start(ctx, "svc_disconnect_thing_handler", trace.WithAttributes(
		attribute.String("channel_id", channelID),
		attribute.String("thing_id", thingID),
	))
	defer span.End()

	return tm.svc.DisconnectThingHandler(ctx, channelID, thingID)
}
