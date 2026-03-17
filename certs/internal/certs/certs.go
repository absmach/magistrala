// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/crypto/ocsp"
)

var (
	ErrCertExpired = errors.New("certificate expired before renewal")
	ErrCertRevoked = errors.New("certificate has been revoked and cannot be renewed")
	ErrUnkonwn     = errors.New("certificate status unknown")
)

type OCSP struct {
	certsURI string
}

func New(certsURI string) *OCSP {
	return &OCSP{
		certsURI: fmt.Sprintf("%s/certs/ocsp", certsURI),
	}
}

func (o *OCSP) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	req, err := ocsp.CreateRequest(verifiedChains[0][0], verifiedChains[0][1], &ocsp.RequestOptions{Hash: crypto.SHA256})
	if err != nil {
		return err
	}
	httpRequest, err := http.NewRequest(http.MethodPost, o.certsURI, bytes.NewBuffer(req))
	if err != nil {
		return err
	}
	ocspURL, err := url.Parse(o.certsURI)
	if err != nil {
		return err
	}
	httpRequest.Header.Add("Content-Type", "application/ocsp-request")
	httpRequest.Header.Add("Accept", "application/ocsp-response")
	httpRequest.Header.Add("host", ocspURL.Host)

	httpClient := &http.Client{}
	httpResponse, err := httpClient.Do(httpRequest)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	output, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return err
	}

	ocspResponse, err := ocsp.ParseResponseForCert(output, verifiedChains[0][0], verifiedChains[0][1])
	if err != nil {
		return err
	}
	switch ocspResponse.Status {
	case ocsp.Good:
		return nil
	case ocsp.Revoked:
		if ocspResponse.RevocationReason == ocsp.Unspecified {
			return ErrCertRevoked
		}
		return ErrCertExpired
	case ocsp.Unknown:
		return ErrUnkonwn
	}

	return nil
}
