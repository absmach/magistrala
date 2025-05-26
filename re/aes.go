// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"crypto/aes"
	"crypto/cipher"

	"github.com/absmach/supermq/pkg/errors"
)

var (
	errInvalidDataSize = errors.New("data is not a multiple of the block size")
	errInvalidIVSize   = errors.New("size of the IV is not the same as block size")
)

// AES CBC-128 DECRYPTION requires 3 data fields
// 1. Key (16 bytes)
// 2. Initialization Vector (IV) (16 bytes)
// 3. Encrypted Data (16 bytes or length multiple a of 16)
// The encrypted data is divided into blocks of 16 bytes (128 bits) which then operated on with the IV and Key.
func encrypt(key []byte, iv []byte, data []byte) ([]byte, error) {
	if len(iv) != aes.BlockSize {
		return nil, errInvalidIVSize
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, errInvalidDataSize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	encrypted := make([]byte, len(data))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted, data)

	return encrypted, nil
}

func decrypt(key []byte, iv []byte, encrypted []byte) ([]byte, error) {
	if len(iv) != aes.BlockSize {
		return nil, errInvalidIVSize
	}
	if len(encrypted)%aes.BlockSize != 0 {
		return nil, errInvalidDataSize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)
	return decrypted, err
}
