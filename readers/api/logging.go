// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"log/slog"
	"time"

	"github.com/absmach/supermq/readers"
)

var _ readers.MessageRepository = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    readers.MessageRepository
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc readers.MessageRepository, logger *slog.Logger) readers.MessageRepository {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm *loggingMiddleware) ReadAll(chanID string, rpm readers.PageMetadata) (page readers.MessagesPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
			slog.Group("page",
				slog.Uint64("offset", rpm.Offset),
				slog.Uint64("limit", rpm.Limit),
				slog.Uint64("total", page.Total),
			),
		}
		if rpm.Subtopic != "" {
			args = append(args, slog.String("subtopic", rpm.Subtopic))
		}
		if rpm.Publisher != "" {
			args = append(args, slog.String("publisher", rpm.Publisher))
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Read all failed", args...)
			return
		}
		lm.logger.Info("Read all completed successfully", args...)
	}(time.Now())

	return lm.svc.ReadAll(chanID, rpm)
}
