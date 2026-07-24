// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap_test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/hkdf"
)

type readResp struct {
	ID          string                `json:"id"`
	ContentType bootstrap.ContentType `json:"content_type"`
	Content     string                `json:"content,omitempty"`
	ClientCert  string                `json:"client_cert,omitempty"`
	ClientKey   string                `json:"client_key,omitempty"`
	CACert      string                `json:"ca_cert,omitempty"`
}

func TestReadConfigEncryptsDeviceResponse(t *testing.T) {
	rootKey := []byte("12345678910111213141516171819202")
	cfg := bootstrap.Config{
		ID:                   "smq_id",
		ExternalID:           "external-id",
		ClientCert:           "client_cert",
		ClientKey:            "client_key",
		CACert:               "ca_cert",
		Content:              "content",
		ContentType:          bootstrap.ContentTypeTextPlain,
		BootstrapKeyVersion:  3,
		BootstrapRootKey:     rootKey,
		BootstrapChallengeID: "challenge-id",
		BootstrapServerNonce: base64.RawURLEncoding.EncodeToString(make([]byte, bootstrap.BootstrapNonceSize)),
		BootstrapDeviceNonce: base64.RawURLEncoding.EncodeToString([]byte("12345678910111213141516171819202")),
	}

	result, err := bootstrap.NewConfigReader().ReadConfig(cfg)
	require.NoError(t, err)
	envelope, ok := result.(bootstrap.EncryptedBootstrapConfig)
	require.True(t, ok)
	require.Equal(t, bootstrap.DeviceBootstrapVersion, envelope.Version)
	require.Equal(t, cfg.BootstrapKeyVersion, envelope.KeyVersion)
	require.Equal(t, cfg.BootstrapChallengeID, envelope.ChallengeID)

	key := make([]byte, bootstrap.BootstrapKeySize)
	_, err = io.ReadFull(
		hkdf.New(sha256.New, rootKey, []byte(cfg.ExternalID), []byte("magistrala-bootstrap-response-v1")),
		key,
	)
	require.NoError(t, err)
	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)
	nonce, err := base64.RawURLEncoding.DecodeString(envelope.Nonce)
	require.NoError(t, err)
	ciphertext, err := base64.RawURLEncoding.DecodeString(envelope.Ciphertext)
	require.NoError(t, err)
	aad := strings.Join([]string{
		"bootstrap-response-v1",
		cfg.ExternalID,
		cfg.BootstrapChallengeID,
		cfg.BootstrapServerNonce,
		cfg.BootstrapDeviceNonce,
		fmt.Sprintf("%d", cfg.BootstrapKeyVersion),
	}, "\n")
	plain, err := aead.Open(nil, nonce, ciphertext, []byte(aad))
	require.NoError(t, err)

	var got readResp
	require.NoError(t, json.Unmarshal(plain, &got))
	require.Equal(t, readResp{
		ID: "smq_id", ContentType: bootstrap.ContentTypeTextPlain,
		Content: "content", ClientCert: "client_cert",
		ClientKey: "client_key", CACert: "ca_cert",
	}, got)

	response, ok := result.(magistrala.Response)
	require.True(t, ok)
	require.Equal(t, "no-store", response.Headers()["Cache-Control"])
}
