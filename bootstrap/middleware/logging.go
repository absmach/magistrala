// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
)

var _ bootstrap.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    bootstrap.Service
}

// LoggingMiddleware adds logging facilities to the bootstrap service.
func LoggingMiddleware(svc bootstrap.Service, logger *slog.Logger) bootstrap.Service {
	return &loggingMiddleware{logger, svc}
}

// Add logs the add request. It logs the client ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Add(ctx context.Context, session mgauthn.Session, token string, cfg bootstrap.Config) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", saved.ClientID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Add new bootstrap failed", args...)
			return
		}
		lm.logger.Info("Add new bootstrap completed successfully", args...)
	}(time.Now())

	return lm.svc.Add(ctx, session, token, cfg)
}

// View logs the view request. It logs the client ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) View(ctx context.Context, session mgauthn.Session, id string) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View client config failed", args...)
			return
		}
		lm.logger.Info("View client config completed successfully", args...)
	}(time.Now())

	return lm.svc.View(ctx, session, id)
}

// Update logs the update request. It logs bootstrap client ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Update(ctx context.Context, session mgauthn.Session, cfg bootstrap.Config) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("config",
				slog.String("client_id", cfg.ClientID),
				slog.String("name", cfg.Name),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update bootstrap config failed", args...)
			return
		}
		lm.logger.Info("Update bootstrap config completed successfully", args...)
	}(time.Now())

	return lm.svc.Update(ctx, session, cfg)
}

// UpdateCert logs the update_cert request. It logs client ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateCert(ctx context.Context, session mgauthn.Session, clientID, clientCert, clientKey, caCert string) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", cfg.ClientID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update bootstrap config certificate failed", args...)
			return
		}
		lm.logger.Info("Update bootstrap config certificate completed successfully", args...)
	}(time.Now())

	return lm.svc.UpdateCert(ctx, session, clientID, clientCert, clientKey, caCert)
}

// UpdateConnections logs the update_connections request. It logs bootstrap ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdateConnections(ctx context.Context, session mgauthn.Session, token, id string, connections []string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", id),
			slog.Any("connections", connections),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update config connections failed", args...)
			return
		}
		lm.logger.Info("Update config connections completed successfully", args...)
	}(time.Now())

	return lm.svc.UpdateConnections(ctx, session, token, id, connections)
}

// List logs the list request. It logs offset, limit and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) List(ctx context.Context, session mgauthn.Session, filter bootstrap.Filter, offset, limit uint64) (res bootstrap.ConfigsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.Any("filter", filter),
				slog.Uint64("offset", offset),
				slog.Uint64("limit", limit),
				slog.Uint64("total", res.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List configs failed", args...)
			return
		}
		lm.logger.Info("List configs completed successfully", args...)
	}(time.Now())

	return lm.svc.List(ctx, session, filter, offset, limit)
}

// Remove logs the remove request. It logs bootstrap ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Remove(ctx context.Context, session mgauthn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Remove bootstrap config failed", args...)
			return
		}
		lm.logger.Info("Remove bootstrap config completed successfully", args...)
	}(time.Now())

	return lm.svc.Remove(ctx, session, id)
}

func (lm *loggingMiddleware) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("external_id", externalID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View bootstrap config failed", args...)
			return
		}
		lm.logger.Info("View bootstrap completed successfully", args...)
	}(time.Now())

	return lm.svc.Bootstrap(ctx, externalKey, externalID, secure)
}

func (lm *loggingMiddleware) ChangeState(ctx context.Context, session mgauthn.Session, token, id string, state bootstrap.State) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("id", id),
			slog.Any("state", state),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Change client state failed", args...)
			return
		}
		lm.logger.Info("Change client state completed successfully", args...)
	}(time.Now())

	return lm.svc.ChangeState(ctx, session, token, id, state)
}

func (lm *loggingMiddleware) UpdateChannelHandler(ctx context.Context, channel bootstrap.Channel) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("channel",
				slog.String("id", channel.ID),
				slog.String("name", channel.Name),
				slog.Any("metadata", channel.Metadata),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update channel handler failed", args...)
			return
		}
		lm.logger.Info("Update channel handler completed successfully", args...)
	}(time.Now())

	return lm.svc.UpdateChannelHandler(ctx, channel)
}

func (lm *loggingMiddleware) RemoveConfigHandler(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Remove config handler failed", args...)
			return
		}
		lm.logger.Info("Remove config handler completed successfully", args...)
	}(time.Now())

	return lm.svc.RemoveConfigHandler(ctx, id)
}

func (lm *loggingMiddleware) RemoveChannelHandler(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Remove channel handler failed", args...)
			return
		}
		lm.logger.Info("Remove channel handler completed successfully", args...)
	}(time.Now())

	return lm.svc.RemoveChannelHandler(ctx, id)
}

func (lm *loggingMiddleware) ConnectClientHandler(ctx context.Context, channelID, clientID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", channelID),
			slog.String("client_id", clientID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Connect client handler failed", args...)
			return
		}
		lm.logger.Info("Connect client handler completed successfully", args...)
	}(time.Now())

	return lm.svc.ConnectClientHandler(ctx, channelID, clientID)
}

func (lm *loggingMiddleware) DisconnectClientHandler(ctx context.Context, channelID, clientID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", channelID),
			slog.String("client_id", clientID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Disconnect client handler failed", args...)
			return
		}
		lm.logger.Info("Disconnect client handler completed successfully", args...)
	}(time.Now())

	return lm.svc.DisconnectClientHandler(ctx, channelID, clientID)
}
