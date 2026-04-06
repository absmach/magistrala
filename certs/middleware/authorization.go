// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	crt "github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
)

var _ crt.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	authz authz.Authorization
	svc   crt.Service
}

func AuthorizationMiddleware(authz authz.Authorization, svc crt.Service) crt.Service {
	return &authorizationMiddleware{authz, svc}
}

func (am *authorizationMiddleware) RenewCert(ctx context.Context, session authn.Session, serialNumber string) (crt.Certificate, error) {
	if err := am.checkUserDomainPermission(ctx, session, policies.MembershipPermission); err != nil {
		return crt.Certificate{}, err
	}
	return am.svc.RenewCert(ctx, session, serialNumber)
}

func (am *authorizationMiddleware) RevokeBySerial(ctx context.Context, session authn.Session, serialNumber string) error {
	if err := am.checkUserDomainPermission(ctx, session, policies.AdminPermission); err != nil {
		return err
	}
	return am.svc.RevokeBySerial(ctx, session, serialNumber)
}

func (am *authorizationMiddleware) RevokeAll(ctx context.Context, session authn.Session, entityID string) error {
	if err := am.checkUserDomainPermission(ctx, session, policies.AdminPermission); err != nil {
		return err
	}
	return am.svc.RevokeAll(ctx, session, entityID)
}

func (am *authorizationMiddleware) IssueCert(ctx context.Context, session authn.Session, entityID, ttl string, ipAddrs []string, options crt.SubjectOptions) (crt.Certificate, error) {
	if err := am.checkUserDomainPermission(ctx, session, policies.MembershipPermission); err != nil {
		return crt.Certificate{}, err
	}
	return am.svc.IssueCert(ctx, session, entityID, ttl, ipAddrs, options)
}

func (am *authorizationMiddleware) ListCerts(ctx context.Context, session authn.Session, pm crt.PageMetadata) (crt.CertificatePage, error) {
	if err := am.checkUserDomainPermission(ctx, session, policies.MembershipPermission); err != nil {
		return crt.CertificatePage{}, err
	}
	return am.svc.ListCerts(ctx, session, pm)
}

func (am *authorizationMiddleware) ViewCert(ctx context.Context, session authn.Session, serialNumber string) (crt.Certificate, error) {
	if err := am.checkUserDomainPermission(ctx, session, policies.MembershipPermission); err != nil {
		return crt.Certificate{}, err
	}
	return am.svc.ViewCert(ctx, session, serialNumber)
}

func (am *authorizationMiddleware) GetEntityID(ctx context.Context, serialNumber string) (string, error) {
	return am.svc.GetEntityID(ctx, serialNumber)
}

func (am *authorizationMiddleware) OCSP(ctx context.Context, serialNumber string, ocspRequestDER []byte) ([]byte, error) {
	return am.svc.OCSP(ctx, serialNumber, ocspRequestDER)
}

func (am *authorizationMiddleware) GenerateCRL(ctx context.Context) ([]byte, error) {
	return am.svc.GenerateCRL(ctx)
}

func (am *authorizationMiddleware) RetrieveCAChain(ctx context.Context) (crt.Certificate, error) {
	return am.svc.RetrieveCAChain(ctx)
}

func (am *authorizationMiddleware) IssueFromCSR(ctx context.Context, session authn.Session, entityID, ttl string, csr crt.CSR) (crt.Certificate, error) {
	if err := am.checkUserDomainPermission(ctx, session, policies.MembershipPermission); err != nil {
		return crt.Certificate{}, err
	}
	return am.svc.IssueFromCSR(ctx, session, entityID, ttl, csr)
}

func (am *authorizationMiddleware) IssueFromCSRInternal(ctx context.Context, entityID, ttl string, csr crt.CSR) (crt.Certificate, error) {
	return am.svc.IssueFromCSRInternal(ctx, entityID, ttl, csr)
}

func (am *authorizationMiddleware) checkUserDomainPermission(ctx context.Context, session authn.Session, permission string) error {
	req := authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Permission:  permission,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}
	if err := am.authz.Authorize(ctx, req, nil); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return nil
}
