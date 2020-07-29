// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/certs/pki"
	"github.com/mainflux/mainflux/pkg/errors"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
)

var (
	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")

	// ErrMalformedEntity indicates malformed entity specification.
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	errFailedKeyCreation         = errors.New("failed to create client private key")
	errFailedDateSetting         = errors.New("failed to set date for certificate")
	errKeyBitsValueWrong         = errors.New("missing RSA bits for certificate creation")
	errMissingCACertificate      = errors.New("missing CA certificate for certificate signing")
	errFailedSerialGeneration    = errors.New("failed to generate certificate serial")
	errFailedPemKeyWrite         = errors.New("failed to write PEM key")
	errFailedPemDataWrite        = errors.New("failed to write pem data for certificate")
	errPrivateKeyUnsupportedType = errors.New("private key type is unsupported")
	errPrivateKeyEmpty           = errors.New("private key is empty")
	errFailedToRemoveCertFromDB  = errors.New("failed to remove cert serial from db")
	errFailedCertCreation        = errors.New("failed to create client certificate")
	errFailedCertRevocation      = errors.New("failed to revoke certificate")
)

var _ Service = (*certsService)(nil)

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// IssueCert issues certificate for given thing id if access is granted with token
	IssueCert(ctx context.Context, token, thingID, daysValid string, keyBits int, keyType string) (Cert, error)

	// ListCerts lists all certificates issued for given owner
	ListCerts(ctx context.Context, token string, offset, limit uint64) (Page, error)

	// RevokeCert revokes certificate for given thing
	RevokeCert(ctx context.Context, token, thingID string) (Revoke, error)
}

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
	AuthnURL       string
	AuthnTimeout   time.Duration
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
	auth      mainflux.AuthNServiceClient
	certsRepo Repository
	sdk       mfsdk.SDK
	conf      Config
	pki       pki.Agent
}

// New returns new Certs service.
func New(auth mainflux.AuthNServiceClient, certs Repository, sdk mfsdk.SDK, config Config, pki pki.Agent) Service {
	return &certsService{
		certsRepo: certs,
		sdk:       sdk,
		auth:      auth,
		conf:      config,
		pki:       pki,
	}
}

type Revoke struct {
	RevocationTime time.Time `mapstructure:"revocation_time"`
}

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
	var c Cert
	owner, err := cs.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return c, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	thing, err := cs.sdk.Thing(thingID, token)
	if err != nil {
		return c, errors.Wrap(errFailedCertCreation, err)
	}

	// If PKIHost is not set we don't use 3rd party PKI service.
	if cs.conf.PKIHost == "" {
		c.ClientCert, c.ClientKey, err = cs.certs(thing.Key, daysValid, keyBits)
		if err != nil {
			return c, errors.Wrap(errFailedCertCreation, err)
		}
		return c, err
	}

	cert, err := cs.pki.IssueCert(thingID, daysValid, keyType, keyBits)
	if err != nil {
		return c, errors.Wrap(errFailedCertCreation, err)
	}

	c.ThingID = thingID
	c.OwnerID = owner.GetValue()
	c.ClientCert = cert.ClientCert
	c.IssuingCA = cert.IssuingCA
	c.CAChain = cert.CAChain
	c.ClientKey = cert.ClientKey
	c.PrivateKeyType = cert.PrivateKeyType
	c.Serial = cert.Serial
	c.Expire = cert.Expire

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
		return revoke, errors.Wrap(errFailedCertRevocation, err)
	}

	cert, err := cs.certsRepo.RetrieveByThing(ctx, thing.ID)
	if err != nil {
		return revoke, errors.Wrap(errFailedCertRevocation, err)
	}

	r, err := cs.pki.Revoke(cert.Serial)
	if err != nil {
		return revoke, errors.Wrap(errFailedCertRevocation, err)
	}
	revoke.RevocationTime = r.RevocationTime
	if err = cs.certsRepo.Remove(context.Background(), cert.Serial); err != nil {
		return revoke, errors.Wrap(errFailedToRemoveCertFromDB, err)
	}
	return revoke, nil
}

func (cs *certsService) ListCerts(ctx context.Context, token string, offset, limit uint64) (Page, error) {
	u, err := cs.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(ErrUnauthorizedAccess, err)
	}

	return cs.certsRepo.RetrieveAll(ctx, u.GetValue(), offset, limit)
}

func (cs *certsService) certs(thingKey, daysValid string, keyBits int) (string, string, error) {
	if cs.conf.SignX509Cert == nil {
		return "", "", errors.Wrap(errFailedCertCreation, errMissingCACertificate)
	}
	if keyBits == 0 {
		return "", "", errors.Wrap(errFailedCertCreation, errKeyBitsValueWrong)
	}
	var priv interface{}
	priv, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return "", "", errors.Wrap(errFailedKeyCreation, err)
	}

	if daysValid == "" {
		daysValid = cs.conf.SignHoursValid
	}

	notBefore := time.Now()
	validFor, err := time.ParseDuration(daysValid)
	if err != nil {
		return "", "", errors.Wrap(errFailedDateSetting, err)
	}
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", errors.Wrap(errFailedSerialGeneration, err)
	}

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Mainflux"},
			CommonName:         thingKey,
			OrganizationalUnit: []string{"mainflux"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
	}

	pubKey, err := publicKey(priv)
	if err != nil {
		return "", "", errors.Wrap(errFailedCertCreation, err)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, cs.conf.SignX509Cert, pubKey, cs.conf.SignTLSCert.PrivateKey)
	if err != nil {
		return "", "", errors.Wrap(errFailedCertCreation, err)
	}

	var bw, keyOut bytes.Buffer
	buffWriter := bufio.NewWriter(&bw)
	buffKeyOut := bufio.NewWriter(&keyOut)

	if err := pem.Encode(buffWriter, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", "", errors.Wrap(errFailedPemDataWrite, err)
	}
	buffWriter.Flush()
	cert := bw.String()

	block, err := pemBlockForKey(priv)
	if err != nil {
		return "", "", errors.Wrap(errFailedPemKeyWrite, err)
	}
	if err := pem.Encode(buffKeyOut, block); err != nil {
		return "", "", errors.Wrap(errFailedPemKeyWrite, err)
	}
	buffKeyOut.Flush()
	key := keyOut.String()

	return cert, key, nil
}

func publicKey(priv interface{}) (interface{}, error) {
	if priv == nil {
		return nil, errPrivateKeyEmpty
	}
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey, nil
	case *ecdsa.PrivateKey:
		return &k.PublicKey, nil
	default:
		return nil, errPrivateKeyUnsupportedType
	}
}

func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, err
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, nil
	}
}
