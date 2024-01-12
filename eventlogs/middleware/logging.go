// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala/eventlogs"
	mglog "github.com/absmach/magistrala/logger"
)

var _ eventlogs.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger  mglog.Logger
	service eventlogs.Service
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(service eventlogs.Service, logger mglog.Logger) eventlogs.Service {
	return &loggingMiddleware{
		logger:  logger,
		service: service,
	}
}

func (lm *loggingMiddleware) ReadAll(ctx context.Context, token string, page eventlogs.Page) (eventsPage eventlogs.EventsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method read_all for operation %s with query %v took %s to complete", page.Operation, page, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.service.ReadAll(ctx, token, page)
}
