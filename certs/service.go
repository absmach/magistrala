// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/certs/pki"
	"github.com/mainflux/mainflux/pkg/errors"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
)

var (
	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")

	// ErrMalformedEntity indicates malformed entity specification
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrFailedCertCreation failed to create certificate
	ErrFailedCertCreation = errors.New("failed to create client certificate")

	// ErrFailedCertRevocation failed to revoke certificate
	ErrFailedCertRevocation = errors.New("failed to revoke certificate")

	errFailedToRemoveCertFromDB = errors.New("failed to remove cert serial from db")
)

var _ Service = (*certsService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// IssueCert issues certificate for given thing id if access is granted with token
	IssueCert(ctx context.Context, token, thingID, daysValid string, keyBits int, keyType string) (Cert, error)

	// ListCerts lists all certificates issued for given owner
	ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error)

	// RevokeCert revokes certificate for given thing
	RevokeCert(ctx context.Context, token, thingID string) (Revoke, error)
}

// Config defines the service parameters
type Config struct {
	LogLevel       string
	ClientTLS      bool
	CaCerts        string
	HTTPPort       string
	ServerCert     string
	ServerKey      string
	BaseURL        string
	ThingsPrefix   string
	JaegerURL      string
	AuthURL        string
	AuthTimeout    time.Duration
	SignTLSCert    tls.Certificate
	SignX509Cert   *x509.Certificate
	SignRSABits    int
	SignHoursValid string
	PKIHost        string
	PKIPath        string
	PKIRole        string
	PKIToken       string
}

type certsService struct {
	auth      mainflux.AuthServiceClient
	certsRepo Repository
	sdk       mfsdk.SDK
	conf      Config
	pki       pki.Agent
}

// New returns new Certs service.
func New(auth mainflux.AuthServiceClient, certs Repository, sdk mfsdk.SDK, config Config, pki pki.Agent) Service {
	return &certsService{
		certsRepo: certs,
		sdk:       sdk,
		auth:      auth,
		conf:      config,
		pki:       pki,
	}
}

// Revoke defines the conditions to revoke a certificate
type Revoke struct {
	RevocationTime time.Time `mapstructure:"revocation_time"`
}

// Cert defines the certificate paremeters
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

func (cs *certsService) IssueCert(ctx context.Context, token, thingID string, daysValid string, keyBits int, keyType string) (Cert, error) {
	owner, err := cs.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Cert{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	thing, err := cs.sdk.Thing(thingID, token)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	cert, err := cs.pki.IssueCert(thing.Key, daysValid, keyType, keyBits)
	if err != nil {
		return Cert{}, errors.Wrap(ErrFailedCertCreation, err)
	}

	c := Cert{
		ThingID:        thingID,
		OwnerID:        owner.GetEmail(),
		ClientCert:     cert.ClientCert,
		IssuingCA:      cert.IssuingCA,
		CAChain:        cert.CAChain,
		ClientKey:      cert.ClientKey,
		PrivateKeyType: cert.PrivateKeyType,
		Serial:         cert.Serial,
		Expire:         cert.Expire,
	}

	_, err = cs.certsRepo.Save(context.Background(), c)
	return c, err
}

func (cs *certsService) RevokeCert(ctx context.Context, token, thingID string) (Revoke, error) {
	var revoke Revoke
	_, err := cs.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return revoke, errors.Wrap(ErrUnauthorizedAccess, err)
	}
	thing, err := cs.sdk.Thing(thingID, token)
	if err != nil {
		return revoke, errors.Wrap(ErrFailedCertRevocation, err)
	}

	cert, err := cs.certsRepo.RetrieveByThing(ctx, thing.ID)
	if err != nil {
		return revoke, errors.Wrap(ErrFailedCertRevocation, err)
	}

	revTime, err := cs.pki.Revoke(cert.Serial)
	if err != nil {
		return revoke, errors.Wrap(ErrFailedCertRevocation, err)
	}
	revoke.RevocationTime = revTime
	if err = cs.certsRepo.Remove(context.Background(), cert.Serial); err != nil {
		return revoke, errors.Wrap(errFailedToRemoveCertFromDB, err)
	}
	return revoke, nil
}

func (cs *certsService) ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (Page, error) {
	u, err := cs.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return cs.certsRepo.RetrieveAll(ctx, u.GetEmail(), thingID, offset, limit)
}
