// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux/certs"
	log "github.com/mainflux/mainflux/logger"
)

var _ certs.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    certs.Service
}

// NewLoggingMiddleware adds logging facilities to the core service.
func NewLoggingMiddleware(svc certs.Service, logger log.Logger) certs.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) IssueCert(ctx context.Context, token, thingID, name, ttl string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method issue_cert for token: %s and thing: %s took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.IssueCert(ctx, token, thingID, name, ttl)
}

func (lm *loggingMiddleware) ListCerts(ctx context.Context, token, certID, thingID, serial, name string, status certs.Status, offset, limit uint64) (cp certs.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_certs for token: %s, cert ID: %s  thing id: %s serial: %s name: %s took %s to complete", token, certID, thingID, serial, name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListCerts(ctx, token, certID, thingID, serial, name, status, offset, limit)
}

func (lm *loggingMiddleware) ViewCert(ctx context.Context, token, certID string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_cert for token: %s and certificate id: %s took %s to complete", token, certID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewCert(ctx, token, certID)
}

func (lm *loggingMiddleware) RevokeCert(ctx context.Context, token, certID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_cert for token: %s and certificate id: %s took %s to complete", token, certID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RevokeCert(ctx, token, certID)
}

func (lm *loggingMiddleware) RenewCert(ctx context.Context, token, certID string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method renew_certs for token: %s and certificate id: %s took %s to complete", token, certID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RenewCert(ctx, token, certID)
}

func (lm *loggingMiddleware) RemoveCert(ctx context.Context, token, certID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method renew_certs for token: %s and certificate id: %s took %s to complete", token, certID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveCert(ctx, token, certID)
}

func (lm *loggingMiddleware) RevokeThingCerts(ctx context.Context, token, thingID string, limit int64) (c uint64, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_cert for token: %s and thing: %s took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors. %d remaining certificates to revoke ", message, c))
	}(time.Now())

	return lm.svc.RevokeThingCerts(ctx, token, thingID, limit)
}

func (lm *loggingMiddleware) RenewThingCerts(ctx context.Context, token, thingID string, limit int64) (c uint64, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method renew_certs token: %s and thing: %s took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors. %d remaining certificates to renew ", message, c))
	}(time.Now())

	return lm.svc.RenewThingCerts(ctx, token, thingID, limit)
}

func (lm *loggingMiddleware) RemoveThingCerts(ctx context.Context, token, thingID string, limit int64) (c uint64, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_certs for token: %s and thing: %s took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors. %d remaining certificates to remove ", message, c))
	}(time.Now())

	return lm.svc.RemoveThingCerts(ctx, token, thingID, limit)
}
