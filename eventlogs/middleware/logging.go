// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/eventlogs"
)

var _ eventlogs.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger  *slog.Logger
	service eventlogs.Service
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(service eventlogs.Service, logger *slog.Logger) eventlogs.Service {
	return &loggingMiddleware{
		logger:  logger,
		service: service,
	}
}

func (lm *loggingMiddleware) ReadAll(ctx context.Context, token string, page eventlogs.Page) (eventsPage eventlogs.EventsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("page",
				slog.String("operation", page.Operation),
				slog.Uint64("offset", page.Offset),
				slog.Uint64("limit", page.Limit),
				slog.Uint64("total", eventsPage.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Read all events failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Read all events completed successfully", args...)
	}(time.Now())

	return lm.service.ReadAll(ctx, token, page)
}
