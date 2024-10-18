// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/channels"
	entityRolesAPI "github.com/absmach/magistrala/pkg/entityroles/api"
)

var _ channels.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    channels.Service
	entityRolesAPI.RolesSvcLoggingMiddleware
}

func LoggingMiddleware(svc channels.Service, logger *slog.Logger) channels.Service {
	rolesSvcLoggingMiddleware := entityRolesAPI.NewRolesSvcLoggingMiddleware("channels", svc, logger)
	return &loggingMiddleware{logger, svc, rolesSvcLoggingMiddleware}
}

func (lm *loggingMiddleware) CreateChannels(ctx context.Context, token string, clients ...channels.Channel) (cs []channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn(fmt.Sprintf("Create %d channels failed", len(clients)), args...)
			return
		}
		lm.logger.Info(fmt.Sprintf("Create %d channel completed successfully", len(clients)), args...)
	}(time.Now())
	return lm.svc.CreateChannels(ctx, token, clients...)
}

func (lm *loggingMiddleware) ViewChannel(ctx context.Context, token, id string) (c channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View channel failed", args...)
			return
		}
		lm.logger.Info("View channel completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewChannel(ctx, token, id)
}

func (lm *loggingMiddleware) ListChannels(ctx context.Context, token string, pm channels.PageMetadata) (cp channels.Page, err error) {
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
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List channels failed", args...)
			return
		}
		lm.logger.Info("List channels completed successfully", args...)
	}(time.Now())
	return lm.svc.ListChannels(ctx, token, pm)
}

func (lm *loggingMiddleware) ListChannelsByThing(ctx context.Context, token string, thingID string, pm channels.PageMetadata) (cp channels.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", thingID),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", cp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List channels by thing failed", args...)
			return
		}
		lm.logger.Info("List channels by thing completed successfully", args...)
	}(time.Now())
	return lm.svc.ListChannelsByThing(ctx, token, thingID, pm)
}

func (lm *loggingMiddleware) UpdateChannel(ctx context.Context, token string, client channels.Channel) (c channels.Channel, err error) {
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
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update channel failed", args...)
			return
		}
		lm.logger.Info("Update channel completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateChannel(ctx, token, client)
}

func (lm *loggingMiddleware) UpdateChannelTags(ctx context.Context, token string, client channels.Channel) (c channels.Channel, err error) {
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
	return lm.svc.UpdateChannelTags(ctx, token, client)
}

func (lm *loggingMiddleware) EnableChannel(ctx context.Context, token, id string) (c channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Enable channel failed", args...)
			return
		}
		lm.logger.Info("Enable channel completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableChannel(ctx, token, id)
}

func (lm *loggingMiddleware) DisableChannel(ctx context.Context, token, id string) (c channels.Channel, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Disable channel failed", args...)
			return
		}
		lm.logger.Info("Disable channel completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableChannel(ctx, token, id)
}

func (lm *loggingMiddleware) RemoveChannel(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete channel failed", args...)
			return
		}
		lm.logger.Info("Delete channel completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveChannel(ctx, token, id)
}

func (lm *loggingMiddleware) Connect(ctx context.Context, token string, chIDs, thIDs []string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Any("channel_ids", chIDs),
			slog.Any("thing_ids", thIDs),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Connect channels and things failed", args...)
			return
		}
		lm.logger.Info("Delete channels and things completed successfully", args...)
	}(time.Now())
	return lm.svc.Connect(ctx, token, chIDs, thIDs)
}

func (lm *loggingMiddleware) Disconnect(ctx context.Context, token string, chIDs, thIDs []string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Any("channel_ids", chIDs),
			slog.Any("thing_ids", thIDs),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Disconnect channels and things failed", args...)
			return
		}
		lm.logger.Info("Disconnect channels and things completed successfully", args...)
	}(time.Now())
	return lm.svc.Disconnect(ctx, token, chIDs, thIDs)
}
