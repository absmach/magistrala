// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/absmach/magistrala/pkg/channels"
	entityRolesTrace "github.com/absmach/magistrala/pkg/entityroles/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ channels.Service = (*tracingMiddleware)(nil)

type tracingMiddleware struct {
	tracer trace.Tracer
	svc    channels.Service
	entityRolesTrace.RolesSvcTracingMiddleware
}

// New returns a new group service with tracing capabilities.
func New(svc channels.Service, tracer trace.Tracer) channels.Service {
	return &tracingMiddleware{tracer, svc, entityRolesTrace.NewRolesSvcTracingMiddleware("channels", svc, tracer)}
}

// CreateChannels traces the "CreateChannels" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) CreateChannels(ctx context.Context, token string, chs ...channels.Channel) ([]channels.Channel, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_create_channel")
	defer span.End()

	return tm.svc.CreateChannels(ctx, token, chs...)
}

// ViewChannel traces the "ViewChannel" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ViewChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_view_channel", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.svc.ViewChannel(ctx, token, id)
}

// ListChannels traces the "ListChannels" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) ListChannels(ctx context.Context, token string, pm channels.PageMetadata) (channels.Page, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_channels")
	defer span.End()
	return tm.svc.ListChannels(ctx, token, pm)
}

func (tm *tracingMiddleware) ListChannelsByThing(ctx context.Context, token, thingID string, pm channels.PageMetadata) (channels.Page, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_list_channels")
	defer span.End()
	return tm.svc.ListChannelsByThing(ctx, token, thingID, pm)
}

// UpdateChannel traces the "UpdateChannel" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) UpdateChannel(ctx context.Context, token string, cli channels.Channel) (channels.Channel, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_channel", trace.WithAttributes(attribute.String("id", cli.ID)))
	defer span.End()

	return tm.svc.UpdateChannel(ctx, token, cli)
}

// UpdateChannelTags traces the "UpdateChannelTags" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) UpdateChannelTags(ctx context.Context, token string, cli channels.Channel) (channels.Channel, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_update_channel_tags", trace.WithAttributes(
		attribute.String("id", cli.ID),
		attribute.StringSlice("tags", cli.Tags),
	))
	defer span.End()

	return tm.svc.UpdateChannelTags(ctx, token, cli)
}

// EnableChannel traces the "EnableChannel" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) EnableChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_enable_channel", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.EnableChannel(ctx, token, id)
}

// DisableChannel traces the "DisableChannel" operation of the wrapped policies.Service.
func (tm *tracingMiddleware) DisableChannel(ctx context.Context, token, id string) (channels.Channel, error) {
	ctx, span := tm.tracer.Start(ctx, "svc_disable_channel", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()

	return tm.svc.DisableChannel(ctx, token, id)
}

// DeleteChannel traces the "DeleteChannel" operation of the wrapped channels.Service.
func (tm *tracingMiddleware) RemoveChannel(ctx context.Context, token, id string) error {
	ctx, span := tm.tracer.Start(ctx, "delete_channel", trace.WithAttributes(attribute.String("id", id)))
	defer span.End()
	return tm.svc.RemoveChannel(ctx, token, id)
}

func (tm *tracingMiddleware) Connect(ctx context.Context, token string, chIDs, thIDs []string) error {
	ctx, span := tm.tracer.Start(ctx, "connect", trace.WithAttributes(
		attribute.StringSlice("channel_ids", chIDs),
		attribute.StringSlice("thing_ids", thIDs),
	))
	defer span.End()
	return tm.svc.Connect(ctx, token, chIDs, thIDs)
}

func (tm *tracingMiddleware) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) error {
	ctx, span := tm.tracer.Start(ctx, "disconnect", trace.WithAttributes(
		attribute.StringSlice("channel_ids", chIDs),
		attribute.StringSlice("thing_ids", thIDs),
	))
	defer span.End()
	return tm.svc.Disconnect(ctx, token, chIDs, thIDs)
}
