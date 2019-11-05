// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test

package api

import (
	"fmt"
	"time"

	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers/senml"
	"github.com/mainflux/mainflux/writers"
)

var _ writers.MessageRepository = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    writers.MessageRepository
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc writers.MessageRepository, logger log.Logger) writers.MessageRepository {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Save(msgs ...senml.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method save took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Save(msgs...)
}
