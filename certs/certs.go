// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
)

type CertType int

const (
	RootCA CertType = iota
	IntermediateCA
	ClientCert
)

const (
	Root    = "RootCA"
	Inter   = "IntermediateCA"
	Client  = "ClientCert"
	Unknown = "Unknown"
)

func (c CertType) String() string {
	switch c {
	case RootCA:
		return Root
	case IntermediateCA:
		return Inter
	case ClientCert:
		return Client
	default:
		return Unknown
	}
}

func CertTypeFromString(s string) (CertType, error) {
	switch s {
	case Root:
		return RootCA, nil
	case Inter:
		return IntermediateCA, nil
	case Client:
		return ClientCert, nil
	default:
		return -1, errors.New("unknown cert type")
	}
}

type CA struct {
	Type         CertType
	Certificate  *x509.Certificate
	PrivateKey   *rsa.PrivateKey
	SerialNumber string
}

type Certificate struct {
	SerialNumber string    `json:"serial_number"`
	Certificate  []byte    `json:"certificate"`
	Key          []byte    `json:"key"`
	Revoked      bool      `json:"revoked"`
	ExpiryTime   time.Time `json:"expiry_time"`
	EntityID     string    `json:"entity_id"`
	Type         CertType  `json:"type"`
	DownloadUrl  string    `json:"-"`
}

type CertificatePage struct {
	PageMetadata
	Certificates []Certificate
}

type PageMetadata struct {
	Total    uint64 `json:"total"`
	Offset   uint64 `json:"offset,omitempty"`
	Limit    uint64 `json:"limit,omitempty"`
	EntityID string `json:"entity_id,omitempty"`
}

type CSRMetadata struct {
	CommonName         string           `json:"common_name"`
	Organization       []string         `json:"organization"`
	OrganizationalUnit []string         `json:"organizational_unit"`
	Country            []string         `json:"country"`
	Province           []string         `json:"province"`
	Locality           []string         `json:"locality"`
	StreetAddress      []string         `json:"street_address"`
	PostalCode         []string         `json:"postal_code"`
	DNSNames           []string         `json:"dns_names"`
	IPAddresses        []string         `json:"ip_addresses"`
	EmailAddresses     []string         `json:"email_addresses"`
	ExtraExtensions    []pkix.Extension `json:"extra_extensions"`
}

type CSR struct {
	CSR        []byte `json:"csr,omitempty"`
	PrivateKey []byte `json:"private_key,omitempty"`
}

type CSRPage struct {
	PageMetadata
	CSRs []CSR `json:"csrs,omitempty"`
}

type SubjectOptions struct {
	CommonName         string   `json:"common_name"`
	Organization       []string `json:"organization"`
	OrganizationalUnit []string `json:"organizational_unit"`
	Country            []string `json:"country"`
	Province           []string `json:"province"`
	Locality           []string `json:"locality"`
	StreetAddress      []string `json:"street_address"`
	PostalCode         []string `json:"postal_code"`
	DnsNames           []string `json:"dns_names"`
	IpAddresses        []net.IP `json:"ip_addresses"`
}

type Service interface {
	// RenewCert renews a certificate by issuing a new certificate with the same parameters.
	// Returns the new certificate with extended TTL and a new serial number.
	RenewCert(ctx context.Context, session authn.Session, serialNumber string) (Certificate, error)

	// RevokeBySerial revokes a single certificate by its serial number.
	RevokeBySerial(ctx context.Context, session authn.Session, serialNumber string) error

	// RevokeAll revokes all certificates for a given entity ID.
	RevokeAll(ctx context.Context, session authn.Session, entityID string) error

	// ViewCert retrieves a certificate record from the database.
	ViewCert(ctx context.Context, session authn.Session, serialNumber string) (Certificate, error)

	// ListCerts retrieves the certificates from the database while applying filters.
	ListCerts(ctx context.Context, session authn.Session, pm PageMetadata) (CertificatePage, error)

	// IssueCert issues a certificate from the database.
	IssueCert(ctx context.Context, session authn.Session, entityID, ttl string, ipAddrs []string, option SubjectOptions) (Certificate, error)

	// OCSP forwards OCSP requests to OpenBao's OCSP endpoint.
	// If ocspRequestDER is provided, it will be used directly; otherwise, a request will be built from the serialNumber.
	OCSP(ctx context.Context, serialNumber string, ocspRequestDER []byte) ([]byte, error)

	// GetEntityID retrieves the entity ID for a certificate.
	GetEntityID(ctx context.Context, serialNumber string) (string, error)

	// GenerateCRL creates cert revocation list.
	GenerateCRL(ctx context.Context) ([]byte, error)

	// RetrieveCAChain retrieves the chain of CA i.e. root and intermediate cert concat together.
	RetrieveCAChain(ctx context.Context) (Certificate, error)

	// IssueFromCSR creates a certificate from a given CSR.
	IssueFromCSR(ctx context.Context, session authn.Session, entityID, ttl string, csr CSR) (Certificate, error)

	// IssueFromCSRInternal creates a certificate from a given CSR using agent token authentication.
	IssueFromCSRInternal(ctx context.Context, entityID, ttl string, csr CSR) (Certificate, error)
}

type Repository interface {
	// SaveCertEntityMapping saves the mapping between certificate serial number and entity ID.
	SaveCertEntityMapping(ctx context.Context, serialNumber, entityID string) error

	// GetEntityIDBySerial retrieves the entity ID for a given certificate serial number.
	GetEntityIDBySerial(ctx context.Context, serialNumber string) (string, error)

	// ListCertsByEntityID lists all certificate serial numbers for a given entity ID.
	ListCertsByEntityID(ctx context.Context, entityID string) ([]string, error)

	// RemoveCertEntityMapping removes the mapping between certificate and entity ID.
	RemoveCertEntityMapping(ctx context.Context, serialNumber string) error
}
