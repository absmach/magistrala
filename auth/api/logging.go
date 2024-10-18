// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/policies"
)

var _ auth.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    auth.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc auth.Service, logger *slog.Logger) auth.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Issue(ctx context.Context, token string, key auth.Key) (tkn auth.Token, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("key",
				slog.String("subject", key.Subject),
				slog.Any("type", key.Type),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Issue key failed", args...)
			return
		}
		lm.logger.Info("Issue key completed successfully", args...)
	}(time.Now())

	return lm.svc.Issue(ctx, token, key)
}

func (lm *loggingMiddleware) Revoke(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("key_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Revoke key failed", args...)
			return
		}
		lm.logger.Info("Revoke key completed successfully", args...)
	}(time.Now())

	return lm.svc.Revoke(ctx, token, id)
}

func (lm *loggingMiddleware) RetrieveKey(ctx context.Context, token, id string) (key auth.Key, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("key_id", id),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Retrieve key failed", args...)
			return
		}
		lm.logger.Info("Retrieve key completed successfully", args...)
	}(time.Now())

	return lm.svc.RetrieveKey(ctx, token, id)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, token string) (id auth.Key, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("key",
				slog.String("subject", id.Subject),
				slog.Any("type", id.Type),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Identify key failed", args...)
			return
		}
		lm.logger.Info("Identify key completed successfully", args...)
	}(time.Now())

	return lm.svc.Identify(ctx, token)
}

func (lm *loggingMiddleware) Authorize(ctx context.Context, pr policies.Policy) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("object",
				slog.String("id", pr.Object),
				slog.String("type", pr.ObjectType),
			),
			slog.Group("subject",
				slog.String("id", pr.Subject),
				slog.String("kind", pr.SubjectKind),
				slog.String("type", pr.SubjectType),
			),
			slog.String("permission", pr.Permission),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Authorize failed", args...)
			return
		}
		lm.logger.Info("Authorize completed successfully", args...)
	}(time.Now())
	return lm.svc.Authorize(ctx, pr)
}
