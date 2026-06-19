// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

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
