// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package hasher

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	"golang.org/x/crypto/scrypt"
)

var (
	errHashToken        = errors.New("failed to generate hash for token")
	errHashCompare      = errors.New("failed to generate hash for given compare string")
	errToken            = errors.New("given token and hash are not same")
	errSalt             = errors.New("failed to generate salt")
	errInvalidHashStore = errors.New("invalid stored hash format")
	errDecode           = errors.New("failed to decode")
)

var _ auth.Hasher = (*bcryptHasher)(nil)

type bcryptHasher struct{}

// New instantiates a bcrypt-based hasher implementation.
func New() auth.Hasher {
	return &bcryptHasher{}
}

func (bh *bcryptHasher) Hash(token string) (string, error) {
	salt, err := generateSalt(25)
	if err != nil {
		return "", err
	}
	// N is kept 16384 to make faster and added large salt, since PAT will be access by automation scripts in high frequency.
	hash, err := scrypt.Key([]byte(token), salt, 16384, 8, 1, 32)
	if err != nil {
		return "", errors.Wrap(errHashToken, err)
	}

	return fmt.Sprintf("%s.%s", base64.StdEncoding.EncodeToString(hash), base64.StdEncoding.EncodeToString(salt)), nil
}

func (bh *bcryptHasher) Compare(plain, hashed string) error {
	parts := strings.Split(hashed, ".")
	if len(parts) != 2 {
		return errInvalidHashStore
	}

	actHash, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return errors.Wrap(errDecode, err)
	}

	salt, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return errors.Wrap(errDecode, err)
	}

	derivedHash, err := scrypt.Key([]byte(plain), salt, 16384, 8, 1, 32)
	if err != nil {
		return errors.Wrap(errHashCompare, err)
	}

	if string(derivedHash) == string(actHash) {
		return nil
	}

	return errToken
}

func generateSalt(length int) ([]byte, error) {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, errors.Wrap(errSalt, err)
	}
	return salt, nil
}
