// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/ws"
)

var _ ws.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    ws.Service
}

// LoggingMiddleware adds logging facilities to the websocket service.
func LoggingMiddleware(svc ws.Service, logger *slog.Logger) ws.Service {
	return &loggingMiddleware{logger, svc}
}

// Subscribe logs the subscribe request. It logs the channel and subtopic(if present) and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Subscribe(ctx context.Context, clientKey, chanID, subtopic string, c *ws.Client) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
		}
		if subtopic != "" {
			args = append(args, "subtopic", subtopic)
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Subscibe failed", args...)
			return
		}
		lm.logger.Info("Subscribe completed successfully", args...)
	}(time.Now())

	return lm.svc.Subscribe(ctx, clientKey, chanID, subtopic, c)
}
