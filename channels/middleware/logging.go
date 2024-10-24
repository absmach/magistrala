// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
)

var _ channels.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    channels.Service
	rmMW.RoleManagerLoggingMiddleware
}

func LoggingMiddleware(svc channels.Service, logger *slog.Logger) channels.Service {
	return &loggingMiddleware{logger, svc, rmMW.NewRoleManagerLoggingMiddleware("channels", svc, logger)}
}

func (lm *loggingMiddleware) CreateChannels(ctx context.Context, session authn.Session, clients ...channels.Channel) (cs []channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(fmt.Sprintf("Create %d channels failed", len(clients)), args...)
			return
		}
		lm.logger.Info(fmt.Sprintf("Create %d channel completed successfully", len(clients)), args...)
	}(time.Now())
	return lm.svc.CreateChannels(ctx, session, clients...)
}

func (lm *loggingMiddleware) ViewChannel(ctx context.Context, session authn.Session, id string) (c channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("View channel failed", args...)
			return
		}
		lm.logger.Info("View channel completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewChannel(ctx, session, id)
}

func (lm *loggingMiddleware) ListChannels(ctx context.Context, session authn.Session, pm channels.PageMetadata) (cp channels.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", cp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List channels failed", args...)
			return
		}
		lm.logger.Info("List channels completed successfully", args...)
	}(time.Now())
	return lm.svc.ListChannels(ctx, session, pm)
}

func (lm *loggingMiddleware) ListChannelsByClient(ctx context.Context, session authn.Session, clientID string, pm channels.PageMetadata) (cp channels.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", clientID),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", cp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List channels by client failed", args...)
			return
		}
		lm.logger.Info("List channels by client completed successfully", args...)
	}(time.Now())
	return lm.svc.ListChannelsByClient(ctx, session, clientID, pm)
}

func (lm *loggingMiddleware) UpdateChannel(ctx context.Context, session authn.Session, client channels.Channel) (c channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", client.ID),
				slog.String("name", client.Name),
				slog.Any("metadata", client.Metadata),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update channel failed", args...)
			return
		}
		lm.logger.Info("Update channel completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateChannel(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateChannelTags(ctx context.Context, session authn.Session, client channels.Channel) (c channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
				slog.Any("tags", c.Tags),
			),
		}
		if err != nil {
			args := append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update channel tags failed", args...)
			return
		}
		lm.logger.Info("Update channel tags completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateChannelTags(ctx, session, client)
}

func (lm *loggingMiddleware) EnableChannel(ctx context.Context, session authn.Session, id string) (c channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Enable channel failed", args...)
			return
		}
		lm.logger.Info("Enable channel completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableChannel(ctx, session, id)
}

func (lm *loggingMiddleware) DisableChannel(ctx context.Context, session authn.Session, id string) (c channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disable channel failed", args...)
			return
		}
		lm.logger.Info("Disable channel completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableChannel(ctx, session, id)
}

func (lm *loggingMiddleware) RemoveChannel(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Delete channel failed", args...)
			return
		}
		lm.logger.Info("Delete channel completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveChannel(ctx, session, id)
}

func (lm *loggingMiddleware) Connect(ctx context.Context, session authn.Session, chIDs, clIDs []string, connTypes []connections.ConnType) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Any("channel_ids", chIDs),
			slog.Any("client_ids", clIDs),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Connect channels and clients failed", args...)
			return
		}
		lm.logger.Info("Connect channels and clients completed successfully", args...)
	}(time.Now())
	return lm.svc.Connect(ctx, session, chIDs, clIDs, connTypes)
}

func (lm *loggingMiddleware) Disconnect(ctx context.Context, session authn.Session, chIDs, clIDs []string, connTypes []connections.ConnType) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Any("channel_ids", chIDs),
			slog.Any("client_ids", clIDs),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disconnect channels and clients failed", args...)
			return
		}
		lm.logger.Info("Disconnect channels and clients completed successfully", args...)
	}(time.Now())
	return lm.svc.Disconnect(ctx, session, chIDs, clIDs, connTypes)
}

func (lm *loggingMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("parent_group_id", parentGroupID),
			slog.String("channel_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Set parent group to channel failed", args...)
			return
		}
		lm.logger.Info("Set parent group to channel completed successfully", args...)
	}(time.Now())
	return lm.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (lm *loggingMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Remove parent group from channel failed", args...)
			return
		}
		lm.logger.Info("Remove parent group from channel completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveParentGroup(ctx, session, id)
}
