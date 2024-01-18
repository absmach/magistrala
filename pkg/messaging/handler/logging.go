// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/mproxy/pkg/session"
)

var _ session.Handler = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    session.Handler
}

// AuthConnect implements session.Handler.
func (lm *loggingMiddleware) AuthConnect(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method auth connect took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AuthConnect(ctx)
}

// AuthPublish implements session.Handler.
func (lm *loggingMiddleware) AuthPublish(ctx context.Context, topic *string, payload *[]byte) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method auth publish took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AuthPublish(ctx, topic, payload)
}

// AuthSubscribe implements session.Handler.
func (lm *loggingMiddleware) AuthSubscribe(ctx context.Context, topics *[]string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method auth subscribe took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AuthSubscribe(ctx, topics)
}

// Connect implements session.Handler.
func (lm *loggingMiddleware) Connect(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method connect took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Connect(ctx)
}

// Disconnect implements session.Handler.
func (lm *loggingMiddleware) Disconnect(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disconnect took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Disconnect(ctx)
}

// Publish logs the publish request. It logs the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Publish(ctx context.Context, topic *string, payload *[]byte) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method publish to channel %s took %s to complete", *topic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, topic, payload)
}

// Subscribe implements session.Handler.
func (lm *loggingMiddleware) Subscribe(ctx context.Context, topics *[]string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method subscribe took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Subscribe(ctx, topics)
}

// Unsubscribe implements session.Handler.
func (lm *loggingMiddleware) Unsubscribe(ctx context.Context, topics *[]string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unsubscribe took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Unsubscribe(ctx, topics)
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc session.Handler, logger *slog.Logger) session.Handler {
	return &loggingMiddleware{logger, svc}
}
