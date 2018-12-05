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
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/readers"
)

var _ readers.MessageRepository = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger logger.Logger
	svc    readers.MessageRepository
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc readers.MessageRepository, logger logger.Logger) readers.MessageRepository {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm *loggingMiddleware) ReadAll(chanID string, offset, limit uint64) []mainflux.Message {
	defer func(begin time.Time) {
		lm.logger.Info(fmt.Sprintf(`Method read_all for offset %d and limit %d took
            %s to complete without errors.`, offset, limit, time.Since(begin)))
	}(time.Now())

	return lm.svc.ReadAll(chanID, offset, limit)
}
