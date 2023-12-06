// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/twins"
)

var _ twins.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mglog.Logger
	svc    twins.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc twins.Service, logger mglog.Logger) twins.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) AddTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) (tw twins.Twin, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_twin for token %s and twin %s took %s to complete", token, twin.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddTwin(ctx, token, twin, def)
}

func (lm *loggingMiddleware) UpdateTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_twin for token %s and twin %s took %s to complete", token, twin.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateTwin(ctx, token, twin, def)
}

func (lm *loggingMiddleware) ViewTwin(ctx context.Context, token, twinID string) (tw twins.Twin, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_twin for token %s and twin %s took %s to complete", token, twinID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewTwin(ctx, token, twinID)
}

func (lm *loggingMiddleware) ListTwins(ctx context.Context, token string, offset, limit uint64, name string, metadata twins.Metadata) (page twins.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_twins for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListTwins(ctx, token, offset, limit, name, metadata)
}

func (lm *loggingMiddleware) SaveStates(ctx context.Context, msg *messaging.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method save_states took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SaveStates(ctx, msg)
}

func (lm *loggingMiddleware) ListStates(ctx context.Context, token string, offset, limit uint64, twinID string) (page twins.StatesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_states for token %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListStates(ctx, token, offset, limit, twinID)
}

func (lm *loggingMiddleware) RemoveTwin(ctx context.Context, token, twinID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_twin for token %s and twin %s took %s to complete", token, twinID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveTwin(ctx, token, twinID)
}
