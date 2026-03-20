// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/certs/api"
	"github.com/absmach/supermq/pkg/errors"
	"golang.org/x/crypto/ocsp"
)

type downloadReq struct{}

func (req downloadReq) validate() error {
	return nil
}

type viewReq struct {
	id string
}

func (req viewReq) validate() error {
	if req.id == "" {
		return errors.Wrap(certs.ErrMalformedEntity, ErrEmptySerialNo)
	}
	return nil
}

type deleteReq struct {
	entityID string
}

func (req deleteReq) validate() error {
	if req.entityID == "" {
		return errors.Wrap(certs.ErrMalformedEntity, ErrMissingEntityID)
	}
	return nil
}

type crlReq struct{}

func (req crlReq) validate() error {
	return nil
}

type issueCertReq struct {
	entityID string               `json:"-"`
	TTL      string               `json:"ttl"`
	IpAddrs  []string             `json:"ip_addresses"`
	Options  certs.SubjectOptions `json:"options"`
}

func (req issueCertReq) validate() error {
	if req.entityID == "" {
		return errors.Wrap(certs.ErrMalformedEntity, ErrMissingEntityID)
	}

	if req.Options.CommonName == "" {
		return errors.Wrap(certs.ErrMalformedEntity, ErrMissingCommonName)
	}

	return nil
}

type listCertsReq struct {
	pm certs.PageMetadata
}

func (req listCertsReq) validate() error {
	return nil
}

type ocspReq struct {
	req          *ocsp.Request
	StatusParam  string `json:"status,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
	Certificate  string `json:"certificate,omitempty"`
}

func (req *ocspReq) validate() error {
	if req.req == nil && req.SerialNumber == "" && req.Certificate == "" {
		return certs.ErrMalformedEntity
	}

	if req.Certificate != "" {
		serialNumber, err := extractSerialFromCertContent(req.Certificate)
		if err != nil {
			return errors.Wrap(certs.ErrMalformedEntity, fmt.Errorf("failed to extract serial from certificate: %w", err))
		}
		req.SerialNumber = serialNumber
	}

	req.SerialNumber = api.NormalizeSerialNumber(req.SerialNumber)

	return nil
}

func extractSerialFromCertContent(certContent string) (string, error) {
	certData := []byte(certContent)

	block, _ := pem.Decode(certData)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse certificate: %w", err)
	}

	serialHex := cert.SerialNumber.Text(16)
	return api.NormalizeSerialNumber(serialHex), nil
}

type IssueFromCSRReq struct {
	entityID string
	ttl      string
	CSR      []byte `json:"csr"`
}

func (req IssueFromCSRReq) validate() error {
	if req.entityID == "" {
		return errors.Wrap(certs.ErrMalformedEntity, ErrMissingEntityID)
	}
	if len(req.CSR) == 0 {
		return errors.Wrap(certs.ErrMalformedEntity, ErrMissingCSR)
	}

	return nil
}

type IssueFromCSRInternalReq struct {
	entityID string
	ttl      string
	CSR      []byte `json:"csr"`
}

func (req IssueFromCSRInternalReq) validate() error {
	if req.entityID == "" {
		return errors.Wrap(certs.ErrMalformedEntity, ErrMissingEntityID)
	}
	if len(req.CSR) == 0 {
		return errors.Wrap(certs.ErrMalformedEntity, ErrMissingCSR)
	}

	return nil
}
