// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package handler

import (
	"context"
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
	defer lm.logAction("AuthConnect", nil, time.Now(), err)
	return lm.svc.AuthConnect(ctx)
}

// AuthPublish implements session.Handler.
func (lm *loggingMiddleware) AuthPublish(ctx context.Context, topic *string, payload *[]byte) (err error) {
	defer lm.logAction("AuthPublish", &[]string{*topic}, time.Now(), err)
	return lm.svc.AuthPublish(ctx, topic, payload)
}

// AuthSubscribe implements session.Handler.
func (lm *loggingMiddleware) AuthSubscribe(ctx context.Context, topics *[]string) (err error) {
	defer lm.logAction("AuthSubscribe", topics, time.Now(), err)
	return lm.svc.AuthSubscribe(ctx, topics)
}

// Connect implements session.Handler.
func (lm *loggingMiddleware) Connect(ctx context.Context) (err error) {
	defer lm.logAction("Connect", nil, time.Now(), err)
	return lm.svc.Connect(ctx)
}

// Disconnect implements session.Handler.
func (lm *loggingMiddleware) Disconnect(ctx context.Context) (err error) {
	defer lm.logAction("Disconnect", nil, time.Now(), err)
	return lm.svc.Disconnect(ctx)
}

// Publish logs the publish request. It logs the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Publish(ctx context.Context, topic *string, payload *[]byte) (err error) {
	defer lm.logAction("Publish", &[]string{*topic}, time.Now(), err)
	return lm.svc.Publish(ctx, topic, payload)
}

// Subscribe implements session.Handler.
func (lm *loggingMiddleware) Subscribe(ctx context.Context, topics *[]string) (err error) {
	defer lm.logAction("Subscribe", topics, time.Now(), err)
	return lm.svc.Subscribe(ctx, topics)
}

// Unsubscribe implements session.Handler.
func (lm *loggingMiddleware) Unsubscribe(ctx context.Context, topics *[]string) (err error) {
	defer lm.logAction("Unsubscribe", topics, time.Now(), err)
	return lm.svc.Unsubscribe(ctx, topics)
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc session.Handler, logger *slog.Logger) session.Handler {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) logAction(action string, topics *[]string, t time.Time, err error) {
	args := []any{
		slog.String("duration", time.Since(t).String()),
	}
	if topics != nil {
		args = append(args, slog.Any("topics", *topics))
	}
	if err != nil {
		args = append(args, slog.Any("error", err))
		lm.logger.Warn(action+" failed", args...)
		return
	}
	lm.logger.Info(action+" completed successfully", args...)
}
