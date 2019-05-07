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

func (lm *loggingMiddleware) Save(msg mainflux.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method Save took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Save(msg)
}
