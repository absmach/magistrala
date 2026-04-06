// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/pkg/errors"
	"golang.org/x/crypto/ocsp"
)

const (
	certsEndpoint = "certs"
	csrEndpoint   = "csrs"
	crlEndpoint   = "crl"
)

func (sdk mgSDK) IssueCert(ctx context.Context, entityID, ttl string, ipAddrs []string, opts Options, domainID, token string) (Certificate, errors.SDKError) {
	type certReq struct {
		IpAddrs []string `json:"ip_addresses"`
		TTL     string   `json:"ttl"`
		Options Options  `json:"options"`
	}
	r := certReq{
		IpAddrs: ipAddrs,
		TTL:     ttl,
		Options: opts,
	}
	d, err := json.Marshal(r)
	if err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/%s/%s/issue/%s", sdk.certsURL, domainID, certsEndpoint, entityID)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, d, nil, http.StatusCreated)
	if sdkerr != nil {
		return Certificate{}, sdkerr
	}
	var cert Certificate
	if err := json.Unmarshal(body, &cert); err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	return cert, nil
}

func (sdk mgSDK) ViewCert(ctx context.Context, serialNumber, domainID, token string) (Certificate, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s", sdk.certsURL, domainID, certsEndpoint, serialNumber)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Certificate{}, sdkerr
	}
	var cert Certificate
	if err := json.Unmarshal(body, &cert); err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	return cert, nil
}

func (sdk mgSDK) RevokeCert(ctx context.Context, serialNumber, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/revoke", sdk.certsURL, domainID, certsEndpoint, serialNumber)
	_, _, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) RenewCert(ctx context.Context, serialNumber, domainID, token string) (Certificate, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/%s/%s/renew", sdk.certsURL, domainID, certsEndpoint, serialNumber)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPatch, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Certificate{}, sdkerr
	}
	var renewRes struct {
		Renewed     bool        `json:"renewed"`
		Certificate Certificate `json:"certificate"`
	}
	if err := json.Unmarshal(body, &renewRes); err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	return renewRes.Certificate, nil
}

func (sdk mgSDK) ListCerts(ctx context.Context, pm PageMetadata, domainID, token string) (CertificatePage, errors.SDKError) {
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s", sdk.certsURL, domainID), certsEndpoint, pm)
	if err != nil {
		return CertificatePage{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, token, nil, nil, http.StatusOK)
	if sdkerr != nil {
		return CertificatePage{}, sdkerr
	}
	var cp CertificatePage
	if err := json.Unmarshal(body, &cp); err != nil {
		return CertificatePage{}, errors.NewSDKError(err)
	}
	return cp, nil
}

func (sdk mgSDK) DeleteCert(ctx context.Context, entityID, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/delete", sdk.certsURL, domainID, certsEndpoint, entityID)
	_, _, sdkerr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) OCSP(ctx context.Context, serialNumber, cert string) (OCSPResponse, errors.SDKError) {
	if serialNumber == "" && cert == "" {
		return OCSPResponse{}, errors.NewSDKError(errors.New("either serial number or certificate must be provided"))
	}
	ocspReq := struct {
		SerialNumber string `json:"serial_number,omitempty"`
		Certificate  string `json:"certificate,omitempty"`
	}{}
	if serialNumber != "" {
		ocspReq.SerialNumber = serialNumber
	}
	if cert != "" {
		ocspReq.Certificate = cert
	}
	requestBody, err := json.Marshal(ocspReq)
	if err != nil {
		return OCSPResponse{}, errors.NewSDKError(err)
	}
	url := fmt.Sprintf("%s/certs/ocsp", sdk.certsURL)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, "", requestBody, nil, http.StatusOK)
	if sdkerr != nil {
		return OCSPResponse{}, sdkerr
	}
	ocspResp, err := ocsp.ParseResponse(body, nil)
	if err != nil {
		return OCSPResponse{}, errors.NewSDKError(fmt.Errorf("failed to parse OCSP response: %w", err))
	}
	var status CertStatus
	switch ocspResp.Status {
	case ocsp.Good:
		status = CertValid
	case ocsp.Revoked:
		status = CertRevoked
	default:
		status = CertUnknown
	}
	resp := OCSPResponse{
		Status:       status,
		SerialNumber: ocspResp.SerialNumber.String(),
		Certificate:  body,
	}
	if ocspResp.RevokedAt != (time.Time{}) {
		resp.RevokedAt = &ocspResp.RevokedAt
	}
	if ocspResp.ProducedAt != (time.Time{}) {
		resp.ProducedAt = &ocspResp.ProducedAt
	}
	if ocspResp.ThisUpdate != (time.Time{}) {
		resp.ThisUpdate = &ocspResp.ThisUpdate
	}
	if ocspResp.NextUpdate != (time.Time{}) {
		resp.NextUpdate = &ocspResp.NextUpdate
	}
	resp.RevocationReason = int(ocspResp.RevocationReason)
	return resp, nil
}

func (sdk mgSDK) ViewCA(ctx context.Context) (Certificate, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/view-ca", sdk.certsURL, certsEndpoint)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, "", nil, nil, http.StatusOK)
	if sdkerr != nil {
		return Certificate{}, sdkerr
	}
	var cert Certificate
	if err := json.Unmarshal(body, &cert); err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	return cert, nil
}

func (sdk mgSDK) DownloadCA(ctx context.Context) (CertificateBundle, errors.SDKError) {
	url := fmt.Sprintf("%s/%s/download-ca", sdk.certsURL, certsEndpoint)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, "", nil, nil, http.StatusOK)
	if sdkerr != nil {
		return CertificateBundle{}, sdkerr
	}
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return CertificateBundle{}, errors.NewSDKError(err)
	}
	var bundle CertificateBundle
	for _, file := range zipReader.File {
		fileContent, err := readZipFile(file)
		if err != nil {
			return CertificateBundle{}, errors.NewSDKError(err)
		}
		if file.Name == "ca.crt" {
			bundle.Certificate = fileContent
		}
	}
	return bundle, nil
}

func (sdk mgSDK) IssueFromCSR(ctx context.Context, entityID, ttl, csr, domainID, token string) (Certificate, errors.SDKError) {
	pm := PageMetadata{TTL: ttl}
	type csrReq struct {
		CSR []byte `json:"csr,omitempty"`
	}
	r := csrReq{CSR: []byte(csr)}
	d, err := json.Marshal(r)
	if err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/%s/%s/%s", sdk.certsURL, domainID, certsEndpoint, csrEndpoint), entityID, pm)
	if err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, d, nil, http.StatusOK)
	if sdkerr != nil {
		return Certificate{}, sdkerr
	}
	var cert Certificate
	if err := json.Unmarshal(body, &cert); err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	return cert, nil
}

func (sdk mgSDK) IssueFromCSRInternal(ctx context.Context, entityID, ttl, csr, token string) (Certificate, errors.SDKError) {
	type csrReq struct {
		CSR []byte `json:"csr,omitempty"`
	}
	r := csrReq{CSR: []byte(csr)}
	d, err := json.Marshal(r)
	if err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	pm := PageMetadata{TTL: ttl}
	url, err := sdk.withQueryParams(fmt.Sprintf("%s/certs/csrs", sdk.certsURL), entityID, pm)
	if err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodPost, url, token, d, nil, http.StatusOK)
	if sdkerr != nil {
		return Certificate{}, sdkerr
	}
	var cert Certificate
	if err := json.Unmarshal(body, &cert); err != nil {
		return Certificate{}, errors.NewSDKError(err)
	}
	return cert, nil
}

func (sdk mgSDK) GenerateCRL(ctx context.Context) ([]byte, errors.SDKError) {
	url := fmt.Sprintf("%s/certs/%s", sdk.certsURL, crlEndpoint)
	_, body, sdkerr := sdk.processRequest(ctx, http.MethodGet, url, "", nil, nil, http.StatusOK)
	if sdkerr != nil {
		return nil, sdkerr
	}
	var crlRes struct {
		CRL string `json:"crl"`
	}
	if err := json.Unmarshal(body, &crlRes); err != nil {
		return nil, errors.NewSDKError(err)
	}
	crlData, err := base64.StdEncoding.DecodeString(crlRes.CRL)
	if err != nil {
		return nil, errors.NewSDKError(err)
	}
	return crlData, nil
}

func (sdk mgSDK) RevokeAll(ctx context.Context, entityID, domainID, token string) errors.SDKError {
	url := fmt.Sprintf("%s/%s/%s/%s/delete", sdk.certsURL, domainID, certsEndpoint, entityID)
	_, _, sdkerr := sdk.processRequest(ctx, http.MethodDelete, url, token, nil, nil, http.StatusNoContent)
	return sdkerr
}

func (sdk mgSDK) EntityID(ctx context.Context, serialNumber, domainID, token string) (string, errors.SDKError) {
	cert, err := sdk.ViewCert(ctx, serialNumber, domainID, token)
	if err != nil {
		return "", err
	}
	return cert.EntityID, nil
}

// CreateCSR creates a Certificate Signing Request from the given metadata and private key.
// The private key may be a PEM-encoded []byte or a crypto.Signer (rsa, ecdsa, ed25519).
func (sdk mgSDK) CreateCSR(ctx context.Context, metadata certs.CSRMetadata, privKey any) (certs.CSR, errors.SDKError) {
	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:         metadata.CommonName,
			Organization:       metadata.Organization,
			OrganizationalUnit: metadata.OrganizationalUnit,
			Country:            metadata.Country,
			Province:           metadata.Province,
			Locality:           metadata.Locality,
			StreetAddress:      metadata.StreetAddress,
			PostalCode:         metadata.PostalCode,
		},
		EmailAddresses:  metadata.EmailAddresses,
		DNSNames:        metadata.DNSNames,
		ExtraExtensions: metadata.ExtraExtensions,
	}
	for _, ip := range metadata.IPAddresses {
		if parsed := net.ParseIP(ip); parsed != nil {
			template.IPAddresses = append(template.IPAddresses, parsed)
		}
	}
	actualKey := privKey
	if keyBytes, ok := privKey.([]byte); ok {
		var err error
		actualKey, err = extractPrivateKey(keyBytes)
		if err != nil {
			return certs.CSR{}, errors.NewSDKError(errors.Wrap(certs.ErrCreateEntity, err))
		}
	}
	var signer crypto.Signer
	switch key := actualKey.(type) {
	case *rsa.PrivateKey, *ecdsa.PrivateKey:
		signer = key.(crypto.Signer)
	case ed25519.PrivateKey:
		signer = key
	default:
		return certs.CSR{}, errors.NewSDKError(errors.Wrap(certs.ErrCreateEntity, certs.ErrPrivKeyType))
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, template, signer)
	if err != nil {
		return certs.CSR{}, errors.NewSDKError(errors.Wrap(certs.ErrCreateEntity, err))
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	return certs.CSR{CSR: csrPEM}, nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	fc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer fc.Close()
	return io.ReadAll(fc)
}

func extractPrivateKey(pemKey []byte) (any, error) {
	block, _ := pem.Decode(pemKey)
	if block == nil {
		return nil, errors.New("failed to parse private key PEM")
	}
	var (
		privateKey any
		err        error
	)
	switch block.Type {
	case certs.RSAPrivateKey:
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case certs.ECPrivateKey:
		privateKey, err = x509.ParseECPrivateKey(block.Bytes)
	case certs.PrivateKey, certs.PKCS8PrivateKey, certs.EDPrivateKey:
		privateKey, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	default:
		err = certs.ErrPrivKeyType
	}
	if err != nil {
		return nil, certs.ErrFailedParse
	}
	return privateKey, nil
}
