// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	oldMasterKey = []byte("12345678910111213141516171819202")
	newMasterKey = []byte("abcdefghijklmnopqrstuvwxyz012345")
)

func TestSecretCipherRotation(t *testing.T) {
	const aad = "bootstrap-config:ws:cfg:ext"

	old, err := NewSecretCipher(oldMasterKey, "old")
	require.NoError(t, err)
	sealed, err := old.seal("config-external-key", []byte("device-root-key"), aad)
	require.NoError(t, err)

	t.Run("rotating without retaining the old key cannot open existing secrets", func(t *testing.T) {
		rotated, err := NewSecretCipher(newMasterKey, "new")
		require.NoError(t, err)

		_, err = rotated.open("config-external-key", sealed, aad)
		assert.Error(t, err, "a secret sealed under a retired key must not silently decrypt")
	})

	t.Run("retaining the old key keeps existing secrets readable", func(t *testing.T) {
		rotated, err := NewSecretCipher(newMasterKey, "new", PreviousKey{ID: "old", Key: oldMasterKey})
		require.NoError(t, err)

		plain, err := rotated.open("config-external-key", sealed, aad)
		require.NoError(t, err, "rotation must not brick secrets sealed under the previous key")
		assert.Equal(t, "device-root-key", string(plain))
	})

	t.Run("new writes use the active key", func(t *testing.T) {
		rotated, err := NewSecretCipher(newMasterKey, "new", PreviousKey{ID: "old", Key: oldMasterKey})
		require.NoError(t, err)

		envelope, err := rotated.seal("config-external-key", []byte("fresh"), aad)
		require.NoError(t, err)
		assert.Contains(t, envelope, "dbv1.new.", "seal must use the active key ID")

		// The retired key alone must not be able to open it.
		onlyOld, err := NewSecretCipher(oldMasterKey, "old")
		require.NoError(t, err)
		_, err = onlyOld.open("config-external-key", envelope, aad)
		assert.Error(t, err)
	})

	t.Run("associated data is still enforced across keys", func(t *testing.T) {
		rotated, err := NewSecretCipher(newMasterKey, "new", PreviousKey{ID: "old", Key: oldMasterKey})
		require.NoError(t, err)

		_, err = rotated.open("config-external-key", sealed, "bootstrap-config:other:cfg:ext")
		assert.Error(t, err, "a retired key must not weaken AAD binding")
	})

	t.Run("malformed previous keys are rejected at construction", func(t *testing.T) {
		_, err := NewSecretCipher(newMasterKey, "new", PreviousKey{ID: "old", Key: []byte("short")})
		assert.Error(t, err)

		_, err = NewSecretCipher(newMasterKey, "new", PreviousKey{ID: "", Key: oldMasterKey})
		assert.Error(t, err)

		_, err = NewSecretCipher(newMasterKey, "new", PreviousKey{ID: "new", Key: oldMasterKey})
		assert.Error(t, err, "a previous key must not shadow the active key ID")
	})
}
