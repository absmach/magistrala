// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
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
	// previous holds retired keys by ID. They are never used to seal, only
	// to open envelopes written before a rotation.
	previous map[string][]byte
}

// PreviousKey is a retired master key, kept so that secrets sealed under it
// remain readable after the active key has been rotated.
type PreviousKey struct {
	ID  string
	Key []byte
}

func (sc *SecretCipher) KeyID() string {
	return sc.keyID
}

// NewSecretCipher creates an at-rest secret cipher. Bootstrap deliberately
// requires a 256-bit master key.
//
// Rotation is performed by promoting the new key to (masterKey, keyID) and
// passing the outgoing one as a PreviousKey: new writes use the active key
// while existing envelopes stay readable. Without that, every secret sealed
// under the old key ID would fail to open and the service would not start.
func NewSecretCipher(masterKey []byte, keyID string, previous ...PreviousKey) (*SecretCipher, error) {
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("database encryption key must contain exactly 32 bytes")
	}
	if strings.TrimSpace(keyID) == "" {
		return nil, fmt.Errorf("database encryption key ID is required")
	}
	sc := &SecretCipher{
		masterKey: append([]byte(nil), masterKey...),
		keyID:     keyID,
		previous:  make(map[string][]byte, len(previous)),
	}
	for _, p := range previous {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			return nil, fmt.Errorf("previous database encryption key ID is required")
		}
		if len(p.Key) != 32 {
			return nil, fmt.Errorf("previous database encryption key %q must contain exactly 32 bytes", id)
		}
		if id == keyID {
			return nil, fmt.Errorf("previous database encryption key %q duplicates the active key ID", id)
		}
		if _, ok := sc.previous[id]; ok {
			return nil, fmt.Errorf("duplicate previous database encryption key ID %q", id)
		}
		sc.previous[id] = append([]byte(nil), p.Key...)
	}
	return sc, nil
}

// masterKeyFor resolves the master key an envelope was sealed under.
func (sc *SecretCipher) masterKeyFor(keyID string) ([]byte, bool) {
	if keyID == sc.keyID {
		return sc.masterKey, true
	}
	key, ok := sc.previous[keyID]
	return key, ok
}

func (sc *SecretCipher) seal(purpose string, plain []byte, aad string) (string, error) {
	aead, err := sc.aead(sc.masterKey, purpose)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := aead.Seal(nil, nonce, plain, []byte(aad))
	payload := make([]byte, len(nonce)+len(ciphertext))
	copy(payload, nonce)
	copy(payload[len(nonce):], ciphertext)
	return strings.Join([]string{
		databaseEnvelopeVersion,
		sc.keyID,
		base64.RawURLEncoding.EncodeToString(payload),
	}, "."), nil
}

func (sc *SecretCipher) open(purpose, envelope, aad string) ([]byte, error) {
	parts := strings.Split(envelope, ".")
	if len(parts) != 3 || parts[0] != databaseEnvelopeVersion {
		return nil, fmt.Errorf("invalid database secret envelope")
	}
	masterKey, ok := sc.masterKeyFor(parts[1])
	if !ok {
		return nil, fmt.Errorf("database secret sealed under unknown key ID %q", parts[1])
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid database secret envelope: %w", err)
	}
	aead, err := sc.aead(masterKey, purpose)
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

func (sc *SecretCipher) aead(masterKey []byte, purpose string) (cipher.AEAD, error) {
	key := make([]byte, 32)
	reader := hkdf.New(sha256.New, masterKey, []byte("magistrala-bootstrap-db-v1"), []byte(purpose))
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
	return strings.Join([]string{"bootstrap-config", cfg.WorkspaceID, cfg.ID, cfg.ExternalID}, ":")
}

func snapshotSecretAAD(configID, slot string) string {
	return strings.Join([]string{"bootstrap-binding-snapshot", configID, slot}, ":")
}

// decoyKeyVersionSpread bounds the synthetic key version handed to unknown or
// disabled enrollments. Real versions start at 1 and advance on each external
// key rotation, so decoys are drawn from the same low range.
const decoyKeyVersionSpread = 4

// decoyKeyVersion derives a stable, unpredictable key version for an external
// ID that has no usable enrollment.
//
// The challenge endpoint is unauthenticated, so its answer must not reveal
// whether an enrollment exists. Returning a constant version made any other
// value proof of a real, rotated enrollment; deriving it from the external ID
// under the service's own key removes that inference while keeping repeated
// probes of the same ID consistent.
//
// This closes the response-content channel only. Distinguishing by response
// time (the real path performs a challenge insert) is left to rate limiting
// at the edge.
func (bs bootstrapService) decoyKeyVersion(externalID string) uint64 {
	if bs.dbCipher == nil {
		return 1
	}
	mac := hmac.New(sha256.New, bs.dbCipher.masterKey)
	mac.Write([]byte("bootstrap-decoy-key-version:"))
	mac.Write([]byte(externalID))
	sum := mac.Sum(nil)
	return 1 + uint64(binary.BigEndian.Uint32(sum[:4])%decoyKeyVersionSpread)
}
