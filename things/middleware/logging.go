// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/things"
)

var _ things.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    things.Service
}

func LoggingMiddleware(svc things.Service, logger *slog.Logger) things.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateThings(ctx context.Context, session auth.Session, clients ...mgclients.Client) (cs []mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn(fmt.Sprintf("Create %d things failed", len(clients)), args...)
			return
		}
		lm.logger.Info(fmt.Sprintf("Create %d things completed successfully", len(clients)), args...)
	}(time.Now())
	return lm.svc.CreateThings(ctx, session, clients...)
}

func (lm *loggingMiddleware) ViewClient(ctx context.Context, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View thing failed", args...)
			return
		}
		lm.logger.Info("View thing completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewClient(ctx, id)
}

func (lm *loggingMiddleware) ViewClientPerms(ctx context.Context, session auth.Session, id string) (p []string, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View thing permissions failed", args...)
			return
		}
		lm.logger.Info("View thing permissions completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewClientPerms(ctx, session, id)
}

func (lm *loggingMiddleware) ListClients(ctx context.Context, session auth.Session, reqUserID string, pm mgclients.Page) (cp mgclients.ClientsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", reqUserID),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", cp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List things failed", args...)
			return
		}
		lm.logger.Info("List things completed successfully", args...)
	}(time.Now())
	return lm.svc.ListClients(ctx, session, reqUserID, pm)
}

func (lm *loggingMiddleware) UpdateClient(ctx context.Context, session auth.Session, client mgclients.Client) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", client.ID),
				slog.String("name", client.Name),
				slog.Any("metadata", client.Metadata),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update thing failed", args...)
			return
		}
		lm.logger.Info("Update thing completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateClient(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateClientTags(ctx context.Context, session auth.Session, client mgclients.Client) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
				slog.Any("tags", c.Tags),
			),
		}
		if err != nil {
			args := append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update thing tags failed", args...)
			return
		}
		lm.logger.Info("Update thing tags completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateClientTags(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateClientSecret(ctx context.Context, session auth.Session, oldSecret, newSecret string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update thing secret failed", args...)
			return
		}
		lm.logger.Info("Update thing secret completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateClientSecret(ctx, session, oldSecret, newSecret)
}

func (lm *loggingMiddleware) EnableClient(ctx context.Context, session auth.Session, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Enable thing failed", args...)
			return
		}
		lm.logger.Info("Enable thing completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableClient(ctx, session, id)
}

func (lm *loggingMiddleware) DisableClient(ctx context.Context, session auth.Session, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Disable thing failed", args...)
			return
		}
		lm.logger.Info("Disable thing completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableClient(ctx, session, id)
}

func (lm *loggingMiddleware) ListClientsByGroup(ctx context.Context, session auth.Session, channelID string, cp mgclients.Page) (mp mgclients.MembersPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", channelID),
			slog.Group("page",
				slog.Uint64("offset", cp.Offset),
				slog.Uint64("limit", cp.Limit),
				slog.Uint64("total", mp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List things by group failed", args...)
			return
		}
		lm.logger.Info("List things by group completed successfully", args...)
	}(time.Now())
	return lm.svc.ListClientsByGroup(ctx, session, channelID, cp)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, key string) (id string, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Identify thing failed", args...)
			return
		}
		lm.logger.Info("Identify thing completed successfully", args...)
	}(time.Now())
	return lm.svc.Identify(ctx, key)
}

func (lm *loggingMiddleware) Share(ctx context.Context, session auth.Session, id, relation string, userids ...string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", id),
			slog.Any("user_ids", userids),
			slog.String("relation", relation),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Share thing failed", args...)
			return
		}
		lm.logger.Info("Share thing completed successfully", args...)
	}(time.Now())
	return lm.svc.Share(ctx, session, id, relation, userids...)
}

func (lm *loggingMiddleware) Unshare(ctx context.Context, session auth.Session, id, relation string, userids ...string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", id),
			slog.Any("user_ids", userids),
			slog.String("relation", relation),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Unshare thing failed", args...)
			return
		}
		lm.logger.Info("Unshare thing completed successfully", args...)
	}(time.Now())
	return lm.svc.Unshare(ctx, session, id, relation, userids...)
}

func (lm *loggingMiddleware) DeleteClient(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete thing failed", args...)
			return
		}
		lm.logger.Info("Delete thing completed successfully", args...)
	}(time.Now())
	return lm.svc.DeleteClient(ctx, id)
}
