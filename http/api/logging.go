// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux/http"
	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
)

var _ http.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mflog.Logger
	svc    http.Service
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc http.Service, logger mflog.Logger) http.Service {
	return &loggingMiddleware{logger, svc}
}

// Publish logs the publish request. It logs the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Publish(ctx context.Context, token string, msg *messaging.Message) (err error) {
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
