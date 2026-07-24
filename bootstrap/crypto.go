// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

const databaseEnvelopeVersion = "dbv1"

// SecretCipher encrypts Bootstrap secrets at rest. A purpose-specific AES key
// is derived from the database master key so config keys and binding snapshots
// do not share the same encryption key.
type SecretCipher struct {
	masterKey []byte
	keyID     string
}

func (sc *SecretCipher) KeyID() string {
	return sc.keyID
}

// NewSecretCipher creates an at-rest secret cipher. Bootstrap deliberately
// requires a 256-bit master key.
func NewSecretCipher(masterKey []byte, keyID string) (*SecretCipher, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("database encryption key must contain exactly 32 bytes")
	}
	if strings.TrimSpace(keyID) == "" {
		return nil, fmt.Errorf("database encryption key ID is required")
	}
	return &SecretCipher{masterKey: append([]byte(nil), masterKey...), keyID: keyID}, nil
}

func (sc *SecretCipher) seal(purpose string, plain []byte, aad string) (string, error) {
	aead, err := sc.aead(purpose)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := aead.Seal(nil, nonce, plain, []byte(aad))
	payload := append(nonce, ciphertext...)
	return strings.Join([]string{
		databaseEnvelopeVersion,
		sc.keyID,
		base64.RawURLEncoding.EncodeToString(payload),
	}, "."), nil
}

func (sc *SecretCipher) open(purpose, envelope, aad string) ([]byte, error) {
	parts := strings.Split(envelope, ".")
	if len(parts) != 3 || parts[0] != databaseEnvelopeVersion || parts[1] != sc.keyID {
		return nil, fmt.Errorf("invalid database secret envelope")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid database secret envelope: %w", err)
	}
	aead, err := sc.aead(purpose)
	if err != nil {
		return nil, err
	}
	if len(payload) < aead.NonceSize()+aead.Overhead() {
		return nil, fmt.Errorf("invalid database secret envelope")
	}
	nonce, ciphertext := payload[:aead.NonceSize()], payload[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt database secret: %w", err)
	}
	return plain, nil
}

func (sc *SecretCipher) aead(purpose string) (cipher.AEAD, error) {
	key := make([]byte, 32)
	reader := hkdf.New(sha256.New, sc.masterKey, []byte("magistrala-bootstrap-db-v1"), []byte(purpose))
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func configSecretAAD(cfg Config) string {
	return strings.Join([]string{"bootstrap-config", cfg.DomainID, cfg.ID, cfg.ExternalID}, ":")
}

func snapshotSecretAAD(configID, slot string) string {
	return strings.Join([]string{"bootstrap-binding-snapshot", configID, slot}, ":")
}
