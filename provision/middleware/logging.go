// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/provision"
)

var _ provision.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    provision.Service
}

// NewLogging adds logging facilities to the core service.
func NewLogging(svc provision.Service, logger *slog.Logger) provision.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Provision(ctx context.Context, domainID, token, name, externalID, externalKey string) (res provision.Result, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("name", name),
			slog.String("external_id", externalID),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Provision failed", args...)
			return
		}
		lm.logger.Info("Provision completed successfully", args...)
	}(time.Now())

	return lm.svc.Provision(ctx, domainID, token, name, externalID, externalKey)
}

func (lm *loggingMiddleware) Cert(ctx context.Context, domainID, token, clientID, duration string) (cert, key string, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", clientID),
			slog.String("ttl", duration),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Client certificate creation failed", args...)
			return
		}
		lm.logger.Info("Client certificate created successfully", args...)
	}(time.Now())

	return lm.svc.Cert(ctx, domainID, token, clientID, duration)
}

func (lm *loggingMiddleware) Mapping() (res map[string]any) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		lm.logger.Info("Mapping completed successfully", args...)
	}(time.Now())

	return lm.svc.Mapping()
}
