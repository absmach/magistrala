//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
)

var _ mainflux.MessagePublisher = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    mainflux.MessagePublisher
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc mainflux.MessagePublisher, logger log.Logger) mainflux.MessagePublisher {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Publish(ctx context.Context, token string, msg mainflux.RawMessage) (err error) {
	defer func(begin time.Time) {
		destChannel := msg.Channel
		if msg.Subtopic != "" {
			destChannel = fmt.Sprintf("%s.%s", destChannel, msg.Subtopic)
		}
		message := fmt.Sprintf("Method publish to channel %s took %s to complete", destChannel, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, token, msg)
}
