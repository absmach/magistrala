// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"log/slog"
	"time"

	"github.com/absmach/magistrala/provision"
)

var _ provision.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    provision.Service
}

// NewLoggingMiddleware adds logging facilities to the core service.
func NewLoggingMiddleware(svc provision.Service, logger *slog.Logger) provision.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Provision(token, name, externalID, externalKey string) (res provision.Result, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("name", name),
			slog.String("external_id", externalID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Provision failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Provision completed successfully", args...)
	}(time.Now())

	return lm.svc.Provision(token, name, externalID, externalKey)
}

func (lm *loggingMiddleware) Cert(token, thingID, duration string) (cert, key string, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", thingID),
			slog.String("ttl", duration),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Thing certificate failed to create successfully", args...)
			return
		}
		lm.logger.Info("Thing certificate created successfully", args...)
	}(time.Now())

	return lm.svc.Cert(token, thingID, duration)
}

func (lm *loggingMiddleware) Mapping(token string) (res map[string]interface{}, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Mapping failed to complete successfully", args...)
			return
		}
		lm.logger.Info("Mapping completed successfully", args...)
	}(time.Now())

	return lm.svc.Mapping(token)
}
