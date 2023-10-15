// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux/auth"
	log "github.com/mainflux/mainflux/logger"
)

var _ auth.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    auth.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc auth.Service, logger log.Logger) auth.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) ListObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_objects took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListObjects(ctx, pr, nextPageToken, limit)
}

func (lm *loggingMiddleware) ListAllObjects(ctx context.Context, pr auth.PolicyReq) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_all_objects took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListAllObjects(ctx, pr)
}

func (lm *loggingMiddleware) CountObjects(ctx context.Context, pr auth.PolicyReq) (count int, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method count_objects took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.CountObjects(ctx, pr)
}

func (lm *loggingMiddleware) ListSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit int32) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_subjects took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSubjects(ctx, pr, nextPageToken, limit)
}

func (lm *loggingMiddleware) ListAllSubjects(ctx context.Context, pr auth.PolicyReq) (p auth.PolicyPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_all_subjects took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListAllSubjects(ctx, pr)
}

func (lm *loggingMiddleware) CountSubjects(ctx context.Context, pr auth.PolicyReq) (count int, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_subjects took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.CountSubjects(ctx, pr)
}

func (lm *loggingMiddleware) Issue(ctx context.Context, token string, key auth.Key) (tkn auth.Token, err error) {
	defer func(begin time.Time) {
		d := ""
		if key.Type != auth.AccessKey && !key.ExpiresAt.IsZero() {
			d = fmt.Sprintf("with expiration date %v", key.ExpiresAt)
		}
		message := fmt.Sprintf("Method issue for %d key %s took %s to complete", key.Type, d, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Issue(ctx, token, key)
}

func (lm *loggingMiddleware) Revoke(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke for key %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Revoke(ctx, token, id)
}

func (lm *loggingMiddleware) RetrieveKey(ctx context.Context, token, id string) (key auth.Key, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method retrieve for key %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RetrieveKey(ctx, token, id)
}

func (lm *loggingMiddleware) Identify(ctx context.Context, key string) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method identify took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Identify(ctx, key)
}

func (lm *loggingMiddleware) Authorize(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method authorize took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.Authorize(ctx, pr)
}

func (lm *loggingMiddleware) AddPolicy(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_policy took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.AddPolicy(ctx, pr)
}

func (lm *loggingMiddleware) AddPolicies(ctx context.Context, token, object string, subjectIDs, relations []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_policy_bulk took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddPolicies(ctx, token, object, subjectIDs, relations)
}

func (lm *loggingMiddleware) DeletePolicy(ctx context.Context, pr auth.PolicyReq) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_policy took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DeletePolicy(ctx, pr)
}

func (lm *loggingMiddleware) DeletePolicies(ctx context.Context, token, object string, subjectIDs, relations []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_policies took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.DeletePolicies(ctx, token, object, subjectIDs, relations)
}
