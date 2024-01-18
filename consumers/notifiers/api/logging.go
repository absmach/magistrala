// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/consumers/notifiers"
)

var _ notifiers.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    notifiers.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc notifiers.Service, logger *slog.Logger) notifiers.Service {
	return &loggingMiddleware{logger, svc}
}

// CreateSubscription logs the create_subscription request. It logs token and subscription ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) CreateSubscription(ctx context.Context, token string, sub notifiers.Subscription) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_subscription with the id %s for token %s took %s to complete", id, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateSubscription(ctx, token, sub)
}

// ViewSubscription logs the view_subscription request. It logs token and subscription topic and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewSubscription(ctx context.Context, token, topic string) (sub notifiers.Subscription, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_subscription with the topic %s for token %s took %s to complete", topic, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewSubscription(ctx, token, topic)
}

// ListSubscriptions logs the list_subscriptions request. It logs token and subscription topic and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListSubscriptions(ctx context.Context, token string, pm notifiers.PageMetadata) (res notifiers.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_subscriptions for topic %s and token %s took %s to complete", pm.Topic, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSubscriptions(ctx, token, pm)
}

// RemoveSubscription logs the remove_subscription request. It logs token and subscription ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) RemoveSubscription(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_subscription for subscription %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveSubscription(ctx, token, id)
}

// ConsumeBlocking logs the consume_blocking request. It logs the message and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ConsumeBlocking(ctx context.Context, msg interface{}) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConsumeBlocking(ctx, msg)
}
