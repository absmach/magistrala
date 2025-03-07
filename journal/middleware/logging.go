// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/supermq/journal"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/go-chi/chi/v5/middleware"
)

var _ journal.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger  *slog.Logger
	service journal.Service
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(service journal.Service, logger *slog.Logger) journal.Service {
	return &loggingMiddleware{
		logger:  logger,
		service: service,
	}
}

func (lm *loggingMiddleware) Save(ctx context.Context, j journal.Journal) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("journal",
				slog.String("occurred_at", j.OccurredAt.Format(time.RFC3339Nano)),
				slog.String("operation", j.Operation),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Save journal failed", args...)
			return
		}
		lm.logger.Info("Save journal completed successfully", args...)
	}(time.Now())

	return lm.service.Save(ctx, j)
}

func (lm *loggingMiddleware) RetrieveAll(ctx context.Context, session smqauthn.Session, page journal.Page) (journalsPage journal.JournalsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.Group("page",
				slog.String("operation", page.Operation),
				slog.String("entity_type", page.EntityType.String()),
				slog.Uint64("offset", page.Offset),
				slog.Uint64("limit", page.Limit),
				slog.Uint64("total", journalsPage.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Retrieve all journals failed", args...)
			return
		}
		lm.logger.Info("Retrieve all journals completed successfully", args...)
	}(time.Now())

	return lm.service.RetrieveAll(ctx, session, page)
}

func (lm *loggingMiddleware) RetrieveClientTelemetry(ctx context.Context, session smqauthn.Session, clientID string) (ct journal.ClientTelemetry, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("request_id", middleware.GetReqID(ctx)),
			slog.String("client_id", clientID),
			slog.String("domain_id", session.DomainID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Retrieve client telemetry failed", args...)
			return
		}
		lm.logger.Info("Retrieve client telemetry completed successfully", args...)
	}(time.Now())

	return lm.service.RetrieveClientTelemetry(ctx, session, clientID)
}
