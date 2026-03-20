// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package certs

import "context"

// Agent represents the PKI interface that all PKI implementations must satisfy.
type Agent interface {
	Issue(ttl string, ipAddrs []string, options SubjectOptions) (Certificate, error)
	View(serialNumber string) (Certificate, error)
	Revoke(serialNumber string) error
	ListCerts(pm PageMetadata) (CertificatePage, error)
	GetCA() ([]byte, error)
	GetCAChain() ([]byte, error)
	GetCRL() ([]byte, error)
	SignCSR(csr []byte, ttl string) (Certificate, error)
	Renew(cert Certificate, increment string) (Certificate, error)
	OCSP(serialNumber string, ocspRequestDER []byte) ([]byte, error)
	StartSecretRenewal(ctx context.Context) error
}
