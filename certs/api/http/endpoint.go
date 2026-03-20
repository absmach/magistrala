// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/pkg/authn"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func renewCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return renewCertRes{}, err
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return renewCertRes{}, svcerr.ErrAuthentication
		}

		cert, err := svc.RenewCert(ctx, session, req.id)
		if err != nil {
			return renewCertRes{}, err
		}

		return renewCertRes{renewed: true, Certificate: cert}, nil
	}
}

func revokeCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return revokeCertRes{revoked: false}, err
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return revokeCertRes{revoked: false}, svcerr.ErrAuthentication
		}

		if err = svc.RevokeBySerial(ctx, session, req.id); err != nil {
			return revokeCertRes{revoked: false}, err
		}

		return revokeCertRes{revoked: true}, nil
	}
}

func deleteCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(deleteReq)
		if err := req.validate(); err != nil {
			return deleteCertRes{deleted: false}, err
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return deleteCertRes{deleted: false}, svcerr.ErrAuthentication
		}

		if err = svc.RevokeAll(ctx, session, req.entityID); err != nil {
			return deleteCertRes{deleted: false}, err
		}

		return deleteCertRes{deleted: true}, nil
	}
}

func issueCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(issueCertReq)
		if err := req.validate(); err != nil {
			return issueCertRes{}, err
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return issueCertRes{}, svcerr.ErrAuthentication
		}

		cert, err := svc.IssueCert(ctx, session, req.entityID, req.TTL, req.IpAddrs, req.Options)
		if err != nil {
			return issueCertRes{}, err
		}

		return issueCertRes{
			SerialNumber: cert.SerialNumber,
			Certificate:  string(cert.Certificate),
			Key:          string(cert.Key),
			ExpiryTime:   cert.ExpiryTime,
			EntityID:     cert.EntityID,
			Revoked:      cert.Revoked,
			issued:       true,
		}, nil
	}
}

func listCertsEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(listCertsReq)
		if err := req.validate(); err != nil {
			return listCertsRes{}, err
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return listCertsRes{}, svcerr.ErrAuthentication
		}

		certPage, err := svc.ListCerts(ctx, session, req.pm)
		if err != nil {
			return listCertsRes{}, err
		}

		var crts []viewCertRes
		for _, c := range certPage.Certificates {
			crts = append(crts, viewCertRes{
				SerialNumber: c.SerialNumber,
				Revoked:      c.Revoked,
				EntityID:     c.EntityID,
				ExpiryTime:   c.ExpiryTime,
			})
		}

		return listCertsRes{
			Total:        certPage.Total,
			Offset:       certPage.Offset,
			Limit:        certPage.Limit,
			Certificates: crts,
		}, nil
	}
}

func viewCertEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return viewCertRes{}, err
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return viewCertRes{}, svcerr.ErrAuthentication
		}

		cert, err := svc.ViewCert(ctx, session, req.id)
		if err != nil {
			return viewCertRes{}, err
		}

		return viewCertRes{
			SerialNumber: cert.SerialNumber,
			Certificate:  string(cert.Certificate),
			Key:          string(cert.Key),
			Revoked:      cert.Revoked,
			ExpiryTime:   cert.ExpiryTime,
			EntityID:     cert.EntityID,
		}, nil
	}
}

func ocspEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(ocspReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var resBytes []byte
		if req.SerialNumber != "" {
			resBytes, err = svc.OCSP(ctx, req.SerialNumber, nil)
			if err != nil {
				return nil, err
			}
		} else {
			ocspRequestDER, err := req.req.Marshal()
			if err != nil {
				return nil, err
			}
			resBytes, err = svc.OCSP(ctx, "", ocspRequestDER)
			if err != nil {
				return nil, err
			}
		}

		return ocspRawRes{
			Data: resBytes,
		}, nil
	}
}

func generateCRLEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(crlReq)
		if err := req.validate(); err != nil {
			return crlRes{}, err
		}

		crlBytes, err := svc.GenerateCRL(ctx)
		if err != nil {
			return crlRes{}, err
		}

		return crlRes{
			CrlBytes: crlBytes,
		}, nil
	}
}

func downloadCAEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(downloadReq)
		if err := req.validate(); err != nil {
			return fileDownloadRes{}, err
		}

		cert, err := svc.RetrieveCAChain(ctx)
		if err != nil {
			return fileDownloadRes{}, err
		}

		return fileDownloadRes{
			Certificate: cert.Certificate,
			Filename:    "ca.zip",
			ContentType: "application/zip",
		}, nil
	}
}

func viewCAEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(downloadReq)
		if err := req.validate(); err != nil {
			return viewCertRes{}, err
		}

		cert, err := svc.RetrieveCAChain(ctx)
		if err != nil {
			return viewCertRes{}, err
		}

		return viewCertRes{
			SerialNumber: cert.SerialNumber,
			Certificate:  string(cert.Certificate),
			Revoked:      cert.Revoked,
			ExpiryTime:   cert.ExpiryTime,
			EntityID:     cert.EntityID,
		}, nil
	}
}

func issueFromCSREndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(IssueFromCSRReq)
		if err := req.validate(); err != nil {
			return issueFromCSRRes{}, err
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return issueFromCSRRes{}, svcerr.ErrAuthentication
		}

		cert, err := svc.IssueFromCSR(ctx, session, req.entityID, req.ttl, certs.CSR{CSR: req.CSR})
		if err != nil {
			return issueFromCSRRes{}, err
		}

		return issueFromCSRRes{
			SerialNumber: cert.SerialNumber,
			Certificate:  string(cert.Certificate),
			Revoked:      cert.Revoked,
			ExpiryTime:   cert.ExpiryTime,
			EntityID:     cert.EntityID,
		}, nil
	}
}

func issueFromCSRInternalEndpoint(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (response any, err error) {
		req := request.(IssueFromCSRInternalReq)
		if err := req.validate(); err != nil {
			return issueFromCSRRes{}, err
		}

		cert, err := svc.IssueFromCSRInternal(ctx, req.entityID, req.ttl, certs.CSR{CSR: req.CSR})
		if err != nil {
			return issueFromCSRRes{}, err
		}

		return issueFromCSRRes{
			SerialNumber: cert.SerialNumber,
			Certificate:  string(cert.Certificate),
			Revoked:      cert.Revoked,
			ExpiryTime:   cert.ExpiryTime,
			EntityID:     cert.EntityID,
		}, nil
	}
}
