//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/normalizer"
)

var _ normalizer.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger logger.Logger
	svc    normalizer.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc normalizer.Service, logger logger.Logger) normalizer.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm loggingMiddleware) Normalize(msg mainflux.RawMessage) (nd normalizer.NormalizedData, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method normalize took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Normalize(msg)
}
