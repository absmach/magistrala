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

func dateConv(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	length := len(data)
	hasSeconds := length > 5

	var seconds *byte
	var coreData []byte

	if hasSeconds {
		s := data[length-1] & 0x3F
		seconds = &s
		coreData = data[1 : length-1]
	} else {
		coreData = data
	}

	var value uint32 = 0
	for i, b := range coreData {
		value |= uint32(b) << (8 * uint(i))
	}

	if value == 0 {
		return []byte{}, nil
	}

	rawDate := value >> 16
	year := ((rawDate >> 5) & 0x7) | ((rawDate >> 9) & 0x78)
	month := (rawDate >> 8) & 0xF
	day := rawDate & 0x1F
	fullYear := 1900 + int(year)
	if year < 100 {
		fullYear += 100
	}

	hour := (value >> 8) & 0x1F
	minute := value & 0x3F

	// Build optional components
	extras := ""
	if seconds != nil {
		extras += fmt.Sprintf(":%02d", *seconds)
	}
	if value&0x8000 != 0 {
		extras += " (summer)"
	}
	if value&0x80 != 0 {
		extras += " (invalid)"
	}

	formatted := fmt.Sprintf("%04d-%02d-%02d %02d:%02d%s", fullYear, month, day, hour, minute, extras)
	return []byte(formatted), nil
}