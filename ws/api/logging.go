// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/ws"
)

var _ ws.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mglog.Logger
	svc    ws.Service
}

// LoggingMiddleware adds logging facilities to the websocket service.
func LoggingMiddleware(svc ws.Service, logger mglog.Logger) ws.Service {
	return &loggingMiddleware{logger, svc}
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
