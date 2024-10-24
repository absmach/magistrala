// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/certs"
)

var _ certs.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    certs.Service
}

// LoggingMiddleware adds logging facilities to the bootstrap service.
func LoggingMiddleware(svc certs.Service, logger *slog.Logger) certs.Service {
	return &loggingMiddleware{logger, svc}
}

// IssueCert logs the issue_cert request. It logs the ttl, client ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) IssueCert(ctx context.Context, domainID, token, clientID, ttl string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", clientID),
			slog.String("ttl", ttl),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Issue certificate failed", args...)
			return
		}
		lm.logger.Info("Issue certificate completed successfully", args...)
	}(time.Now())

	return lm.svc.IssueCert(ctx, domainID, token, clientID, ttl)
}

// ListCerts logs the list_certs request. It logs the client ID and the time it took to complete the request.
func (lm *loggingMiddleware) ListCerts(ctx context.Context, clientID string, pm certs.PageMetadata) (cp certs.CertPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", clientID),
			slog.Group("page",
				slog.Uint64("offset", cp.Offset),
				slog.Uint64("limit", cp.Limit),
				slog.Uint64("total", cp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List certificates failed", args...)
			return
		}
		lm.logger.Info("List certificates completed successfully", args...)
	}(time.Now())

	return lm.svc.ListCerts(ctx, clientID, pm)
}

// ListSerials logs the list_serials request. It logs the client ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ListSerials(ctx context.Context, clientID string, pm certs.PageMetadata) (cp certs.CertPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", clientID),
			slog.String("revoke", pm.Revoked),
			slog.Group("page",
				slog.Uint64("offset", cp.Offset),
				slog.Uint64("limit", cp.Limit),
				slog.Uint64("total", cp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("List certifcates serials failed", args...)
			return
		}
		lm.logger.Info("List certificates serials completed successfully", args...)
	}(time.Now())

	return lm.svc.ListSerials(ctx, clientID, pm)
}

// ViewCert logs the view_cert request. It logs the serial ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) ViewCert(ctx context.Context, serialID string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("serial_id", serialID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("View certificate failed", args...)
			return
		}
		lm.logger.Info("View certificate completed successfully", args...)
	}(time.Now())

	return lm.svc.ViewCert(ctx, serialID)
}

// RevokeCert logs the revoke_cert request. It logs the client ID and the time it took to complete the request.
// If the request fails, it logs the error.
func (lm *loggingMiddleware) RevokeCert(ctx context.Context, domainID, token, clientID string) (c certs.Revoke, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("client_id", clientID),
		}
		if err != nil {
			args = append(args, slog.Any("error", err))
			lm.logger.Warn("Revoke certificate failed", args...)
			return
		}
		lm.logger.Info("Revoke certificate completed successfully", args...)
	}(time.Now())

	return lm.svc.RevokeCert(ctx, domainID, token, clientID)
}
