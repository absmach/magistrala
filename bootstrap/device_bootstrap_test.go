// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"sync"
	"testing"
	"time"

	smqauthn "github.com/absmach/magistrala/pkg/authn"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/stretchr/testify/require"
)

type deviceTestConfigRepository struct {
	config Config
}

func (r *deviceTestConfigRepository) Save(_ context.Context, cfg Config) (string, error) {
	r.config = cfg
	return cfg.ID, nil
}

func (r *deviceTestConfigRepository) RetrieveByID(_ context.Context, workspaceID, id string) (Config, error) {
	if r.config.WorkspaceID != workspaceID || r.config.ID != id {
		return Config{}, repoerr.ErrNotFound
	}
	return r.config, nil
}

func (r *deviceTestConfigRepository) RetrieveAll(context.Context, string, Filter, uint64, uint64) (ConfigsPage, error) {
	return ConfigsPage{}, nil
}

func (r *deviceTestConfigRepository) RetrieveByExternalID(_ context.Context, externalID string) (Config, error) {
	if r.config.ExternalID != externalID {
		return Config{}, repoerr.ErrNotFound
	}
	return r.config, nil
}

func (r *deviceTestConfigRepository) Update(_ context.Context, cfg Config) error {
	r.config = cfg
	return nil
}

func (*deviceTestConfigRepository) AssignProfile(context.Context, string, string, string) error {
	return nil
}

func (*deviceTestConfigRepository) UpdateCert(context.Context, string, string, string, string, string) (Config, error) {
	return Config{}, nil
}

func (*deviceTestConfigRepository) Remove(context.Context, string, string) error {
	return nil
}

func (*deviceTestConfigRepository) ChangeStatus(context.Context, string, string, Status) error {
	return nil
}

type deviceTestChallengeRepository struct {
	mu         sync.Mutex
	challenges map[string]BootstrapChallenge
}

func (r *deviceTestChallengeRepository) Create(_ context.Context, challenge BootstrapChallenge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.challenges[challenge.ID] = challenge
	return nil
}

func (r *deviceTestChallengeRepository) Retrieve(_ context.Context, challengeID, configID string) (BootstrapChallenge, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	challenge, ok := r.challenges[challengeID]
	if !ok || challenge.ConfigID != configID {
		return BootstrapChallenge{}, repoerr.ErrNotFound
	}
	return challenge, nil
}

func (r *deviceTestChallengeRepository) Consume(_ context.Context, challengeID, configID string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	challenge, ok := r.challenges[challengeID]
	if !ok || challenge.ConfigID != configID || challenge.ConsumedAt != nil || !now.Before(challenge.ExpiresAt) {
		return repoerr.ErrConflict
	}
	challenge.ConsumedAt = &now
	r.challenges[challengeID] = challenge
	return nil
}

type deviceTestIDProvider struct {
	id string
}

func (p deviceTestIDProvider) ID() (string, error) {
	return p.id, nil
}

func TestDeviceBootstrapChallengeProofAndReplayProtection(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	externalKey := "key-20260721093505-20"
	root := []byte(externalKey)
	cfg := Config{
		ID: "config-id", WorkspaceID: "domain-id", ExternalID: "device-1",
		ExternalKey: externalKey, BootstrapKeyVersion: 4, Content: `{"channels":["channel-1"]}`, Status: Active,
	}
	cipher, err := NewSecretCipher([]byte("12345678910111213141516171819202"), "primary")
	require.NoError(t, err)
	cfg.ExternalKey, err = cipher.seal("config-external-key", []byte(externalKey), configSecretAAD(cfg))
	require.NoError(t, err)

	configs := &deviceTestConfigRepository{config: cfg}
	challenges := &deviceTestChallengeRepository{challenges: make(map[string]BootstrapChallenge)}
	svcIface, err := New(
		configs, nil, nil, challenges, nil, nil, nil,
		[]byte("12345678910111213141516171819202"), "primary",
		deviceTestIDProvider{id: "challenge-id"},
	)
	require.NoError(t, err)
	svc := svcIface.(*bootstrapService)
	svc.now = func() time.Time { return now }

	challenge, err := svc.IssueBootstrapChallenge(ctx, cfg.ExternalID)
	require.NoError(t, err)
	require.Equal(t, uint64(4), challenge.KeyVersion)
	require.Equal(t, "challenge-id", challenge.ChallengeID)

	deviceNonce := base64.RawURLEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwx12345678"))
	calculated, err := calculateBootstrapProof(
		root, cfg.ExternalID, challenge.ChallengeID, challenge.ServerNonce, deviceNonce, challenge.KeyVersion,
	)
	require.NoError(t, err)
	proof := DeviceBootstrapProof{
		ChallengeID: challenge.ChallengeID,
		DeviceNonce: deviceNonce,
		Proof:       base64.RawURLEncoding.EncodeToString(calculated),
	}

	got, err := svc.Bootstrap(ctx, cfg.ExternalID, proof)
	require.NoError(t, err)
	require.Equal(t, cfg.Content, got.Content)
	require.Empty(t, got.ExternalKey)
	require.True(t, hmac.Equal(root, got.BootstrapRootKey))
	require.Equal(t, challenge.ChallengeID, got.BootstrapChallengeID)

	_, err = svc.Bootstrap(ctx, cfg.ExternalID, proof)
	require.ErrorIs(t, err, ErrDeviceBootstrapAuth)
}

func TestDeviceBootstrapRejectsWrongProofWithoutConsumingChallenge(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	externalKey := "key-20260721093505-20"
	root := []byte(externalKey)
	cfg := Config{
		ID: "config-id", WorkspaceID: "domain-id", ExternalID: "device-1",
		ExternalKey: externalKey, BootstrapKeyVersion: 1, Status: Active,
	}
	cipher, err := NewSecretCipher([]byte("12345678910111213141516171819202"), "primary")
	require.NoError(t, err)
	cfg.ExternalKey, err = cipher.seal("config-external-key", []byte(externalKey), configSecretAAD(cfg))
	require.NoError(t, err)
	challenges := &deviceTestChallengeRepository{challenges: make(map[string]BootstrapChallenge)}
	svcIface, err := New(
		&deviceTestConfigRepository{config: cfg}, nil, nil, challenges, nil, nil, nil,
		[]byte("12345678910111213141516171819202"), "primary",
		deviceTestIDProvider{id: "challenge-id"},
	)
	require.NoError(t, err)
	svc := svcIface.(*bootstrapService)
	svc.now = func() time.Time { return now }

	challenge, err := svc.IssueBootstrapChallenge(ctx, cfg.ExternalID)
	require.NoError(t, err)
	deviceNonce := base64.RawURLEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwx12345678"))
	wrong := DeviceBootstrapProof{
		ChallengeID: challenge.ChallengeID,
		DeviceNonce: deviceNonce,
		Proof:       base64.RawURLEncoding.EncodeToString(make([]byte, BootstrapProofSize)),
	}
	_, err = svc.Bootstrap(ctx, cfg.ExternalID, wrong)
	require.ErrorIs(t, err, ErrDeviceBootstrapAuth)

	calculated, err := calculateBootstrapProof(
		root, cfg.ExternalID, challenge.ChallengeID, challenge.ServerNonce, deviceNonce, challenge.KeyVersion,
	)
	require.NoError(t, err)
	wrong.Proof = base64.RawURLEncoding.EncodeToString(calculated)
	_, err = svc.Bootstrap(ctx, cfg.ExternalID, wrong)
	require.NoError(t, err)
}

func TestAddGeneratesRecoverableDeviceKey(t *testing.T) {
	configs := &deviceTestConfigRepository{}
	svc, err := New(
		configs, nil, nil, nil, nil, nil, nil,
		[]byte("12345678910111213141516171819202"), "primary",
		deviceTestIDProvider{id: "config-id"},
	)
	require.NoError(t, err)
	session := smqauthn.Session{WorkspaceID: "domain-id"}
	created, err := svc.Add(context.Background(), session, "", Config{ExternalID: "device-1"})
	require.NoError(t, err)
	require.Equal(t, uint64(1), created.BootstrapKeyVersion)
	require.Len(t, created.ExternalKey, 43)
	keyMaterial, err := bootstrapKeyMaterial(created.ExternalKey)
	require.NoError(t, err)
	require.Equal(t, []byte(created.ExternalKey), keyMaterial)
	require.NotEqual(t, created.ExternalKey, configs.config.ExternalKey)

	viewed, err := svc.View(context.Background(), session, created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ExternalKey, viewed.ExternalKey)
}

func TestAddRejectsShortBootstrapKey(t *testing.T) {
	svc, err := New(
		&deviceTestConfigRepository{}, nil, nil, nil, nil, nil, nil,
		[]byte("12345678910111213141516171819202"), "primary",
		deviceTestIDProvider{id: "config-id"},
	)
	require.NoError(t, err)

	_, err = svc.Add(
		context.Background(),
		smqauthn.Session{WorkspaceID: "domain-id"},
		"",
		Config{ExternalID: "device-1", ExternalKey: "123456789"},
	)

	require.Error(t, err)
	require.ErrorContains(t, err, "at least 10 characters")
}
