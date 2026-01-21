// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"time"

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
	// For RefreshKey types, the token ID is stored as active in the cache.
	Issue(ctx context.Context, key Key) (token string, err error)

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

// TokensCache represents a cache repository for managing active refresh tokens per user.
type TokensCache interface {
	// SaveActive saves an active refresh token ID for a user with TTL and optional description.
	SaveActive(ctx context.Context, userID, tokenID, description string, ttl time.Duration) error

	// IsActive checks if the token ID is active.
	IsActive(ctx context.Context, tokenID string) (bool, error)

	// ListUserTokens lists all active token IDs with descriptions for a given user.
	ListUserTokens(ctx context.Context, userID string) ([]TokenInfo, error)

	// RemoveActive removes an active refresh token ID.
	RemoveActive(ctx context.Context, tokenID string) error
}

// TokenInfo represents information about an active refresh token.
type TokenInfo struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
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
