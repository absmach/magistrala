// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package asymmetric_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/auth/tokenizer/asymmetric"
	smqerrors "github.com/absmach/supermq/pkg/errors"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockIDProvider struct {
	id string
}

func (m *mockIDProvider) ID() (string, error) {
	return m.id, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestNewKeyManager(t *testing.T) {
	idProvider := &mockIDProvider{id: "unused"}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "private.key")

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Key,
	}

	cases := []struct {
		name        string
		setupKey    func() string
		expectErr   bool
		expectedErr error
	}{
		{
			name: "valid PEM key",
			setupKey: func() string {
				err := os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), 0o600)
				require.NoError(t, err)
				return keyPath
			},
			expectErr: false,
		},
		{
			name: "valid raw key",
			setupKey: func() string {
				rawKeyPath := filepath.Join(tmpDir, "raw_private.key")
				err := os.WriteFile(rawKeyPath, privateKey, 0o600)
				require.NoError(t, err)
				return rawKeyPath
			},
			expectErr: false,
		},
		{
			name: "non-existent key file",
			setupKey: func() string {
				return filepath.Join(tmpDir, "nonexistent.key")
			},
			expectErr:   true,
			expectedErr: smqerrors.New("failed to load private key"),
		},
		{
			name: "invalid key size",
			setupKey: func() string {
				invalidPath := filepath.Join(tmpDir, "invalid.key")
				err := os.WriteFile(invalidPath, []byte("invalid"), 0o600)
				require.NoError(t, err)
				return invalidPath
			},
			expectErr:   true,
			expectedErr: smqerrors.New("invalid ED25519 key size"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setupKey()

			km, err := asymmetric.NewTokenizer(path, "", idProvider, newTestLogger())

			if tc.expectErr {
				assert.Error(t, err)
				if tc.expectedErr != nil {
					assert.True(t, smqerrors.Contains(err, tc.expectedErr))
				}
				assert.Nil(t, km)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, km)
			}
		})
	}
}

func TestSign(t *testing.T) {
	idProvider := &mockIDProvider{id: "unused"}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "private.key")

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Key,
	}

	err = os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), 0o600)
	require.NoError(t, err)

	km, err := asymmetric.NewTokenizer(keyPath, "", idProvider, newTestLogger())
	require.NoError(t, err)

	cases := []struct {
		name string
		key  auth.Key
	}{
		{
			name: "sign valid key with all fields",
			key: auth.Key{
				ID:        "key-id",
				Type:      auth.AccessKey,
				Issuer:    "supermq.auth",
				Subject:   "user-id",
				Role:      auth.UserRole,
				IssuedAt:  time.Now().UTC(),
				ExpiresAt: time.Now().Add(1 * time.Hour).UTC(),
				Verified:  true,
			},
		},
		{
			name: "sign key without subject",
			key: auth.Key{
				ID:        "key-id",
				Type:      auth.APIKey,
				Issuer:    "supermq.auth",
				Role:      auth.AdminRole,
				IssuedAt:  time.Now().UTC(),
				ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
				Verified:  false,
			},
		},
		{
			name: "sign key without ID",
			key: auth.Key{
				Type:      auth.AccessKey,
				Subject:   "user-id",
				Role:      auth.UserRole,
				IssuedAt:  time.Now().UTC(),
				ExpiresAt: time.Now().Add(1 * time.Hour).UTC(),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := km.Issue(tc.key)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			parts := splitJWT(token)
			assert.Equal(t, 3, len(parts), "JWT should have 3 parts")
		})
	}
}

func TestVerify(t *testing.T) {
	idProvider := &mockIDProvider{id: "unused"}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "private.key")
	kid := "private"

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Key,
	}

	err = os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), 0o600)
	require.NoError(t, err)

	km, err := asymmetric.NewTokenizer(keyPath, "", idProvider, newTestLogger())
	require.NoError(t, err)

	validKey := auth.Key{
		ID:        "key-id",
		Type:      auth.AccessKey,
		Issuer:    "supermq.auth",
		Subject:   "user-id",
		Role:      auth.UserRole,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().Add(1 * time.Hour).UTC(),
		Verified:  true,
	}

	validToken, err := km.Issue(validKey)
	require.NoError(t, err, "Signing a valid token should succeed")

	expiredKey := validKey
	expiredKey.ExpiresAt = time.Now().Add(-1 * time.Hour).UTC()
	expiredToken, err := km.Issue(expiredKey)
	require.NoError(t, err, "Creating an expired token should succeed")

	wrongIssuerKey := validKey
	wrongIssuerKey.Issuer = "wrong.issuer"

	privateJwk, err := jwk.FromRaw(privateKey)
	require.NoError(t, err)
	require.NoError(t, privateJwk.Set(jwk.AlgorithmKey, jwa.EdDSA))
	require.NoError(t, privateJwk.Set(jwk.KeyIDKey, kid))

	builder := jwt.NewBuilder()
	builder.Issuer(wrongIssuerKey.Issuer).
		Subject(wrongIssuerKey.Subject).
		IssuedAt(wrongIssuerKey.IssuedAt).
		Expiration(wrongIssuerKey.ExpiresAt).
		JwtID(wrongIssuerKey.ID).
		Claim("type", wrongIssuerKey.Type).
		Claim("role", wrongIssuerKey.Role).
		Claim("verified", wrongIssuerKey.Verified)

	wrongIssuerJWT, err := builder.Build()
	require.NoError(t, err)

	wrongIssuerTokenBytes, err := jwt.Sign(wrongIssuerJWT, jwt.WithKey(jwa.EdDSA, privateJwk))
	require.NoError(t, err)
	wrongIssuerToken := string(wrongIssuerTokenBytes)

	cases := []struct {
		name        string
		token       string
		expectErr   bool
		errContains string
	}{
		{
			name:      "verify valid token",
			token:     validToken,
			expectErr: false,
		},
		{
			name:        "verify expired token",
			token:       expiredToken,
			expectErr:   true,
			errContains: "exp",
		},
		{
			name:        "verify token with wrong issuer",
			token:       wrongIssuerToken,
			expectErr:   true,
			errContains: "invalid token issuer",
		},
		{
			name:        "verify malformed token",
			token:       "not.a.valid.jwt",
			expectErr:   true,
			errContains: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, err := km.Parse(context.Background(), tc.token)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, validKey.Subject, key.Subject)
				assert.Equal(t, validKey.Type, key.Type)
				assert.Equal(t, validKey.Role, key.Role)
			}
		})
	}
}

func TestPublicKeys(t *testing.T) {
	idProvider := &mockIDProvider{id: "unused"}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "private.key")
	kid := "private"

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Key,
	}

	err = os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), 0o600)
	require.NoError(t, err)

	km, err := asymmetric.NewTokenizer(keyPath, "", idProvider, newTestLogger())
	require.NoError(t, err)

	keys, err := km.RetrieveJWKS()
	assert.NoError(t, err)
	assert.Len(t, keys, 1)

	key := keys[0]
	assert.Equal(t, kid, key.KeyID)
	assert.Equal(t, "OKP", key.KeyType)
	assert.Equal(t, "EdDSA", key.Algorithm)
	assert.Equal(t, "sig", key.Use)
	assert.Equal(t, "Ed25519", key.Curve)
	assert.NotEmpty(t, key.X)

	decoded, err := base64.RawURLEncoding.DecodeString(key.X)
	assert.NoError(t, err, "The public key should be decoded")
	assert.Equal(t, publicKey, ed25519.PublicKey(decoded))
}

func TestSignAndVerifyRoundTrip(t *testing.T) {
	idProvider := &mockIDProvider{id: "unused"}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "private.key")

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Key,
	}

	err = os.WriteFile(keyPath, pem.EncodeToMemory(pemBlock), 0o600)
	require.NoError(t, err)

	km, err := asymmetric.NewTokenizer(keyPath, "", idProvider, newTestLogger())
	require.NoError(t, err)

	originalKey := auth.Key{
		ID:        "key-123",
		Type:      auth.AccessKey,
		Issuer:    "supermq.auth",
		Subject:   "user-456",
		Role:      auth.UserRole,
		IssuedAt:  time.Now().UTC().Truncate(time.Second),
		ExpiresAt: time.Now().Add(1 * time.Hour).UTC().Truncate(time.Second),
		Verified:  true,
	}

	token, err := km.Issue(originalKey)
	require.NoError(t, err)

	verifiedKey, err := km.Parse(context.Background(), token)
	require.NoError(t, err, "Verification of a valid key should succeed")

	assert.Equal(t, originalKey.ID, verifiedKey.ID)
	assert.Equal(t, originalKey.Type, verifiedKey.Type)
	assert.Equal(t, originalKey.Subject, verifiedKey.Subject)
	assert.Equal(t, originalKey.Role, verifiedKey.Role)
	assert.Equal(t, originalKey.Verified, verifiedKey.Verified)
	assert.WithinDuration(t, originalKey.IssuedAt, verifiedKey.IssuedAt, time.Second)
	assert.WithinDuration(t, originalKey.ExpiresAt, verifiedKey.ExpiresAt, time.Second)
}

func splitJWT(token string) []string {
	parts := []string{}
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}
