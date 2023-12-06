// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/certs/pki"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
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
type Service interface {
	// IssueCert issues certificate for given thing id if access is granted with token
	IssueCert(ctx context.Context, token, thingID, ttl string) (Cert, error)

	// ListCerts lists certificates issued for a given thing ID
	ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error)

	// ListSerials lists certificate serial IDs issued for a given thing ID
	ListSerials(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error)

	// ViewCert retrieves the certificate issued for a given serial ID
	ViewCert(ctx context.Context, token, serialID string) (Cert, error)

	// RevokeCert revokes a certificate for a given serial ID
	RevokeCert(ctx context.Context, token, serialID string) (Revoke, error)
}

type certsService struct {
	auth      magistrala.AuthServiceClient
	certsRepo Repository
	sdk       mgsdk.SDK
	pki       pki.Agent
}

// New returns new Certs service.
func New(auth magistrala.AuthServiceClient, certs Repository, sdk mgsdk.SDK, pkiAgent pki.Agent) Service {
	return &certsService{
		certsRepo: certs,
		sdk:       sdk,
		auth:      auth,
		pki:       pkiAgent,
	}
}

// Revoke defines the conditions to revoke a certificate.
type Revoke struct {
	RevocationTime time.Time `mapstructure:"revocation_time"`
}

// Cert defines the certificate paremeters.
type Cert struct {
	OwnerID        string    `json:"owner_id" mapstructure:"owner_id"`
	ThingID        string    `json:"thing_id" mapstructure:"thing_id"`
	ClientCert     string    `json:"client_cert" mapstructure:"certificate"`
	IssuingCA      string    `json:"issuing_ca" mapstructure:"issuing_ca"`
	CAChain        []string  `json:"ca_chain" mapstructure:"ca_chain"`
	ClientKey      string    `json:"client_key" mapstructure:"private_key"`
	PrivateKeyType string    `json:"private_key_type" mapstructure:"private_key_type"`
	Serial         string    `json:"serial" mapstructure:"serial_number"`
	Expire         time.Time `json:"expire" mapstructure:"-"`
}

func (cs *certsService) IssueCert(ctx context.Context, token, thingID, ttl string) (Cert, error) {
	owner, err := cs.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return Cert{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	thing, err := cs.sdk.Thing(thingID, token)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	cert, err := cs.pki.IssueCert(thing.Credentials.Secret, ttl)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	c := Cert{
		ThingID:        thingID,
		OwnerID:        owner.GetId(),
		ClientCert:     cert.ClientCert,
		IssuingCA:      cert.IssuingCA,
		CAChain:        cert.CAChain,
		ClientKey:      cert.ClientKey,
		PrivateKeyType: cert.PrivateKeyType,
		Serial:         cert.Serial,
		Expire:         time.Unix(0, int64(cert.Expire)*int64(time.Second)),
	}

	_, err = cs.certsRepo.Save(ctx, c)
	return c, err
}

func (cs *certsService) RevokeCert(ctx context.Context, token, thingID string) (Revoke, error) {
	var revoke Revoke
	u, err := cs.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return revoke, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	thing, err := cs.sdk.Thing(thingID, token)
	if err != nil {
		return revoke, errors.Wrap(ErrFailedCertRevocation, err)
	}

	offset, limit := uint64(0), uint64(10000)
	cp, err := cs.certsRepo.RetrieveByThing(ctx, u.GetId(), thing.ID, offset, limit)
	if err != nil {
		return revoke, errors.Wrap(ErrFailedCertRevocation, err)
	}

	for _, c := range cp.Certs {
		revTime, err := cs.pki.Revoke(c.Serial)
		if err != nil {
			return revoke, errors.Wrap(ErrFailedCertRevocation, err)
		}
		revoke.RevocationTime = revTime
		if err = cs.certsRepo.Remove(ctx, u.GetId(), c.Serial); err != nil {
			return revoke, errors.Wrap(ErrFailedToRemoveCertFromDB, err)
		}
	}

	return revoke, nil
}

func (cs *certsService) ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error) {
	u, err := cs.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return Page{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	cp, err := cs.certsRepo.RetrieveByThing(ctx, u.GetId(), thingID, offset, limit)
	if err != nil {
		return Page{}, errors.Wrap(repoerr.ErrNotFound, err)
	}

	for i, cert := range cp.Certs {
		vcert, err := cs.pki.Read(cert.Serial)
		if err != nil {
			return Page{}, errors.Wrap(ErrFailedReadFromPKI, err)
		}
		cp.Certs[i].ClientCert = vcert.ClientCert
		cp.Certs[i].ClientKey = vcert.ClientKey
	}

	return cp, nil
}

func (cs *certsService) ListSerials(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error) {
	u, err := cs.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return Page{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	return cs.certsRepo.RetrieveByThing(ctx, u.GetId(), thingID, offset, limit)
}

func (cs *certsService) ViewCert(ctx context.Context, token, serialID string) (Cert, error) {
	u, err := cs.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return Cert{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	cert, err := cs.certsRepo.RetrieveBySerial(ctx, u.GetId(), serialID)
	if err != nil {
		return Cert{}, errors.Wrap(repoerr.ErrNotFound, err)
	}

	vcert, err := cs.pki.Read(serialID)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedReadFromPKI, err)
	}

	c := Cert{
		ThingID:    cert.ThingID,
		ClientCert: vcert.ClientCert,
		Serial:     cert.Serial,
		Expire:     cert.Expire,
	}

	return c, nil
}
