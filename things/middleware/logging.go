// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
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

func (lm *loggingMiddleware) CreateClients(ctx context.Context, session authn.Session, clients ...things.Client) (cs []things.Client, err error) {
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
	return lm.svc.CreateClients(ctx, session, clients...)
}

func (lm *loggingMiddleware) View(ctx context.Context, session authn.Session, id string) (c things.Client, err error) {
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
	return lm.svc.View(ctx, session, id)
}

func (lm *loggingMiddleware) ViewPerms(ctx context.Context, session authn.Session, id string) (p []string, err error) {
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
	return lm.svc.ViewPerms(ctx, session, id)
}

func (lm *loggingMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm things.Page) (cp things.ClientsPage, err error) {
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

func (lm *loggingMiddleware) Update(ctx context.Context, session authn.Session, client things.Client) (c things.Client, err error) {
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
	return lm.svc.Update(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateTags(ctx context.Context, session authn.Session, client things.Client) (c things.Client, err error) {
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
	return lm.svc.UpdateTags(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (c things.Client, err error) {
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
	return lm.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
}

func (lm *loggingMiddleware) Enable(ctx context.Context, session authn.Session, id string) (c things.Client, err error) {
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
	return lm.svc.Enable(ctx, session, id)
}

func (lm *loggingMiddleware) Disable(ctx context.Context, session authn.Session, id string) (c things.Client, err error) {
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
	return lm.svc.Disable(ctx, session, id)
}

func (lm *loggingMiddleware) ListClientsByGroup(ctx context.Context, session authn.Session, channelID string, cp things.Page) (mp things.MembersPage, err error) {
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

func (lm *loggingMiddleware) Authorize(ctx context.Context, req things.AuthzReq) (id string, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("clientID", req.ClientID),
			slog.String("clientKey", req.ClientKey),
			slog.String("channelID", req.ChannelID),
			slog.String("permission", req.Permission),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Authorize failed", args...)
			return
		}
		lm.logger.Info("Authorize completed successfully", args...)
	}(time.Now())
	return lm.svc.Authorize(ctx, req)
}

func (lm *loggingMiddleware) Share(ctx context.Context, session authn.Session, id, relation string, userids ...string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", id),
			slog.Any("user_ids", userids),
			slog.String("relation", relation),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Share client failed", args...)
			return
		}
		lm.logger.Info("Share client completed successfully", args...)
	}(time.Now())
	return lm.svc.Share(ctx, session, id, relation, userids...)
}

func (lm *loggingMiddleware) Unshare(ctx context.Context, session authn.Session, id, relation string, userids ...string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", id),
			slog.Any("user_ids", userids),
			slog.String("relation", relation),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Unshare client failed", args...)
			return
		}
		lm.logger.Info("Unshare client completed successfully", args...)
	}(time.Now())
	return lm.svc.Unshare(ctx, session, id, relation, userids...)
}

func (lm *loggingMiddleware) Delete(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete client failed", args...)
			return
		}
		lm.logger.Info("Delete client completed successfully", args...)
	}(time.Now())
	return lm.svc.Delete(ctx, session, id)
}
