// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/pkg/authn"
)

var _ certs.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    certs.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc certs.Service, logger *slog.Logger) certs.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) RenewCert(ctx context.Context, session authn.Session, serialNumber string) (cert certs.Certificate, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method renew_cert for cert %s took %s to complete", serialNumber, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s and returned new cert %s.", message, cert.SerialNumber))
	}(time.Now())
	return lm.svc.RenewCert(ctx, session, serialNumber)
}

func (lm *loggingMiddleware) IssueCert(ctx context.Context, session authn.Session, entityID, ttl string, ipAddrs []string, options certs.SubjectOptions) (cert certs.Certificate, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method issue_cert for entity %s took %s to complete", entityID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.IssueCert(ctx, session, entityID, ttl, ipAddrs, options)
}

func (lm *loggingMiddleware) ListCerts(ctx context.Context, session authn.Session, pm certs.PageMetadata) (cp certs.CertificatePage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_certs took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.ListCerts(ctx, session, pm)
}

func (lm *loggingMiddleware) RevokeBySerial(ctx context.Context, session authn.Session, serialNumber string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_by_serial took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.RevokeBySerial(ctx, session, serialNumber)
}

func (lm *loggingMiddleware) RevokeAll(ctx context.Context, session authn.Session, entityId string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_all took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.RevokeAll(ctx, session, entityId)
}

func (lm *loggingMiddleware) ViewCert(ctx context.Context, session authn.Session, serialNumber string) (cert certs.Certificate, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_cert for serial number %s took %s to complete", serialNumber, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.ViewCert(ctx, session, serialNumber)
}

func (lm *loggingMiddleware) OCSP(ctx context.Context, serialNumber string, ocspRequestDER []byte) (ocspBytes []byte, err error) {
	defer func(begin time.Time) {
		requestType := "serial_number"
		if len(ocspRequestDER) > 0 {
			requestType = "raw_request"
		}
		message := fmt.Sprintf("Method ocsp (%s) for serial number %s took %s to complete", requestType, serialNumber, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.OCSP(ctx, serialNumber, ocspRequestDER)
}

func (lm *loggingMiddleware) GetEntityID(ctx context.Context, serialNumber string) (entityID string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method get_entity_id for serial number %s took %s to complete", serialNumber, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.GetEntityID(ctx, serialNumber)
}

func (lm *loggingMiddleware) GenerateCRL(ctx context.Context) (crl []byte, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method generate_crl took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.GenerateCRL(ctx)
}

func (lm *loggingMiddleware) RetrieveCAChain(ctx context.Context) (cert certs.Certificate, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method retrieve_ca_chain took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())
	return lm.svc.RetrieveCAChain(ctx)
}

func (lm *loggingMiddleware) IssueFromCSR(ctx context.Context, session authn.Session, entityID, ttl string, csr certs.CSR) (c certs.Certificate, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method issue_from_csr took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.IssueFromCSR(ctx, session, entityID, ttl, csr)
}

func (lm *loggingMiddleware) IssueFromCSRInternal(ctx context.Context, entityID, ttl string, csr certs.CSR) (c certs.Certificate, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method issue_from_csr_internal for entity %s took %s to complete", entityID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(message)
	}(time.Now())
	return lm.svc.IssueFromCSRInternal(ctx, entityID, ttl, csr)
}
