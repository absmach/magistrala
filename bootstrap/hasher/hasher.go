// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package hasher

import (
	"crypto/subtle"
	"encoding/base64"
	"strings"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
)

const (
	cost                = 10
	legacyScryptPrefix  = "scrypt$"
	legacyScryptKeyN    = 16384
	legacyScryptKeyR    = 8
	legacyScryptKeyP    = 1
	legacyScryptKeySize = 32
)

var (
	errHashExternalKey    = errors.NewServiceError("generate hash from external key failed")
	errCompareExternalKey = errors.NewServiceError("compare external key and hash failed")
	errInvalidHashStore   = errors.New("invalid stored external key hash format")
	errDecode             = errors.New("failed to decode external key hash")
)

var _ bootstrap.Hasher = (*bcryptHasher)(nil)

type bcryptHasher struct{}

// New instantiates a bcrypt-based hasher implementation.
func New() bootstrap.Hasher {
	return &bcryptHasher{}
}

func (*bcryptHasher) Hash(key string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), cost)
	if err != nil {
		return "", errors.Wrap(errHashExternalKey, err)
	}

	return string(hash), nil
}

func (*bcryptHasher) Compare(plain, hashed string) error {
	if strings.HasPrefix(hashed, legacyScryptPrefix) {
		return compareLegacyScryptHash(plain, hashed)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)); err == nil {
		return nil
	}

	// Legacy rows may still contain plaintext external keys.
	if subtle.ConstantTimeCompare([]byte(plain), []byte(hashed)) == 1 {
		return nil
	}

	return bootstrap.ErrExternalKey
}

func compareLegacyScryptHash(plain, hashed string) error {
	parts := strings.Split(strings.TrimPrefix(hashed, legacyScryptPrefix), ".")
	if len(parts) != 2 {
		return errInvalidHashStore
	}

	actualHash, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return errors.Wrap(errDecode, err)
	}

	salt, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return errors.Wrap(errDecode, err)
	}

	derivedHash, err := scrypt.Key([]byte(plain), salt, legacyScryptKeyN, legacyScryptKeyR, legacyScryptKeyP, legacyScryptKeySize)
	if err != nil {
		return errors.Wrap(errCompareExternalKey, err)
	}

	if subtle.ConstantTimeCompare(derivedHash, actualHash) == 1 {
		return nil
	}

	return bootstrap.ErrExternalKey
}
