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

func decrypt(keyHex string, iv string, encryptedHex string) string {
	/*
	AES CBC-128 DECRYPTION requires 3 data fields
	1. Key (16 bytes)
	2. Initialization Vector (16 bytes). {Lua script generates this}
	3. Encrypted Data (16 bytes or length be multiple a of 16) {Not the whole Telegram rather the encrypted part}

	The encrypted data is divided into blocks of 16 bytes (128 bits) which then operated on with the IV and Key.   
	*/
	
	// Convert key hex string to bytes
	key,err := hex.DecodeString(strings.ReplaceAll(keyHex, " ", ""))
	if err != nil {
		log.Fatalf("Key Hex decode error: %v", err)
	}

	// Convert encrypted hex string to bytes
	encrypted,err := hex.DecodeString(strings.ReplaceAll(encryptedHex, " ", ""))
	if err != nil {
		log.Fatalf("Encrypted Hex decode error: %v", err)
	}

		iv_bytes := []byte{}
		// Convert hex string to bytes
		newBytes, err := hex.DecodeString(iv)

		if err != nil {
			log.Fatalf("Failed to decode hex string: %v", err)
		}
		
		// The IV bytes
		iv_bytes = append(iv_bytes, newBytes...)
		
		// Create the Cipher from Key

		block, err := aes.NewCipher(key)
		if err != nil {
			log.Fatalf("NewCipher error: %v", err)
		}

		// The encrypted block should be 16 bytes or a multiple of 16
		if len(encrypted)%aes.BlockSize != 0 {
			log.Fatalf("Encrypted data is not a multiple of the block size")
		}

		// Decryption done with the key, IV and encrypted hex
		mode := cipher.NewCBCDecrypter(block, iv_bytes)
		decrypted := make([]byte, len(encrypted))
		mode.CryptBlocks(decrypted, encrypted)

	return strings.ToUpper(hex.EncodeToString(decrypted))
}