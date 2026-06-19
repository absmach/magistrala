// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

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
)

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

// newAtomAPIKey mints a fresh Atom-format API key for a device. Atom expects
// `atom_<32hex-credId>_<64hex-secret>` and verifies argon2 over the raw 32 secret
// bytes (see atom src/auth.rs parse_api_key / auth_from_api_key). Magistrala's
// own device secret cannot be reused (arbitrary format, looked up differently),
// so the key is re-issued and must be re-provisioned to the device.
//
// credID is derived deterministically from the client id so re-runs are
// idempotent (same credential row id), but the returned plaintext key is only
// usable from the run that generated it.
func newAtomAPIKey(clientID string) (credID, plaintextKey, secretHash string, err error) {
	cu := uuid.NewSHA1(uuidNS, []byte("devcred|"+clientID))
	credIDHex := strings.ReplaceAll(cu.String(), "-", "")

	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return "", "", "", err
	}
	hash, err := hashArgon2id(secret)
	if err != nil {
		return "", "", "", err
	}
	key := "atom_" + credIDHex + "_" + hex.EncodeToString(secret)
	return cu.String(), key, hash, nil
}
