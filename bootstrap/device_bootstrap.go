// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"encoding/base64"
	"time"
)

const (
	DeviceBootstrapVersion = "v1"
	BootstrapKeySize       = 32
	BootstrapKeyMinLength  = 10
	BootstrapNonceSize     = 32
	BootstrapProofSize     = 32
	DefaultChallengeTTL    = time.Minute
)

// BootstrapChallenge is a short-lived server challenge bound to one
// enrollment and one version of its device Bootstrap key.
type BootstrapChallenge struct {
	ID          string
	ConfigID    string
	KeyVersion  uint64
	ServerNonce []byte
	CreatedAt   time.Time
	ExpiresAt   time.Time
	ConsumedAt  *time.Time
}

// BootstrapChallengeRepository persists short-lived device challenges.
type BootstrapChallengeRepository interface {
	Create(ctx context.Context, challenge BootstrapChallenge) error
	Retrieve(ctx context.Context, challengeID, configID string) (BootstrapChallenge, error)
	Consume(ctx context.Context, challengeID, configID string, now time.Time) error
}

// BootstrapChallengeResponse is returned to a device before it proves
// possession of its enrollment-specific Bootstrap key.
type BootstrapChallengeResponse struct {
	ChallengeID string    `json:"challenge_id"`
	ServerNonce string    `json:"server_nonce"`
	ExpiresAt   time.Time `json:"expires_at"`
	KeyVersion  uint64    `json:"key_version"`
}

// DeviceBootstrapProof authenticates a device response to a server challenge.
type DeviceBootstrapProof struct {
	ChallengeID string `json:"challenge_id"`
	DeviceNonce string `json:"device_nonce"`
	Proof       string `json:"proof"`
}

// EncryptedBootstrapConfig is the application-encrypted device configuration.
type EncryptedBootstrapConfig struct {
	Version     string `json:"version"`
	KeyVersion  uint64 `json:"key_version"`
	ChallengeID string `json:"challenge_id"`
	Nonce       string `json:"nonce"`
	Ciphertext  string `json:"ciphertext"`
}

func encodeBootstrapBytes(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}
