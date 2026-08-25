// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeviceBootstrapEndpoints(t *testing.T) {
	server, svc, _ := newBootstrapServer()
	defer server.Close()

	challenge := bootstrap.BootstrapChallengeResponse{
		ChallengeID: "challenge-id",
		ServerNonce: base64.RawURLEncoding.EncodeToString(make([]byte, bootstrap.BootstrapNonceSize)),
		ExpiresAt:   time.Now().UTC().Add(time.Minute),
		KeyVersion:  2,
	}
	challengeCall := svc.On("IssueBootstrapChallenge", mock.Anything, "device-1").Return(challenge, nil).Once()

	res, err := server.Client().Post(
		server.URL+"/devices/bootstrap/challenges/device-1",
		"application/json",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "no-store", res.Header.Get("Cache-Control"))
	var gotChallenge bootstrap.BootstrapChallengeResponse
	require.NoError(t, json.NewDecoder(res.Body).Decode(&gotChallenge))
	require.NoError(t, res.Body.Close())
	require.Equal(t, challenge, gotChallenge)
	challengeCall.Unset()

	proof := bootstrap.DeviceBootstrapProof{
		ChallengeID: challenge.ChallengeID,
		DeviceNonce: base64.RawURLEncoding.EncodeToString([]byte("12345678910111213141516171819202")),
		Proof:       base64.RawURLEncoding.EncodeToString(make([]byte, bootstrap.BootstrapProofSize)),
	}
	body, err := json.Marshal(proof)
	require.NoError(t, err)
	cfg := bootstrap.Config{
		ID:                   "config-id",
		ExternalID:           "device-1",
		Content:              `{"channels":["channel-1"]}`,
		BootstrapKeyVersion:  challenge.KeyVersion,
		BootstrapRootKey:     []byte("12345678910111213141516171819202"),
		BootstrapChallengeID: challenge.ChallengeID,
		BootstrapServerNonce: challenge.ServerNonce,
		BootstrapDeviceNonce: proof.DeviceNonce,
	}
	bootstrapCall := svc.On("Bootstrap", mock.Anything, "device-1", proof).Return(cfg, nil).Once()

	res, err = server.Client().Post(
		server.URL+"/devices/bootstrap/configurations/device-1",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "no-store", res.Header.Get("Cache-Control"))
	var envelope bootstrap.EncryptedBootstrapConfig
	require.NoError(t, json.NewDecoder(res.Body).Decode(&envelope))
	require.NoError(t, res.Body.Close())
	require.Equal(t, bootstrap.DeviceBootstrapVersion, envelope.Version)
	require.Equal(t, challenge.KeyVersion, envelope.KeyVersion)
	require.Equal(t, challenge.ChallengeID, envelope.ChallengeID)
	require.NotEmpty(t, envelope.Nonce)
	require.NotEmpty(t, envelope.Ciphertext)
	bootstrapCall.Unset()

	invalidProofCall := svc.On("Bootstrap", mock.Anything, "device-1", proof).
		Return(bootstrap.Config{}, bootstrap.ErrDeviceBootstrapAuth).
		Once()
	res, err = server.Client().Post(
		server.URL+"/devices/bootstrap/configurations/device-1",
		"application/json",
		bytes.NewReader(body),
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, res.StatusCode)
	require.NoError(t, res.Body.Close())
	invalidProofCall.Unset()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/devices/bootstrap/device-1", nil)
	require.NoError(t, err)
	res, err = server.Client().Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, res.StatusCode)
	require.NoError(t, res.Body.Close())
}
