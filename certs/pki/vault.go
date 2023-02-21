// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package pki wraps vault client
package pki

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mitchellh/mapstructure"
)

const (
	issue  = "issue"
	cert   = "cert"
	revoke = "revoke"
)

var (
	// ErrMissingCACertificate indicates missing CA certificate
	ErrMissingCACertificate = errors.New("missing CA certificate for certificate signing")

	// ErrFailedCertCreation indicates failed to certificate creation
	ErrFailedCertCreation = errors.New("failed to create client certificate")

	// ErrFailedCertRevocation indicates failed certificate revocation
	ErrFailedCertRevocation = errors.New("failed to revoke certificate")

	errFailedVaultCertIssue = errors.New("failed to issue vault certificate")
	errFailedVaultRead      = errors.New("failed to read vault certificate")
	errFailedCertDecoding   = errors.New("failed to decode response from vault service")
	expSerialNotFound       = regexp.MustCompile(`Errors:\s*(.*?)\s*certificate with serial\s*(.*?)\s*not found`)
)

type Cert struct {
	Certificate    string    `json:"certificate" mapstructure:"certificate"`
	IssuingCA      string    `json:"issuing_ca" mapstructure:"issuing_ca"`
	CAChain        []string  `json:"ca_chain" mapstructure:"ca_chain"`
	PrivateKey     string    `json:"private_key" mapstructure:"private_key"`
	PrivateKeyType string    `json:"private_key_type" mapstructure:"private_key_type"`
	Serial         string    `json:"serial" mapstructure:"serial_number"`
	Expire         time.Time `json:"expire" mapstructure:"-"`
}

// Agent represents the Vault PKI interface.
type Agent interface {
	// IssueCert issues certificate on PKI
	IssueCert(cn string, ttl string) (Cert, error)

	// Read retrieves certificate from PKI
	Read(serial string) (Cert, error)

	// Revoke revokes certificate from PKI
	Revoke(serial string) (time.Time, error)
}

type pkiAgent struct {
	token     string
	path      string
	role      string
	host      string
	issueURL  string
	readURL   string
	revokeURL string
	client    *api.Client
}

type certReq struct {
	CommonName string `json:"common_name"`
	TTL        string `json:"ttl"`
}

type certRevokeReq struct {
	SerialNumber string `json:"serial_number"`
}

// NewVaultClient instantiates a Vault client.
func NewVaultClient(token, host, path, role string) (Agent, error) {
	conf := api.DefaultConfig()
	conf.Address = host

	client, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)
	p := pkiAgent{
		token:     token,
		host:      host,
		role:      role,
		path:      path,
		client:    client,
		issueURL:  "/" + path + "/" + issue + "/" + role,
		readURL:   "/" + path + "/" + cert + "/",
		revokeURL: "/" + path + "/" + revoke,
	}
	return &p, nil
}

func (p *pkiAgent) IssueCert(cn string, ttl string) (Cert, error) {
	cReq := certReq{
		CommonName: cn,
		TTL:        ttl,
	}

	var certIssueReq map[string]interface{}
	data, err := json.Marshal(cReq)
	if err != nil {
		return Cert{}, err
	}
	if err := json.Unmarshal(data, &certIssueReq); err != nil {
		return Cert{}, nil
	}

	s, err := p.client.Logical().Write(p.issueURL, certIssueReq)
	if err != nil {
		return Cert{}, errors.Wrap(errFailedVaultCertIssue, err)
	}

	cert := Cert{}
	if err = mapstructure.Decode(s.Data, &cert); err != nil {
		return Cert{}, errors.Wrap(errFailedCertDecoding, err)
	}
	pubCert, err := p.parseCert(cert.Certificate)
	if err != nil {
		return Cert{}, errors.Wrap(errFailedCertDecoding, err)
	}
	cert.Expire = pubCert.NotAfter

	return cert, nil
}

func (p *pkiAgent) Read(serial string) (Cert, error) {
	s, err := p.client.Logical().Read(p.readURL + serial)
	if err != nil {
		return Cert{}, errors.Wrap(errFailedVaultRead, err)
	}
	cert := Cert{}
	if err = mapstructure.Decode(s.Data, &cert); err != nil {
		return Cert{}, errors.Wrap(errFailedCertDecoding, err)
	}
	return cert, nil
}

func (p *pkiAgent) Revoke(serial string) (time.Time, error) {
	cReq := certRevokeReq{
		SerialNumber: serial,
	}

	var certRevokeReq map[string]interface{}
	data, err := json.Marshal(cReq)
	if err != nil {
		if expSerialNotFound.Match([]byte(err.Error())) {
			return time.Time{}, errors.ErrNotFound
		}
		return time.Time{}, err
	}
	if err := json.Unmarshal(data, &certRevokeReq); err != nil {
		return time.Time{}, nil
	}

	s, err := p.client.Logical().Write(p.revokeURL, certRevokeReq)
	if err != nil {
		return time.Time{}, err
	}
	rev, err := s.Data["revocation_time"].(json.Number).Float64()
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, int64(rev)*int64(time.Second)), nil
}

func (c *pkiAgent) parseCert(data string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("failed to decode client certificate")
	}
	return x509.ParseCertificate(block.Bytes)
}
