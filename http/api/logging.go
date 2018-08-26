//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package api

import (
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

func (lm *loggingMiddleware) Publish(msg mainflux.RawMessage) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method publish took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(msg)
}
