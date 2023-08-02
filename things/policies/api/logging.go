// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things/policies"
)

var _ policies.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mflog.Logger
	svc    policies.Service
}

// LoggingMiddleware returns a new logging middleware.
func LoggingMiddleware(svc policies.Service, logger mflog.Logger) policies.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Authorize(ctx context.Context, ar policies.AccessRequest) (policy policies.Policy, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method authorize for channel with id %s by client with id %s took %s to complete", ar.Object, ar.Subject, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.Authorize(ctx, ar)
}

func (lm *loggingMiddleware) AddPolicy(ctx context.Context, token string, external bool, p policies.Policy) (policy policies.Policy, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_policy for client with id %s using token %s took %s to complete", p.Subject, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.AddPolicy(ctx, token, external, p)
}

func (lm *loggingMiddleware) UpdatePolicy(ctx context.Context, token string, p policies.Policy) (policy policies.Policy, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_policy for client with id %s using token %s took %s to complete", p.Subject, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdatePolicy(ctx, token, p)
}

func (lm *loggingMiddleware) ListPolicies(ctx context.Context, token string, p policies.Page) (policypage policies.PolicyPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_policy for client with id %s using token %s took %s to complete", p.Subject, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListPolicies(ctx, token, p)
}

func (lm *loggingMiddleware) DeletePolicy(ctx context.Context, token string, p policies.Policy) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_policy for client with id %s using token %s took %s to complete", p.Subject, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DeletePolicy(ctx, token, p)
}
