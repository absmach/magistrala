// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/mainflux/consumers"
	log "github.com/mainflux/mainflux/logger"
)

var _ consumers.BlockingConsumer = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger   log.Logger
	consumer consumers.BlockingConsumer
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(consumer consumers.BlockingConsumer, logger log.Logger) consumers.BlockingConsumer {
	return &loggingMiddleware{
		logger:   logger,
		consumer: consumer,
	}
}

func (lm *loggingMiddleware) ConsumeBlocking(msgs interface{}) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.consumer.ConsumeBlocking(msgs)
}
