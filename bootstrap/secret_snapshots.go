// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
)

const secretSnapshotCiphertextKey = "ciphertext"

func (bs bootstrapService) encryptSecretSnapshots(bindings []BindingSnapshot) ([]BindingSnapshot, error) {
	encrypted := make([]BindingSnapshot, len(bindings))
	for i, binding := range bindings {
		encrypted[i] = binding
		if len(binding.SecretSnapshot) == 0 {
			continue
		}
		secret, err := json.Marshal(binding.SecretSnapshot)
		if err != nil {
			return nil, err
		}
		ciphertext, err := bs.encrypt(secret)
		if err != nil {
			return nil, err
		}
		encrypted[i].SecretSnapshot = map[string]any{
			secretSnapshotCiphertextKey: ciphertext,
		}
	}
	return encrypted, nil
}

func (bs bootstrapService) decryptSecretSnapshots(bindings []BindingSnapshot) ([]BindingSnapshot, error) {
	decrypted := make([]BindingSnapshot, len(bindings))
	for i, binding := range bindings {
		decrypted[i] = binding
		ciphertext, ok := binding.SecretSnapshot[secretSnapshotCiphertextKey].(string)
		if !ok {
			continue
		}
		plain, err := bs.decrypt(ciphertext)
		if err != nil {
			return nil, err
		}
		var secret map[string]any
		if err := json.Unmarshal(plain, &secret); err != nil {
			return nil, err
		}
		decrypted[i].SecretSnapshot = secret
	}
	return decrypted, nil
}

func hideSecretSnapshots(bindings []BindingSnapshot) []BindingSnapshot {
	hidden := make([]BindingSnapshot, len(bindings))
	for i, binding := range bindings {
		hidden[i] = binding
		hidden[i].SecretSnapshot = nil
	}
	return hidden
}

func (bs bootstrapService) encrypt(plain []byte) (string, error) {
	block, err := aes.NewCipher(bs.encKey)
	if err != nil {
		return "", err
	}
	ciphertext := make([]byte, aes.BlockSize+len(plain))
	iv := ciphertext[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plain)
	return hex.EncodeToString(ciphertext), nil
}

func (bs bootstrapService) decrypt(in string) ([]byte, error) {
	ciphertext, err := hex.DecodeString(in)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(bs.encKey)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, ErrExternalKeySecure
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}
