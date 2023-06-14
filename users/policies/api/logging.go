// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	mflog "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users/policies"
)

var _ policies.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mflog.Logger
	svc    policies.Service
}

// LoggingMiddleware adds logging facilities to the policies service.
func LoggingMiddleware(svc policies.Service, logger mflog.Logger) policies.Service {
	return &loggingMiddleware{logger, svc}
}

// Authorize logs the authorize request. It logs the subject, object, action and entity and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) Authorize(ctx context.Context, ar policies.AccessRequest) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method authorize for subject %s on object %s for action %s with entity %s took %s to complete", ar.Subject, ar.Object, ar.Action, ar.Entity, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.Authorize(ctx, ar)
}

// AddPolicy logs the add_policy request. It logs the subject, object, actions and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) AddPolicy(ctx context.Context, token string, p policies.Policy) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_policy with subject %s object %s and actions %s using token %s took %s to complete", p.Subject, p.Object, p.Actions, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.AddPolicy(ctx, token, p)
}

// UpdatePolicy logs the update_policy request. It logs the subject, object and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) UpdatePolicy(ctx context.Context, token string, p policies.Policy) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_policy for subject %s and object %s using token %s took %s to complete", p.Subject, p.Object, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.UpdatePolicy(ctx, token, p)
}

// ListPolicies logs the list_policies request. It logs the token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListPolicies(ctx context.Context, token string, cp policies.Page) (cg policies.PolicyPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_policy using token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.ListPolicies(ctx, token, cp)
}

// DeletePolicy logs the delete_policy request. It logs the subject, object and token and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) DeletePolicy(ctx context.Context, token string, p policies.Policy) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_policy for subject %s and object %s using token %s took %s to complete", p.Subject, p.Object, token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DeletePolicy(ctx, token, p)
}
