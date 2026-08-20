// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package middleware

import (
	"log/slog"
	"time"

	"github.com/absmach/magistrala/readers"
)

var _ readers.MessageRepository = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    readers.MessageRepository
}

// LoggingMiddleware adds logging facilities to the service.
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

func (lm *loggingMiddleware) ListGatewayDevices(chanID, publisherID string, rpm readers.PageMetadata) (page readers.DeviceStatsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
			slog.String("publisher_id", publisherID),
			slog.Group("page",
				slog.Uint64("offset", rpm.Offset),
				slog.Uint64("limit", rpm.Limit),
				slog.Uint64("total", page.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List gateway devices failed", args...)
			return
		}
		lm.logger.Info("List gateway devices completed successfully", args...)
	}(time.Now())

	return lm.svc.ListGatewayDevices(chanID, publisherID, rpm)
}

func (lm *loggingMiddleware) ListDeviceGateways(chanID, deviceID string, rpm readers.PageMetadata) (page readers.DeviceStatsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
			slog.String("device_id", deviceID),
			slog.Group("page",
				slog.Uint64("offset", rpm.Offset),
				slog.Uint64("limit", rpm.Limit),
				slog.Uint64("total", page.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List device gateways failed", args...)
			return
		}
		lm.logger.Info("List device gateways completed successfully", args...)
	}(time.Now())

	return lm.svc.ListDeviceGateways(chanID, deviceID, rpm)
}
