// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
)


// AES CBC-128 DECRYPTION requires 3 data fields
// 1. Key (16 bytes)
// 2. Initialization Vector (16 bytes). {Lua script generates this}
// 3. Encrypted Data (16 bytes or length be multiple a of 16) {Not the whole Telegram rather the encrypted part}
// The encrypted data is divided into blocks of 16 bytes (128 bits) which then operated on with the IV and Key.   


// decrypt takes key, iv, and encrypted data as hex strings and returns the decrypted result as a hex string
func decrypt(key []byte, iv []byte, encrypted []byte) []byte {
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Fatalf("NewCipher error: %v", err)
	}

	if len(encrypted)%aes.BlockSize != 0 {
		log.Fatalf("Encrypted data is not a multiple of the block size")
	}

	// Decrypt using CBC mode
	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(encrypted))
	mode.CryptBlocks(decrypted, encrypted)

	return decrypted

}