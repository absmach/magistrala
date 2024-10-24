// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/authn"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
)

var _ clients.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    clients.Service
	rmMW.RoleManagerLoggingMiddleware
}

func LoggingMiddleware(svc clients.Service, logger *slog.Logger) clients.Service {
	return &loggingMiddleware{
		logger:                       logger,
		svc:                          svc,
		RoleManagerLoggingMiddleware: rmMW.NewRoleManagerLoggingMiddleware("clients", svc, logger),
	}
}

func (lm *loggingMiddleware) CreateClients(ctx context.Context, session authn.Session, clients ...clients.Client) (cs []clients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(fmt.Sprintf("Create %d clients failed", len(clients)), args...)
			return
		}
		lm.logger.Info(fmt.Sprintf("Create %d clients completed successfully", len(clients)), args...)
	}(time.Now())
	return lm.svc.CreateClients(ctx, session, clients...)
}

func (lm *loggingMiddleware) View(ctx context.Context, session authn.Session, id string) (c clients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("client",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("View client failed", args...)
			return
		}
		lm.logger.Info("View client completed successfully", args...)
	}(time.Now())
	return lm.svc.View(ctx, session, id)
}

func (lm *loggingMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm clients.Page) (cp clients.ClientsPage, err error) {
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
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List clients failed", args...)
			return
		}
		lm.logger.Info("List clients completed successfully", args...)
	}(time.Now())
	return lm.svc.ListClients(ctx, session, reqUserID, pm)
}

func (lm *loggingMiddleware) Update(ctx context.Context, session authn.Session, client clients.Client) (c clients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("client",
				slog.String("id", client.ID),
				slog.String("name", client.Name),
				slog.Any("metadata", client.Metadata),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update client failed", args...)
			return
		}
		lm.logger.Info("Update client completed successfully", args...)
	}(time.Now())
	return lm.svc.Update(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateTags(ctx context.Context, session authn.Session, client clients.Client) (c clients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("client",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
				slog.Any("tags", c.Tags),
			),
		}
		if err != nil {
			args := append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update client tags failed", args...)
			return
		}
		lm.logger.Info("Update client tags completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateTags(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (c clients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("client",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update client secret failed", args...)
			return
		}
		lm.logger.Info("Update client secret completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateSecret(ctx, session, oldSecret, newSecret)
}

func (lm *loggingMiddleware) Enable(ctx context.Context, session authn.Session, id string) (c clients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("client",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Enable client failed", args...)
			return
		}
		lm.logger.Info("Enable client completed successfully", args...)
	}(time.Now())
	return lm.svc.Enable(ctx, session, id)
}

func (lm *loggingMiddleware) Disable(ctx context.Context, session authn.Session, id string) (c clients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("client",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disable client failed", args...)
			return
		}
		lm.logger.Info("Disable client completed successfully", args...)
	}(time.Now())
	return lm.svc.Disable(ctx, session, id)
}

func (lm *loggingMiddleware) Delete(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Delete client failed", args...)
			return
		}
		lm.logger.Info("Delete client completed successfully", args...)
	}(time.Now())
	return lm.svc.Delete(ctx, session, id)
}

func (lm *loggingMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("parent_group_id", parentGroupID),
			slog.String("client_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Set parent group to client failed", args...)
			return
		}
		lm.logger.Info("Set parent group to client completed successfully", args...)
	}(time.Now())
	return lm.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (lm *loggingMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Remove parent group from client failed", args...)
			return
		}
		lm.logger.Info("Remove parent group from client completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveParentGroup(ctx, session, id)
}
