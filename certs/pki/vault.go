// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package pki wraps vault client
package pki

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mitchellh/mapstructure"
)

const (
	issue  = "issue"
	cert   = "cert"
	revoke = "revoke"
	apiVer = "v1"
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
)

type Cert struct {
	ClientCert     string    `json:"client_cert" mapstructure:"certificate"`
	IssuingCA      string    `json:"issuing_ca" mapstructure:"issuing_ca"`
	CAChain        []string  `json:"ca_chain" mapstructure:"ca_chain"`
	ClientKey      string    `json:"client_key" mapstructure:"private_key"`
	PrivateKeyType string    `json:"private_key_type" mapstructure:"private_key_type"`
	Serial         string    `json:"serial" mapstructure:"serial_number"`
	Expire         time.Time `json:"expire" mapstructure:"-"`
}

// Agent represents the Vault PKI interface.
type Agent interface {
	// IssueCert issues certificate on PKI
	IssueCert(cn string, ttl, keyType string, keyBits int) (Cert, error)

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
	KeyBits    int    `json:"key_bits"`
	KeyType    string `json:"key_type"`
}

type certRevokeReq struct {
	SerialNumber string `json:"serial_number"`
}

// NewVaultClient instantiates a Vault client.
func NewVaultClient(token, host, path, role string) (Agent, error) {
	conf := &api.Config{
		Address: host,
	}

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
		issueURL:  "/" + apiVer + "/" + path + "/" + issue + "/" + role,
		readURL:   "/" + apiVer + "/" + path + "/" + cert + "/",
		revokeURL: "/" + apiVer + "/" + path + "/" + revoke,
	}
	return &p, nil
}

func (p *pkiAgent) IssueCert(cn string, ttl, keyType string, keyBits int) (Cert, error) {
	cReq := certReq{
		CommonName: cn,
		TTL:        ttl,
		KeyBits:    keyBits,
		KeyType:    keyType,
	}

	r := p.client.NewRequest("POST", p.issueURL)
	if err := r.SetJSONBody(cReq); err != nil {
		return Cert{}, err
	}

	resp, err := p.client.RawRequest(r)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return Cert{}, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return Cert{}, err
		}
		return Cert{}, errors.Wrap(errFailedVaultCertIssue, err)
	}

	s, err := api.ParseSecret(resp.Body)
	if err != nil {
		return Cert{}, err
	}

	cert := Cert{}
	if err = mapstructure.Decode(s.Data, &cert); err != nil {
		return Cert{}, errors.Wrap(errFailedCertDecoding, err)
	}

	return cert, nil
}

func (p *pkiAgent) Read(serial string) (Cert, error) {
	r := p.client.NewRequest("GET", p.readURL+"/"+serial)

	resp, err := p.client.RawRequest(r)
	if err != nil {
		return Cert{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return Cert{}, err
		}
		return Cert{}, errors.Wrap(errFailedVaultRead, err)
	}

	s, err := api.ParseSecret(resp.Body)
	if err != nil {
		return Cert{}, err
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

	r := p.client.NewRequest("POST", p.revokeURL)
	if err := r.SetJSONBody(cReq); err != nil {
		return time.Time{}, err
	}

	resp, err := p.client.RawRequest(r)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return time.Time{}, err
		}
		return time.Time{}, errors.Wrap(errFailedVaultCertIssue, err)
	}

	s, err := api.ParseSecret(resp.Body)
	if err != nil {
		return time.Time{}, err
	}

	rev, err := s.Data["revocation_time"].(json.Number).Float64()
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, int64(rev)*int64(time.Millisecond)), nil
}
