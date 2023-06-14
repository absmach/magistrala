// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/ws"
)

var _ ws.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mflog.Logger
	svc    ws.Service
}

// LoggingMiddleware adds logging facilities to the websocket service.
func LoggingMiddleware(svc ws.Service, logger mflog.Logger) ws.Service {
	return &loggingMiddleware{logger, svc}
}

// Publish logs the publish request. It logs the channel and subtopic(if present) and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Publish(ctx context.Context, thingKey string, msg *messaging.Message) (err error) {
	defer func(begin time.Time) {
		destChannel := msg.GetChannel()
		if msg.Subtopic != "" {
			destChannel = fmt.Sprintf("%s.%s", destChannel, msg.Subtopic)
		}
		message := fmt.Sprintf("Method publish to %s took %s to complete", destChannel, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, thingKey, msg)
}

// Subscribe logs the subscribe request. It logs the channel and subtopic(if present) and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Subscribe(ctx context.Context, thingKey, chanID, subtopic string, c *ws.Client) (err error) {
	defer func(begin time.Time) {
		destChannel := chanID
		if subtopic != "" {
			destChannel = fmt.Sprintf("%s.%s", destChannel, subtopic)
		}
		message := fmt.Sprintf("Method subscribe to channel %s took %s to complete", destChannel, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Subscribe(ctx, thingKey, chanID, subtopic, c)
}

// Unsubscribe logs the unsubscribe request. It logs the channel and subtopic(if present) and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Unsubscribe(ctx context.Context, thingKey, chanID, subtopic string) (err error) {
	defer func(begin time.Time) {
		destChannel := chanID
		if subtopic != "" {
			destChannel = fmt.Sprintf("%s.%s", destChannel, subtopic)
		}
		message := fmt.Sprintf("Method unsubscribe from channel %s took %s to complete", destChannel, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Unsubscribe(ctx, thingKey, chanID, subtopic)
}
