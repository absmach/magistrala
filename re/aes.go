// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
)

// Converts a full raw hex frame to decrypted output
func decryptAxiomaFrame(rawHex string, keyHex string) (string, error) {
	// Convert inputs to bytes
	raw, err := hex.DecodeString(rawHex)
	if err != nil {
		return "", fmt.Errorf("invalid raw hex: %v", err)
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return "", fmt.Errorf("invalid key hex: %v", err)
	}
	if len(raw) < 16 {
		return "", errors.New("raw frame too short")
	}
	if len(key) != 16 {
		return "", errors.New("AES-128 requires 16-byte key")
	}

	// Extract Meter ID (bytes 4–7, index 4–7)
	mid := raw[4:8]

	// Extract Access Number (byte 12, index 11 in 0-based)
	accessNumber := raw[11]

	// Construct IV: 4 zeroes + meter ID + access number + 3 zeroes
	iv := make([]byte, 16)
	copy(iv[4:], mid)
	iv[8] = accessNumber
	// iv[9:] are already zero

	// Encrypted part starts from byte 16 (index 15)
	encrypted := raw[15:]
	if len(encrypted)%aes.BlockSize != 0 {
		return "", errors.New("encrypted data must be multiple of 16 bytes")
	}

	// Decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	return hex.EncodeToString(decrypted), nil
}

