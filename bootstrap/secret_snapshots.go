// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
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
		if bs.dbCipher == nil {
			return nil, ErrSecretEncryption
		}
		ciphertext, err := bs.dbCipher.seal("binding-snapshot", secret, snapshotSecretAAD(binding.ConfigID, binding.Slot))
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
		if bs.dbCipher == nil {
			return nil, ErrSecretEncryption
		}
		plain, err := bs.dbCipher.open("binding-snapshot", ciphertext, snapshotSecretAAD(binding.ConfigID, binding.Slot))
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
