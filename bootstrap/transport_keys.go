// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"encoding/base64"
	"time"
)

const (
	TransportKeyActive   = "active"
	TransportKeyRetiring = "retiring"
	TransportKeyRevoked  = "revoked"
)

// DomainTransportKey is the domain-scoped key used only for encrypted device
// Bootstrap requests and responses. EncryptedSecret is never exposed by the
// management API; Secret is populated only on create, rotate and reveal.
type DomainTransportKey struct {
	DomainID        string     `json:"domain_id,omitempty"`
	KeyID           string     `json:"key_id"`
	EncryptedSecret string     `json:"-"`
	WrappingKeyID   string     `json:"wrapping_key_id,omitempty"`
	Status          string     `json:"status"`
	Secret          string     `json:"secret,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	RetireAt        *time.Time `json:"retire_at,omitempty"`
}

// DomainTransportKeyRepository persists encrypted per-domain transport keys
// and replay identifiers for secure Bootstrap requests.
type DomainTransportKeyRepository interface {
	Create(ctx context.Context, key DomainTransportKey) error
	RetrieveCurrent(ctx context.Context, domainID string) (DomainTransportKey, error)
	Retrieve(ctx context.Context, domainID, keyID string) (DomainTransportKey, error)
	Rotate(ctx context.Context, oldKeyID string, next DomainTransportKey, retireAt time.Time) error
	ConsumeRequestID(ctx context.Context, domainID, keyID, requestID string, expiresAt time.Time) error
}

type SecureBootstrapRequest struct {
	ExternalID  string `json:"external_id"`
	ExternalKey string `json:"external_key"`
	IssuedAt    int64  `json:"issued_at"`
	RequestID   string `json:"request_id"`
}

// SecureBootstrapCredential is a short-lived, single-use credential for the
// encrypted device Bootstrap endpoint. It is generated on demand and is never
// persisted.
type SecureBootstrapCredential struct {
	ExternalID           string    `json:"external_id"`
	KeyID                string    `json:"key_id"`
	EncryptedExternalKey string    `json:"encrypted_external_key"`
	Authorization        string    `json:"authorization"`
	RequestID            string    `json:"request_id"`
	ExpiresAt            time.Time `json:"expires_at"`
	BootstrapURL         string    `json:"bootstrap_url"`
}

func encodeTransportSecret(secret []byte) string {
	return base64.RawURLEncoding.EncodeToString(secret)
}
