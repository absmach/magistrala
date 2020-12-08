// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux/authz"
	log "github.com/mainflux/mainflux/logger"
)

var _ authz.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    authz.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc authz.Service, logger log.Logger) authz.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) AddPolicy(ctx context.Context, token string, p authz.Policy) (b bool, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_policy for token %s and policy %v took %s to complete", token, p, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddPolicy(ctx, token, p)
}

func (lm *loggingMiddleware) RemovePolicy(ctx context.Context, token string, p authz.Policy) (b bool, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_policy for token %s and policy %v took %s to complete", token, p, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemovePolicy(ctx, token, p)
}

func (lm *loggingMiddleware) Authorize(ctx context.Context, p authz.Policy) (b bool, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method authorize for %v took %s to complete", p, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Authorize(ctx, p)
}
