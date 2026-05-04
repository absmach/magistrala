// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
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
func (lm *loggingMiddleware) Add(ctx context.Context, session smqauthn.Session, token string, cfg bootstrap.Config) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", saved.ID),
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
func (lm *loggingMiddleware) View(ctx context.Context, session smqauthn.Session, id string) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", id),
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
func (lm *loggingMiddleware) Update(ctx context.Context, session smqauthn.Session, cfg bootstrap.Config) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("config",
				slog.String("config_id", cfg.ID),
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
func (lm *loggingMiddleware) UpdateCert(ctx context.Context, session smqauthn.Session, clientID, clientCert, clientKey, caCert string) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", cfg.ID),
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

// List logs the list request. It logs offset, limit and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) List(ctx context.Context, session smqauthn.Session, filter bootstrap.Filter, offset, limit uint64) (res bootstrap.ConfigsPage, err error) {
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
func (lm *loggingMiddleware) Remove(ctx context.Context, session smqauthn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", id),
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

func (lm *loggingMiddleware) EnableConfig(ctx context.Context, session smqauthn.Session, id string) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Enable config failed", args...)
			return
		}
		lm.logger.Info("Enable config completed successfully", args...)
	}(time.Now())

	return lm.svc.EnableConfig(ctx, session, id)
}

func (lm *loggingMiddleware) DisableConfig(ctx context.Context, session smqauthn.Session, id string) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Disable config failed", args...)
			return
		}
		lm.logger.Info("Disable config completed successfully", args...)
	}(time.Now())

	return lm.svc.DisableConfig(ctx, session, id)
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

func (lm *loggingMiddleware) CreateProfile(ctx context.Context, session smqauthn.Session, p bootstrap.Profile) (saved bootstrap.Profile, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("profile_id", saved.ID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Create profile failed", args...)
			return
		}
		lm.logger.Info("Create profile completed successfully", args...)
	}(time.Now())

	return lm.svc.CreateProfile(ctx, session, p)
}

func (lm *loggingMiddleware) ViewProfile(ctx context.Context, session smqauthn.Session, profileID string) (p bootstrap.Profile, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("profile_id", profileID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View profile failed", args...)
			return
		}
		lm.logger.Info("View profile completed successfully", args...)
	}(time.Now())

	return lm.svc.ViewProfile(ctx, session, profileID)
}

func (lm *loggingMiddleware) UpdateProfile(ctx context.Context, session smqauthn.Session, p bootstrap.Profile) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("profile_id", p.ID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update profile failed", args...)
			return
		}
		lm.logger.Info("Update profile completed successfully", args...)
	}(time.Now())

	return lm.svc.UpdateProfile(ctx, session, p)
}

func (lm *loggingMiddleware) ListProfiles(ctx context.Context, session smqauthn.Session, offset, limit uint64) (page bootstrap.ProfilesPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Uint64("offset", offset),
			slog.Uint64("limit", limit),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List profiles failed", args...)
			return
		}
		lm.logger.Info("List profiles completed successfully", args...)
	}(time.Now())

	return lm.svc.ListProfiles(ctx, session, offset, limit)
}

func (lm *loggingMiddleware) DeleteProfile(ctx context.Context, session smqauthn.Session, profileID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("profile_id", profileID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Delete profile failed", args...)
			return
		}
		lm.logger.Info("Delete profile completed successfully", args...)
	}(time.Now())

	return lm.svc.DeleteProfile(ctx, session, profileID)
}

func (lm *loggingMiddleware) AssignProfile(ctx context.Context, session smqauthn.Session, configID, profileID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", configID),
			slog.String("profile_id", profileID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Assign profile failed", args...)
			return
		}
		lm.logger.Info("Assign profile completed successfully", args...)
	}(time.Now())

	return lm.svc.AssignProfile(ctx, session, configID, profileID)
}

func (lm *loggingMiddleware) BindResources(ctx context.Context, session smqauthn.Session, token, configID string, bindings []bootstrap.BindingRequest) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", configID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Bind resources failed", args...)
			return
		}
		lm.logger.Info("Bind resources completed successfully", args...)
	}(time.Now())

	return lm.svc.BindResources(ctx, session, token, configID, bindings)
}

func (lm *loggingMiddleware) ListBindings(ctx context.Context, session smqauthn.Session, configID string) (snapshots []bootstrap.BindingSnapshot, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", configID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List bindings failed", args...)
			return
		}
		lm.logger.Info("List bindings completed successfully", args...)
	}(time.Now())

	return lm.svc.ListBindings(ctx, session, configID)
}

func (lm *loggingMiddleware) RefreshBindings(ctx context.Context, session smqauthn.Session, token, configID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("config_id", configID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Refresh bindings failed", args...)
			return
		}
		lm.logger.Info("Refresh bindings completed successfully", args...)
	}(time.Now())

	return lm.svc.RefreshBindings(ctx, session, token, configID)
}
