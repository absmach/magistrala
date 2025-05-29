// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

// encrypt implements AES CBC-128 ENCRYPTION which requires 3 data fields
// 1. Key (16 bytes)
// 2. Initialization Vector (IV) (16 bytes)
// 3. Encrypted Data (16 bytes or length multiple a of 16)
// The encrypted data is divided into blocks of 16 bytes (128 bits) which then operated on with the IV and Key.
func encrypt(key []byte, iv []byte, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	if len(data)%blockSize != 0 {
		return nil, fmt.Errorf("payload length %d is not a multiple of AES block size %d", len(data), blockSize)
	}

	if len(iv) != blockSize {
		return nil, fmt.Errorf("size of the IV %d is not the same as block size %d", len(iv), blockSize)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	encrypted := make([]byte, len(data))
	mode.CryptBlocks(encrypted, data)

	return encrypted, nil
}

// decrypt implements AES CBC-128 DECRYPTION which requires 3 data fields
// 1. Key (16 bytes)
// 2. Initialization Vector (IV) (16 bytes)
// 3. Encrypted Data (16 bytes or length multiple a of 16)
// The encrypted data is divided into blocks of 16 bytes (128 bits) which then operated on with the IV and Key.
func decrypt(key []byte, iv []byte, encrypted []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	if len(encrypted)%blockSize != 0 {
		return nil, fmt.Errorf("encrypted payload length %d is not a multiple of AES block size %d", len(encrypted), blockSize)
	}

	if len(iv) != blockSize {
		return nil, fmt.Errorf("size of the IV %d is not the same as block size %d", len(iv), blockSize)
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	return decrypted, nil
}
