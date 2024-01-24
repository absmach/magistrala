// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/twins"
)

var _ twins.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    twins.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc twins.Service, logger *slog.Logger) twins.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) AddTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) (tw twins.Twin, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("twin",
				slog.String("id", tw.ID),
				slog.String("name", tw.Name),
				slog.Any("definitions", tw.Definitions),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Add twin failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Add twin completed successfully", args...)
	}(time.Now())

	return lm.svc.AddTwin(ctx, token, twin, def)
}

func (lm *loggingMiddleware) UpdateTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("twin",
				slog.String("id", twin.ID),
				slog.String("name", twin.Name),
				slog.Any("definitions", def),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update twin failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Update twin completed successfully", args...)
	}(time.Now())

	return lm.svc.UpdateTwin(ctx, token, twin, def)
}

func (lm *loggingMiddleware) ViewTwin(ctx context.Context, token, twinID string) (tw twins.Twin, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("twin_id", twinID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View twin failed to complete successfully", args...)
			return
		}
		lm.logger.Info("View twin completed successfully", args...)
	}(time.Now())

	return lm.svc.ViewTwin(ctx, token, twinID)
}

func (lm *loggingMiddleware) ListTwins(ctx context.Context, token string, offset, limit uint64, name string, metadata twins.Metadata) (page twins.Page, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.String("name", name),
				slog.Uint64("offset", offset),
				slog.Uint64("limit", limit),
				slog.Uint64("total", page.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List twins failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List twins completed successfully", args...)
	}(time.Now())

	return lm.svc.ListTwins(ctx, token, offset, limit, name, metadata)
}

func (lm *loggingMiddleware) SaveStates(ctx context.Context, msg *messaging.Message) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("message",
				slog.String("channel", msg.GetChannel()),
				slog.String("subtopic", msg.GetSubtopic()),
				slog.String("publisher", msg.GetPublisher()),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Save states failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Save states completed successfully", args...)
	}(time.Now())

	return lm.svc.SaveStates(ctx, msg)
}

func (lm *loggingMiddleware) ListStates(ctx context.Context, token string, offset, limit uint64, twinID string) (page twins.StatesPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("twin_id", twinID),
			slog.Group("page",
				slog.Uint64("offset", offset),
				slog.Uint64("limit", limit),
				slog.Uint64("total", page.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List states failed to complete successfully", args...)
			return
		}
		lm.logger.Info("List states completed successfully", args...)
	}(time.Now())

	return lm.svc.ListStates(ctx, token, offset, limit, twinID)
}

func (lm *loggingMiddleware) RemoveTwin(ctx context.Context, token, twinID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("twin_id", twinID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Remove twin failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Remove twin completed successfully", args...)
	}(time.Now())

	return lm.svc.RemoveTwin(ctx, token, twinID)
}
