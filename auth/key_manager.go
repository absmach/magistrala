// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	"github.com/absmach/supermq/pkg/errors"
)

var (
	ErrUnsupportedKeyAlgorithm = errors.New("unsupported key algorithm")
	ErrInvalidSymmetricKey     = errors.New("invalid symmetric key")
	ErrPublicKeysNotSupported  = errors.New("public keys not supported for symmetric algorithm")
	ErrRevokedToken            = errors.NewAuthNError("token is revoked")
)

// PublicKeyInfo represents a public key for external distribution via JWKS.
// This follows RFC 7517 (JSON Web Key) specification.
type PublicKeyInfo struct {
	KeyID     string `json:"kid"`
	KeyType   string `json:"kty"`
	Algorithm string `json:"alg"`
	Use       string `json:"use,omitempty"`

	// EdDSA (Ed25519) fields
	Curve string `json:"crv,omitempty"`
	X     string `json:"x,omitempty"`

	// Future: RSA fields (n, e), ECDSA fields (x, y, crv), etc.
}

// Tokenizer handles token creation and verification for authentication.
// Implementations manage underlying cryptographic operations and key distribution.
type Tokenizer interface {
	// Issue creates a signed token string from the given key claims.
	Issue(key Key) (token string, err error)

	// Parse verifies and parses a token string (JWT or PAT), returning the extracted claims.
	// For PAT tokens (prefix "pat"), returns a Key with Type set to PersonalAccessToken.
	// For JWT tokens, performs cryptographic verification and returns the parsed claims.
	Parse(ctx context.Context, token string) (key Key, err error)

	// RetrieveJWKS returns public keys for distribution via JWKS endpoint.
	// Returns ErrPublicKeysNotSupported for symmetric tokenizers (HMAC).
	RetrieveJWKS() ([]PublicKeyInfo, error)

	// Revoke revokes a refresh token.
	Revoke(ctx context.Context, token string) error
}

// TokensCache represents a cache repository. It allows saving, checking, and removing refresh tokens.
type TokensCache interface {
	// Save saves the value in the cache.
	Save(ctx context.Context, value string) error

	// Contains checks if the value exists in the cache.
	Contains(ctx context.Context, value string) bool

	// Remove removes the value from the cache.
	Remove(ctx context.Context, value string) error
}

// TokensRepository specifies methods for persisting and checking refresh tokens in the repository.
type TokensRepository interface {
	// Save persists the token.
	Save(ctx context.Context, id string) error

	// Contains checks if token with provided ID exists.
	Contains(ctx context.Context, id string) bool
}

// IsSymmetricAlgorithm determines if the given algorithm is symmetric (HMAC-based).
// Returns true for HMAC algorithms (HS256, HS384, HS512).
// Returns false for asymmetric algorithms (EdDSA).
// Returns error for unsupported algorithms.
func IsSymmetricAlgorithm(alg string) (bool, error) {
	switch alg {
	case "EdDSA":
		return false, nil
	case "HS256", "HS384", "HS512":
		return true, nil
	default:
		return false, ErrUnsupportedKeyAlgorithm
	}
}
