// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package amcerts

import (
	"time"

	"github.com/absmach/certs/sdk"
)

type Cert struct {
	SerialNumber string    `json:"serial_number"`
	Certificate  string    `json:"certificate,omitempty"`
	Key          string    `json:"key,omitempty"`
	Revoked      bool      `json:"revoked"`
	ExpiryTime   time.Time `json:"expiry_time"`
	ThingID      string    `json:"entity_id"`
	DownloadUrl  string    `json:"-"`
}

type CertPage struct {
	Total        uint64 `json:"total"`
	Offset       uint64 `json:"offset"`
	Limit        uint64 `json:"limit"`
	Certificates []Cert `json:"certificates,omitempty"`
}

type Agent interface {
	Issue(entityId, ttl string, ipAddrs []string) (Cert, error)

	View(serialNumber string) (Cert, error)

	Revoke(serialNumber string) error

	ListCerts(pm sdk.PageMetadata) (CertPage, error)
}

type sdkAgent struct {
	sdk sdk.SDK
}

func NewAgent(host, certsURL string, TLSVerification bool) (Agent, error) {
	msgContentType := string(sdk.CTJSONSenML)
	certConfig := sdk.Config{
		CertsURL:        certsURL,
		HostURL:         host,
		MsgContentType:  sdk.ContentType(msgContentType),
		TLSVerification: TLSVerification,
	}

	return sdkAgent{
		sdk: sdk.NewSDK(certConfig),
	}, nil
}

func (c sdkAgent) Issue(entityId, ttl string, ipAddrs []string) (Cert, error) {
	cert, err := c.sdk.IssueCert(entityId, ttl, ipAddrs, sdk.Options{CommonName: "Magistrala"})
	if err != nil {
		return Cert{}, err
	}

	return Cert{
		SerialNumber: cert.SerialNumber,
		Certificate:  cert.Certificate,
		Revoked:      cert.Revoked,
		ExpiryTime:   cert.ExpiryTime,
		ThingID:      cert.EntityID,
	}, nil
}

func (c sdkAgent) View(serial string) (Cert, error) {
	cert, err := c.sdk.ViewCert(serial)
	if err != nil {
		return Cert{}, err
	}
	return Cert{
		SerialNumber: cert.SerialNumber,
		Certificate:  cert.Certificate,
		Key:          cert.Key,
		Revoked:      cert.Revoked,
		ExpiryTime:   cert.ExpiryTime,
		ThingID:      cert.EntityID,
	}, nil
}

func (c sdkAgent) Revoke(serial string) error {
	if err := c.sdk.RevokeCert(serial); err != nil {
		return err
	}

	return nil
}

func (c sdkAgent) ListCerts(pm sdk.PageMetadata) (CertPage, error) {
	certPage, err := c.sdk.ListCerts(pm)
	if err != nil {
		return CertPage{}, err
	}

	var crts []Cert
	for _, c := range certPage.Certificates {
		crts = append(crts, Cert{
			SerialNumber: c.SerialNumber,
			Certificate:  c.Certificate,
			Key:          c.Key,
			Revoked:      c.Revoked,
			ExpiryTime:   c.ExpiryTime,
			ThingID:      c.EntityID,
		})
	}

	return CertPage{
		Total:        certPage.Total,
		Limit:        certPage.Limit,
		Offset:       certPage.Offset,
		Certificates: crts,
	}, nil
}
