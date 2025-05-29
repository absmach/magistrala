// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncrypt(t *testing.T) {
	validKey := make([]byte, aes.BlockSize)
	validIV := make([]byte, aes.BlockSize)
	validData := make([]byte, aes.BlockSize*2) // 2 blocks

	_, err := rand.Read(validKey)
	require.Nil(t, err, "Failed to generate valid key")
	_, err = rand.Read(validIV)
	require.Nil(t, err, "Failed to generate valid IV")
	_, err = rand.Read(validData)
	require.Nil(t, err, "Failed to generate valid data")

	cases := []struct {
		name string
		key  []byte
		iv   []byte
		data []byte
		err  error
	}{
		{
			name: "valid encryption - single block",
			key:  validKey,
			iv:   validIV,
			data: make([]byte, 16),
			err:  nil,
		},
		{
			name: "valid encryption - multiple blocks",
			key:  validKey,
			iv:   validIV,
			data: validData,
			err:  nil,
		},
		{
			name: "valid encryption - empty data",
			key:  validKey,
			iv:   validIV,
			data: make([]byte, 0),
			err:  nil,
		},
		{
			name: "invalid IV size - too short",
			key:  validKey,
			iv:   make([]byte, 8),
			data: validData,
			err:  errors.New("size of the IV 8 is not the same as block size 16"),
		},
		{
			name: "invalid IV size - too long",
			key:  validKey,
			iv:   make([]byte, 32),
			data: validData,
			err:  errors.New("size of the IV 32 is not the same as block size 16"),
		},
		{
			name: "invalid IV size - nil",
			key:  validKey,
			iv:   nil,
			data: validData,
			err:  errors.New("size of the IV 0 is not the same as block size 16"),
		},
		{
			name: "invalid data size - not multiple of block size",
			key:  validKey,
			iv:   validIV,
			data: make([]byte, 15),
			err:  errors.New("payload length 15 is not a multiple of AES block size 16"),
		},
		{
			name: "invalid data size - odd length",
			key:  validKey,
			iv:   validIV,
			data: make([]byte, 17),
			err:  errors.New("payload length 17 is not a multiple of AES block size 16"),
		},
		{
			name: "invalid key size - too short",
			key:  make([]byte, 8),
			iv:   validIV,
			data: validData,
			err:  aes.KeySizeError(8),
		},
		{
			name: "invalid key size - nil",
			key:  nil,
			iv:   validIV,
			data: validData,
			err:  aes.KeySizeError(0),
		},
		{
			name: "AES-192 key",
			key:  make([]byte, 24),
			iv:   validIV,
			data: validData,
			err:  nil,
		},
		{
			name: "AES-256 key",
			key:  make([]byte, 32),
			iv:   validIV,
			data: validData,
			err:  nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := encrypt(tt.key, tt.iv, tt.data)
			if tt.err != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.err, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, len(tt.data), len(result))

				// Ensure encrypted data is different from original (unless data is all zeros)
				if len(tt.data) > 0 && !bytes.Equal(tt.data, make([]byte, len(tt.data))) {
					assert.NotEqual(t, tt.data, result)
				}
			}
		})
	}
}

func TestDecrypt(t *testing.T) {
	validKey := make([]byte, aes.BlockSize)
	validIV := make([]byte, aes.BlockSize)
	validData := make([]byte, aes.BlockSize*2) // 2 blocks

	_, err := rand.Read(validKey)
	require.Nil(t, err, "Failed to generate valid key")
	_, err = rand.Read(validIV)
	require.Nil(t, err, "Failed to generate valid IV")
	_, err = rand.Read(validData)
	require.Nil(t, err, "Failed to generate valid data")

	validEncrypted, _ := encrypt(validKey, validIV, validData)

	cases := []struct {
		name      string
		key       []byte
		iv        []byte
		encrypted []byte
		err       error
	}{
		{
			name:      "valid decryption - single block",
			key:       validKey,
			iv:        validIV,
			encrypted: validEncrypted[:16],
			err:       nil,
		},
		{
			name:      "valid decryption - multiple blocks",
			key:       validKey,
			iv:        validIV,
			encrypted: validEncrypted,
			err:       nil,
		},
		{
			name:      "valid decryption - empty data",
			key:       validKey,
			iv:        validIV,
			encrypted: make([]byte, 0),
			err:       nil,
		},
		{
			name:      "invalid IV size - too short",
			key:       validKey,
			iv:        make([]byte, 8),
			encrypted: validEncrypted,
			err:       errors.New("size of the IV 8 is not the same as block size 16"),
		},
		{
			name:      "invalid IV size - too long",
			key:       validKey,
			iv:        make([]byte, 32),
			encrypted: validEncrypted,
			err:       errors.New("size of the IV 32 is not the same as block size 16"),
		},
		{
			name:      "invalid IV size - nil",
			key:       validKey,
			iv:        nil,
			encrypted: validEncrypted,
			err:       errors.New("size of the IV 0 is not the same as block size 16"),
		},
		{
			name:      "invalid encrypted data size - not multiple of block size",
			key:       validKey,
			iv:        validIV,
			encrypted: make([]byte, 15),
			err:       errors.New("encrypted payload length 15 is not a multiple of AES block size 16"),
		},
		{
			name:      "invalid encrypted data size - odd length",
			key:       validKey,
			iv:        validIV,
			encrypted: make([]byte, 17),
			err:       errors.New("encrypted payload length 17 is not a multiple of AES block size 16"),
		},
		{
			name:      "invalid key size - too short",
			key:       make([]byte, 8),
			iv:        validIV,
			encrypted: validEncrypted,
			err:       aes.KeySizeError(8),
		},
		{
			name:      "invalid key size - nil",
			key:       nil,
			iv:        validIV,
			encrypted: validEncrypted,
			err:       aes.KeySizeError(0),
		},
		{
			name:      "AES-192 key",
			key:       make([]byte, 24),
			iv:        validIV,
			encrypted: validEncrypted,
			err:       nil,
		},
		{
			name:      "AES-256 key",
			key:       make([]byte, 32),
			iv:        validIV,
			encrypted: validEncrypted,
			err:       nil,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			result, err := decrypt(tt.key, tt.iv, tt.encrypted)
			if tt.err != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.err, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, len(tt.encrypted), len(result))
			}
		})
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		keySize  int
		dataSize int
	}{
		{
			name:     "AES-128 single block",
			keySize:  16,
			dataSize: 16,
		},
		{
			name:     "AES-128 multiple blocks",
			keySize:  16,
			dataSize: 64,
		},
		{
			name:     "AES-192 single block",
			keySize:  24,
			dataSize: 16,
		},
		{
			name:     "AES-192 multiple blocks",
			keySize:  24,
			dataSize: 48,
		},
		{
			name:     "AES-256 single block",
			keySize:  32,
			dataSize: 16,
		},
		{
			name:     "AES-256 multiple blocks",
			keySize:  32,
			dataSize: 80,
		},
		{
			name:     "empty data",
			keySize:  16,
			dataSize: 0,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			iv := make([]byte, aes.BlockSize)
			originalData := make([]byte, tt.dataSize)

			_, err := rand.Read(key)
			require.Nil(t, err, "Failed to generate valid key")
			_, err = rand.Read(iv)
			require.Nil(t, err, "Failed to generate valid IV")

			if tt.dataSize > 0 {
				_, err = rand.Read(originalData)
				require.Nil(t, err, "Failed to generate valid data")
			}

			encrypted, err := encrypt(key, iv, originalData)
			assert.NoError(t, err)
			assert.NotNil(t, encrypted)

			decrypted, err := decrypt(key, iv, encrypted)
			assert.NoError(t, err)
			assert.NotNil(t, decrypted)

			assert.Equal(t, originalData, decrypted)
		})
	}
}

func TestEncryptDecryptWithSample(t *testing.T) {
	iv := "0907780613000704d2d2d2d2d2d2d2d2"
	ivBytes, err := hex.DecodeString(iv)
	assert.NoError(t, err, "Failed to decode IV")
	payload := "Ba56dc989e08a76f855ae12ae8B00ef13fae6ad436eBe8e03e97f17B5751c241"
	payloadBytes, err := hex.DecodeString(payload)
	assert.NoError(t, err, "Failed to decode payload")
	key := "CB6ABFAA8D2247B59127D3B839CF34B4"
	keyBytes, err := hex.DecodeString(key)
	assert.NoError(t, err, "Failed to decode key")
	expected := "2f2f0c0613760100046d27350f380c13555134022f2f2f2f2f2f2f2f2f2f2f2f"

	decrypted, err := decrypt(keyBytes, ivBytes, payloadBytes)
	assert.NoError(t, err, "Failed to decrypt")
	assert.NotNil(t, decrypted, "Decrypted payload is nil")
	assert.Equal(t, expected, hex.EncodeToString(decrypted), "Decrypted payload does not match expected")
}
