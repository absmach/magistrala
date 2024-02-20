// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

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
	"sync"
	"time"

	"github.com/absmach/magistrala/certs/pki"
	"github.com/absmach/magistrala/pkg/errors"
)

const keyBits = 2048

var (
	errPrivateKeyEmpty           = errors.New("private key is empty")
	errPrivateKeyUnsupportedType = errors.New("private key type is unsupported")
)

var _ pki.Agent = (*agent)(nil)

type agent struct {
	AuthTimeout time.Duration
	TLSCert     tls.Certificate
	X509Cert    *x509.Certificate
	TTL         string
	mu          sync.Mutex
	counter     uint64
	certs       map[string]pki.Cert
}

func NewPkiAgent(tlsCert tls.Certificate, caCert *x509.Certificate, ttl string, timeout time.Duration) pki.Agent {
	return &agent{
		AuthTimeout: timeout,
		TLSCert:     tlsCert,
		X509Cert:    caCert,
		TTL:         ttl,
		certs:       make(map[string]pki.Cert),
	}
}

func (a *agent) IssueCert(cn, ttl string) (pki.Cert, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.X509Cert == nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, pki.ErrMissingCACertificate)
	}

	var priv interface{}
	priv, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}

	if ttl == "" {
		ttl = a.TTL
	}

	notBefore := time.Now()
	validFor, err := time.ParseDuration(ttl)
	if err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}
	notAfter := notBefore.Add(validFor)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Magistrala"},
			CommonName:         cn,
			OrganizationalUnit: []string{"magistrala"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
	}

	pubKey, err := publicKey(priv)
	if err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, a.X509Cert, pubKey, a.TLSCert.PrivateKey)
	if err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}

	x509cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}

	var bw, keyOut bytes.Buffer
	buffWriter := bufio.NewWriter(&bw)
	buffKeyOut := bufio.NewWriter(&keyOut)

	if err := pem.Encode(buffWriter, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}
	buffWriter.Flush()
	cert := bw.String()

	block, err := pemBlockForKey(priv)
	if err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}
	if err := pem.Encode(buffKeyOut, block); err != nil {
		return pki.Cert{}, errors.Wrap(pki.ErrFailedCertCreation, err)
	}
	buffKeyOut.Flush()
	key := keyOut.String()

	a.certs[x509cert.SerialNumber.String()] = pki.Cert{
		ClientCert: cert,
	}
	a.counter++

	return pki.Cert{
		ClientCert: cert,
		ClientKey:  key,
		Serial:     x509cert.SerialNumber.String(),
		Expire:     x509cert.NotAfter.Unix(),
		IssuingCA:  x509cert.Issuer.String(),
	}, nil
}

func (a *agent) Read(serial string) (pki.Cert, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	crt, ok := a.certs[serial]
	if !ok {
		return pki.Cert{}, errors.ErrNotFound
	}

	return crt, nil
}

func (a *agent) Revoke(serial string) (time.Time, error) {
	return time.Now(), nil
}

func (a *agent) LoginAndRenew(ctx context.Context) error {
	return nil
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
