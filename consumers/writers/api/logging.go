// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/supermq/consumers"
)

var _ consumers.BlockingConsumer = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger   *slog.Logger
	consumer consumers.BlockingConsumer
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(consumer consumers.BlockingConsumer, logger *slog.Logger) consumers.BlockingConsumer {
	return &loggingMiddleware{
		logger:   logger,
		consumer: consumer,
	}
}

// ConsumeBlocking logs the consume request. It logs the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ConsumeBlocking(ctx context.Context, msgs interface{}) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Blocking consumer failed to consume messages successfully", args...)
			return
		}
		lm.logger.Info("Blocking consumer consumed messages successfully", args...)
	}(time.Now())

	return lm.consumer.ConsumeBlocking(ctx, msgs)
}
