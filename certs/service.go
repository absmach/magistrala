// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"time"

	"github.com/absmach/certs/sdk"
	pki "github.com/absmach/magistrala/certs/pki/amcerts"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
)

var (
	// ErrFailedCertCreation failed to create certificate.
	ErrFailedCertCreation = errors.New("failed to create client certificate")

	// ErrFailedCertRevocation failed to revoke certificate.
	ErrFailedCertRevocation = errors.New("failed to revoke certificate")

	ErrFailedToRemoveCertFromDB = errors.New("failed to remove cert serial from db")

	ErrFailedReadFromPKI = errors.New("failed to read certificate from PKI")
)

var _ Service = (*certsService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// IssueCert issues certificate for given client id if access is granted with token
	IssueCert(ctx context.Context, domainID, token, clientID, ttl string) (Cert, error)

	// ListCerts lists certificates issued for a given client ID
	ListCerts(ctx context.Context, clientID string, pm PageMetadata) (CertPage, error)

	// ListSerials lists certificate serial IDs issued for a given client ID
	ListSerials(ctx context.Context, clientID string, pm PageMetadata) (CertPage, error)

	// ViewCert retrieves the certificate issued for a given serial ID
	ViewCert(ctx context.Context, serialID string) (Cert, error)

	// RevokeCert revokes a certificate for a given client ID
	RevokeCert(ctx context.Context, domainID, token, clientID string) (Revoke, error)
}

type certsService struct {
	sdk mgsdk.SDK
	pki pki.Agent
}

// New returns new Certs service.
func New(sdk mgsdk.SDK, pkiAgent pki.Agent) Service {
	return &certsService{
		sdk: sdk,
		pki: pkiAgent,
	}
}

// Revoke defines the conditions to revoke a certificate.
type Revoke struct {
	RevocationTime time.Time `mapstructure:"revocation_time"`
}

func (cs *certsService) IssueCert(ctx context.Context, domainID, token, clientID, ttl string) (Cert, error) {
	var err error

	client, err := cs.sdk.Client(clientID, domainID, token)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	cert, err := cs.pki.Issue(client.ID, ttl, []string{})
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	return Cert{
		SerialNumber: cert.SerialNumber,
		Certificate:  cert.Certificate,
		Key:          cert.Key,
		Revoked:      cert.Revoked,
		ExpiryTime:   cert.ExpiryTime,
		ClientID:     cert.ClientID,
	}, err
}

func (cs *certsService) RevokeCert(ctx context.Context, domainID, token, clientID string) (Revoke, error) {
	var revoke Revoke
	var err error

	client, err := cs.sdk.Client(clientID, domainID, token)
	if err != nil {
		return revoke, errors.Wrap(ErrFailedCertRevocation, err)
	}

	cp, err := cs.pki.ListCerts(sdk.PageMetadata{Offset: 0, Limit: 10000, EntityID: client.ID})
	if err != nil {
		return revoke, errors.Wrap(ErrFailedCertRevocation, err)
	}

	for _, c := range cp.Certificates {
		err := cs.pki.Revoke(c.SerialNumber)
		if err != nil {
			return revoke, errors.Wrap(ErrFailedCertRevocation, err)
		}
		revoke.RevocationTime = time.Now()
	}

	return revoke, nil
}

func (cs *certsService) ListCerts(ctx context.Context, clientID string, pm PageMetadata) (CertPage, error) {
	cp, err := cs.pki.ListCerts(sdk.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, EntityID: clientID})
	if err != nil {
		return CertPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	var crts []Cert

	for _, c := range cp.Certificates {
		crts = append(crts, Cert{
			SerialNumber: c.SerialNumber,
			Certificate:  c.Certificate,
			Key:          c.Key,
			Revoked:      c.Revoked,
			ExpiryTime:   c.ExpiryTime,
			ClientID:     c.ClientID,
		})
	}

	return CertPage{
		Total:        cp.Total,
		Limit:        cp.Limit,
		Offset:       cp.Offset,
		Certificates: crts,
	}, nil
}

func (cs *certsService) ListSerials(ctx context.Context, clientID string, pm PageMetadata) (CertPage, error) {
	cp, err := cs.pki.ListCerts(sdk.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, EntityID: clientID})
	if err != nil {
		return CertPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	var certs []Cert
	for _, c := range cp.Certificates {
		if (pm.Revoked == "true" && c.Revoked) || (pm.Revoked == "false" && !c.Revoked) || (pm.Revoked == "all") {
			certs = append(certs, Cert{
				SerialNumber: c.SerialNumber,
				ClientID:     c.ClientID,
				ExpiryTime:   c.ExpiryTime,
				Revoked:      c.Revoked,
			})
		}
	}

	return CertPage{
		Offset:       cp.Offset,
		Limit:        cp.Limit,
		Total:        uint64(len(certs)),
		Certificates: certs,
	}, nil
}

func (cs *certsService) ViewCert(ctx context.Context, serialID string) (Cert, error) {
	cert, err := cs.pki.View(serialID)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedReadFromPKI, err)
	}

	return Cert{
		SerialNumber: cert.SerialNumber,
		Certificate:  cert.Certificate,
		Key:          cert.Key,
		Revoked:      cert.Revoked,
		ExpiryTime:   cert.ExpiryTime,
		ClientID:     cert.ClientID,
	}, nil
}
