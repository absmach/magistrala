// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

// Atom uses argon2 crate 0.5 `Argon2::default()` => argon2id, v=19,
// m=19456 KiB, t=2, p=1, 32-byte tag. We emit the matching PHC string so the
// hash verifies in Atom.
const (
	argonMemory  = 19456
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
	aeadNonceLen = 12
)

const sharedKeyAEADAlg = "AES-256-GCM"

type sharedKeyMaterial struct {
	Hash       string
	Ciphertext []byte
	Nonce      []byte
	KeyID      string
	EncAlg     string
	LookupHash []byte
}

// hashArgon2id produces a PHC-encoded argon2id hash compatible with Atom.
func hashArgon2id(secret []byte) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey(secret, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64 := base64.RawStdEncoding // PHC uses unpadded standard base64
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		b64.EncodeToString(salt), b64.EncodeToString(key),
	), nil
}

func newSharedKeyMaterial(credentialID, secret string, cfg config) (sharedKeyMaterial, error) {
	if len(cfg.AtomKeyEncryptionKey) != 32 {
		return sharedKeyMaterial{}, fmt.Errorf("ATOM_KEY_ENCRYPTION_KEY is required to migrate client shared keys")
	}
	hash, err := hashArgon2id([]byte(secret))
	if err != nil {
		return sharedKeyMaterial{}, err
	}
	credUUID, err := uuid.Parse(credentialID)
	if err != nil {
		return sharedKeyMaterial{}, err
	}
	block, err := aes.NewCipher(cfg.AtomKeyEncryptionKey)
	if err != nil {
		return sharedKeyMaterial{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return sharedKeyMaterial{}, err
	}
	nonce := make([]byte, aeadNonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return sharedKeyMaterial{}, err
	}
	ciphertext := aead.Seal(nil, nonce, []byte(secret), credUUID[:])

	mac := hmac.New(sha256.New, cfg.AtomKeyEncryptionKey)
	if _, err := mac.Write([]byte(secret)); err != nil {
		return sharedKeyMaterial{}, err
	}

	return sharedKeyMaterial{
		Hash:       hash,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		KeyID:      cfg.AtomKeyEncryptionKeyID,
		EncAlg:     sharedKeyAEADAlg,
		LookupHash: mac.Sum(nil),
	}, nil
}
