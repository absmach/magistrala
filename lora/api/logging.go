// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/lora"
)

var _ lora.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    lora.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc lora.Service, logger *slog.Logger) lora.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm loggingMiddleware) CreateThing(ctx context.Context, thingID, loraDevEUI string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", thingID),
			slog.String("dev_eui", loraDevEUI),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Create thing route-map failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Create thing route-map completed successfully", args...)
	}(time.Now())

	return lm.svc.CreateThing(ctx, thingID, loraDevEUI)
}

func (lm loggingMiddleware) UpdateThing(ctx context.Context, thingID, loraDevEUI string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", thingID),
			slog.String("dev_eui", loraDevEUI),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Update thing route-map failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Update thing route-map completed successfully", args...)
	}(time.Now())

	return lm.svc.UpdateThing(ctx, thingID, loraDevEUI)
}

func (lm loggingMiddleware) RemoveThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", thingID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Remove thing route-map failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Remove thing route-map completed successfully", args...)
	}(time.Now())

	return lm.svc.RemoveThing(ctx, thingID)
}

func (lm loggingMiddleware) CreateChannel(ctx context.Context, chanID, loraApp string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
			slog.String("lora_app", loraApp),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Create channel route-map failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Create channel route-map completed successfully", args...)
	}(time.Now())

	return lm.svc.CreateChannel(ctx, chanID, loraApp)
}

func (lm loggingMiddleware) UpdateChannel(ctx context.Context, chanID, loraApp string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
			slog.String("lora_app", loraApp),
		}
		if err != nil {
			lm.logger.Warn("Update channel route-map failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Update channel route-map completed successfully", args...)
	}(time.Now())

	return lm.svc.UpdateChannel(ctx, chanID, loraApp)
}

func (lm loggingMiddleware) RemoveChannel(ctx context.Context, chanID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
		}
		if err != nil {
			lm.logger.Warn("Remove channel route-map failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Remove channel route-map completed successfully", args...)
	}(time.Now())

	return lm.svc.RemoveChannel(ctx, chanID)
}

func (lm loggingMiddleware) ConnectThing(ctx context.Context, chanID, thingID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
			slog.String("thing_id", thingID),
		}
		if err != nil {
			args := append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Connect thing to channel failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Connect thing to channel completed successfully", args...)
	}(time.Now())

	return lm.svc.ConnectThing(ctx, chanID, thingID)
}

func (lm loggingMiddleware) DisconnectThing(ctx context.Context, chanID, thingID string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("channel_id", chanID),
			slog.String("thing_id", thingID),
		}
		if err != nil {
			args := append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disconnect thing from channel failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Disconnect thing from channel completed successfully", args...)
	}(time.Now())

	return lm.svc.DisconnectThing(ctx, chanID, thingID)
}

func (lm loggingMiddleware) Publish(ctx context.Context, msg *lora.Message) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("message",
				slog.String("application_id", msg.ApplicationID),
				slog.String("device_eui", msg.DevEUI),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Publish failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Publish completed successfully", args...)
	}(time.Now())

	return lm.svc.Publish(ctx, msg)
}
