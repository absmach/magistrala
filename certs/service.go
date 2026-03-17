// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
)

const (
	PrivateKeyBytes              = 2048
	RootCAValidityPeriod         = time.Hour * 24 * 365
	IntermediateCAValidityPeriod = time.Hour * 24 * 90
	certValidityPeriod           = time.Hour * 24 * 30
	PrivateKey                   = "PRIVATE KEY"
	RSAPrivateKey                = "RSA PRIVATE KEY"
	ECPrivateKey                 = "EC PRIVATE KEY"
	PKCS8PrivateKey              = "PKCS8 PRIVATE KEY"
	EDPrivateKey                 = "ED25519 PRIVATE KEY"
)

var (
	ErrNotFound               = errors.New("entity not found")
	ErrConflict               = errors.New("entity already exists")
	ErrCreateEntity           = errors.New("failed to create entity")
	ErrViewEntity             = errors.New("view entity failed")
	ErrUpdateEntity           = errors.New("update entity failed")
	ErrDeleteEntity           = errors.New("delete entity failed")
	ErrMalformedEntity        = errors.New("malformed entity specification")
	ErrRootCANotFound         = errors.New("root CA not found")
	ErrIntermediateCANotFound = errors.New("intermediate CA not found")
	ErrCertExpired            = errors.New("certificate expired before renewal")
	ErrCertRevoked            = errors.New("certificate has been revoked and cannot be renewed")
	ErrCertInvalidType        = errors.New("invalid cert type")
	ErrInvalidLength          = errors.New("invalid length of serial numbers")
	ErrPrivKeyType            = errors.New("unsupported private key type")
	ErrPubKeyType             = errors.New("unsupported public key type")
	ErrFailedParse            = errors.New("failed to parse key PEM")
	ErrFailedCertCreation     = errors.New("failed to create certificate")
	ErrInvalidIP              = errors.New("invalid IP address")
)

type service struct {
	pki  Agent
	repo Repository
}

var _ Service = (*service)(nil)

func NewService(ctx context.Context, pki Agent, repo Repository) (Service, error) {
	var svc service

	svc.pki = pki
	svc.repo = repo

	return &svc, nil
}

// IssueCert generates and issues a certificate for a given entityID.
// It uses the PKI agent to generate and issue a certificate.
// The certificate is managed by OpenBao PKI internally.
// EntityType is used to customize certificate properties based on the entity type.
func (s *service) IssueCert(ctx context.Context, session authn.Session, entityID, ttl string, ipAddrs []string, options SubjectOptions) (Certificate, error) {
	cert, err := s.pki.Issue(ttl, ipAddrs, options)
	if err != nil {
		return Certificate{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	if err := s.repo.SaveCertEntityMapping(ctx, cert.SerialNumber, entityID); err != nil {
		return Certificate{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	cert.EntityID = entityID

	return cert, nil
}

func (s *service) ListCerts(ctx context.Context, session authn.Session, pm PageMetadata) (CertificatePage, error) {
	if pm.EntityID != "" {
		serialNumbers, err := s.repo.ListCertsByEntityID(ctx, pm.EntityID)
		if err != nil {
			return CertificatePage{}, errors.Wrap(ErrViewEntity, err)
		}

		certPg := CertificatePage{
			PageMetadata: pm,
			Certificates: make([]Certificate, 0),
		}

		start := pm.Offset
		end := pm.Offset + pm.Limit
		if pm.Limit == 0 {
			end = uint64(len(serialNumbers))
		}
		if start >= uint64(len(serialNumbers)) {
			return certPg, nil
		}
		if end > uint64(len(serialNumbers)) {
			end = uint64(len(serialNumbers))
		}

		for i := start; i < end; i++ {
			cert, err := s.pki.View(serialNumbers[i])
			if err != nil {
				continue
			}
			cert.EntityID = pm.EntityID
			certPg.Certificates = append(certPg.Certificates, cert)
		}

		certPg.Total = uint64(len(serialNumbers))
		return certPg, nil
	}

	certPg, err := s.pki.ListCerts(pm)
	if err != nil {
		return CertificatePage{}, errors.Wrap(ErrViewEntity, err)
	}

	for i, cert := range certPg.Certificates {
		if entityID, err := s.repo.GetEntityIDBySerial(ctx, cert.SerialNumber); err == nil {
			certPg.Certificates[i].EntityID = entityID
		}
	}

	return certPg, nil
}

func (s *service) RevokeBySerial(ctx context.Context, session authn.Session, serialNumber string) error {
	err := s.pki.Revoke(serialNumber)
	if err != nil {
		return errors.Wrap(ErrUpdateEntity, err)
	}
	return nil
}

// RevokeAll revokes all certificates for a given entity ID.
// It uses the repository to find all certificates for the entity ID, then revokes each one.
func (s *service) RevokeAll(ctx context.Context, session authn.Session, entityID string) error {
	serialNumbers, err := s.repo.ListCertsByEntityID(ctx, entityID)
	if err != nil {
		return errors.Wrap(ErrViewEntity, err)
	}

	if len(serialNumbers) == 0 {
		return errors.Wrap(ErrNotFound, fmt.Errorf("no certificates found for entity ID: %s", entityID))
	}

	for _, serialNumber := range serialNumbers {
		if err := s.pki.Revoke(serialNumber); err != nil {
			return errors.Wrap(ErrUpdateEntity, err)
		}
		if err := s.repo.RemoveCertEntityMapping(ctx, serialNumber); err != nil {
			return errors.Wrap(ErrDeleteEntity, err)
		}
	}

	return nil
}

func (s *service) ViewCert(ctx context.Context, session authn.Session, serialNumber string) (Certificate, error) {
	cert, err := s.pki.View(serialNumber)
	if err != nil {
		return Certificate{}, errors.Wrap(ErrViewEntity, err)
	}

	if entityID, err := s.repo.GetEntityIDBySerial(ctx, serialNumber); err == nil {
		cert.EntityID = entityID
	}

	return cert, nil
}

func (s *service) ViewCA(ctx context.Context) (Certificate, error) {
	caPEM, err := s.pki.GetCA()
	if err != nil {
		return Certificate{}, errors.Wrap(ErrViewEntity, err)
	}

	if len(caPEM) == 0 {
		return Certificate{}, errors.New("CA certificate PEM is empty")
	}

	block, _ := pem.Decode(caPEM)
	if block == nil {
		caPreview := string(caPEM)
		if len(caPreview) > 100 {
			caPreview = caPreview[:100] + "..."
		}
		return Certificate{}, errors.New("failed to decode CA certificate PEM - received: " + caPreview)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return Certificate{}, errors.Wrap(ErrViewEntity, err)
	}

	return Certificate{
		SerialNumber: cert.SerialNumber.String(),
		Certificate:  caPEM,
		Key:          nil,
		Revoked:      false,
		ExpiryTime:   cert.NotAfter,
		EntityID:     cert.Subject.CommonName,
		Type:         IntermediateCA,
	}, nil
}

// RenewCert renews a certificate by issuing a new certificate with the same parameters.
// Returns the new certificate with extended TTL and a new serial number.
func (s *service) RenewCert(ctx context.Context, session authn.Session, serialNumber string) (Certificate, error) {
	cert, err := s.pki.View(serialNumber)
	if err != nil {
		return Certificate{}, errors.Wrap(ErrViewEntity, err)
	}
	if cert.Revoked {
		return Certificate{}, ErrCertRevoked
	}
	newCert, err := s.pki.Renew(cert, certValidityPeriod.String())
	if err != nil {
		return Certificate{}, errors.Wrap(ErrUpdateEntity, err)
	}

	return newCert, nil
}

// OCSP forwards OCSP requests to OpenBao's OCSP endpoint.
// If ocspRequestDER is provided, it will be used directly; otherwise, a request will be built from the serialNumber.
func (s *service) OCSP(ctx context.Context, serialNumber string, ocspRequestDER []byte) ([]byte, error) {
	return s.pki.OCSP(serialNumber, ocspRequestDER)
}

func (s *service) GetEntityID(ctx context.Context, serialNumber string) (string, error) {
	entityID, err := s.repo.GetEntityIDBySerial(ctx, serialNumber)
	if err != nil {
		return "", errors.Wrap(ErrViewEntity, err)
	}
	return entityID, nil
}

func (s *service) GenerateCRL(ctx context.Context) ([]byte, error) {
	crl, err := s.pki.GetCRL()
	if err != nil {
		return nil, errors.Wrap(ErrFailedCertCreation, err)
	}
	return crl, nil
}

func (s *service) RetrieveCAChain(ctx context.Context) (Certificate, error) {
	return s.getConcatCAs(ctx)
}

func (s *service) IssueFromCSR(ctx context.Context, session authn.Session, entityID, ttl string, csr CSR) (Certificate, error) {
	cert, err := s.pki.SignCSR(csr.CSR, ttl)
	if err != nil {
		return Certificate{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	if err := s.repo.SaveCertEntityMapping(ctx, cert.SerialNumber, entityID); err != nil {
		return Certificate{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	cert.EntityID = entityID

	return cert, nil
}

func (s *service) IssueFromCSRInternal(ctx context.Context, entityID, ttl string, csr CSR) (Certificate, error) {
	cert, err := s.pki.SignCSR(csr.CSR, ttl)
	if err != nil {
		return Certificate{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	if err := s.repo.SaveCertEntityMapping(ctx, cert.SerialNumber, entityID); err != nil {
		return Certificate{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	cert.EntityID = entityID

	return cert, nil
}

func (s *service) getConcatCAs(_ context.Context) (Certificate, error) {
	caChain, err := s.pki.GetCAChain()
	if err != nil {
		return Certificate{}, errors.Wrap(ErrViewEntity, err)
	}

	block, _ := pem.Decode(caChain)
	if block == nil {
		return Certificate{}, errors.New("failed to decode CA chain PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return Certificate{}, errors.Wrap(ErrViewEntity, err)
	}

	return Certificate{
		Certificate: caChain,
		ExpiryTime:  cert.NotAfter,
	}, nil
}
