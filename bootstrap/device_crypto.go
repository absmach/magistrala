// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/hkdf"
)

const (
	bootstrapAuthKeyInfo     = "magistrala-bootstrap-auth-v1"
	bootstrapResponseKeyInfo = "magistrala-bootstrap-response-v1"
)

func generateBootstrapKey() (string, error) {
	key := make([]byte, BootstrapKeySize)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return encodeBootstrapBytes(key), nil
}

func bootstrapKeyMaterial(value string) ([]byte, error) {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < BootstrapKeyMinLength {
		return nil, fmt.Errorf("bootstrap key must contain at least %d characters", BootstrapKeyMinLength)
	}
	return []byte(value), nil
}

func decodeBootstrapField(name, value string, size int) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != size || encodeBootstrapBytes(decoded) != value {
		return nil, fmt.Errorf("%s must be unpadded base64url containing exactly %d bytes", name, size)
	}
	return decoded, nil
}

func deriveDeviceKey(root []byte, externalID, purpose string) ([]byte, error) {
	key := make([]byte, BootstrapKeySize)
	reader := hkdf.New(sha256.New, root, []byte(externalID), []byte(purpose))
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func bootstrapProofInput(externalID, challengeID, serverNonce, deviceNonce string, keyVersion uint64) string {
	return strings.Join([]string{
		DeviceBootstrapVersion,
		externalID,
		challengeID,
		serverNonce,
		deviceNonce,
		fmt.Sprintf("%d", keyVersion),
	}, "\n")
}

func calculateBootstrapProof(root []byte, externalID, challengeID, serverNonce, deviceNonce string, keyVersion uint64) ([]byte, error) {
	key, err := deriveDeviceKey(root, externalID, bootstrapAuthKeyInfo)
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(bootstrapProofInput(externalID, challengeID, serverNonce, deviceNonce, keyVersion)))
	return mac.Sum(nil), nil
}

func bootstrapResponseAAD(externalID, challengeID, serverNonce, deviceNonce string, keyVersion uint64) string {
	return strings.Join([]string{
		"bootstrap-response-v1",
		externalID,
		challengeID,
		serverNonce,
		deviceNonce,
		fmt.Sprintf("%d", keyVersion),
	}, "\n")
}

func newDeviceBootstrapAEAD(root []byte, externalID string) (cipher.AEAD, error) {
	key, err := deriveDeviceKey(root, externalID, bootstrapResponseKeyInfo)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func encryptDeviceBootstrapResponse(cfg Config, plain []byte) (EncryptedBootstrapConfig, error) {
	aead, err := newDeviceBootstrapAEAD(cfg.BootstrapRootKey, cfg.ExternalID)
	if err != nil {
		return EncryptedBootstrapConfig{}, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return EncryptedBootstrapConfig{}, err
	}
	aad := bootstrapResponseAAD(
		cfg.ExternalID,
		cfg.BootstrapChallengeID,
		cfg.BootstrapServerNonce,
		cfg.BootstrapDeviceNonce,
		cfg.BootstrapKeyVersion,
	)
	ciphertext := aead.Seal(nil, nonce, plain, []byte(aad))
	return EncryptedBootstrapConfig{
		Version: DeviceBootstrapVersion, KeyVersion: cfg.BootstrapKeyVersion,
		ChallengeID: cfg.BootstrapChallengeID,
		Nonce:       encodeBootstrapBytes(nonce), Ciphertext: encodeBootstrapBytes(ciphertext),
	}, nil
}
