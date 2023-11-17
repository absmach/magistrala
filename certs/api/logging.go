// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala/certs"
	mglog "github.com/absmach/magistrala/logger"
)

var _ certs.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mglog.Logger
	svc    certs.Service
}

// LoggingMiddleware adds logging facilities to the bootstrap service.
func LoggingMiddleware(svc certs.Service, logger mglog.Logger) certs.Service {
	return &loggingMiddleware{logger, svc}
}

// IssueCert logs the issue_cert request. It logs the token, thing ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) IssueCert(ctx context.Context, token, thingID, ttl string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method issue_cert using token %s and thing %s took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.IssueCert(ctx, token, thingID, ttl)
}

// ListCerts logs the list_certs request. It logs the token, thing ID and the time it took to complete the request.
func (lm *loggingMiddleware) ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (cp certs.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_certs using token %s and thing id %s took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListCerts(ctx, token, thingID, offset, limit)
}

// ListSerials logs the list_serials request. It logs the token, thing ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListSerials(ctx context.Context, token, thingID string, offset, limit uint64) (cp certs.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_serials using token %s and thing id %s took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSerials(ctx, token, thingID, offset, limit)
}

// ViewCert logs the view_cert request. It logs the token, serial ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewCert(ctx context.Context, token, serialID string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_cert using token %s and serial id %s took %s to complete", token, serialID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewCert(ctx, token, serialID)
}

// RevokeCert logs the revoke_cert request. It logs the token, thing ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) RevokeCert(ctx context.Context, token, thingID string) (c certs.Revoke, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_cert using token %s and thing %s took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RevokeCert(ctx, token, thingID)
}
