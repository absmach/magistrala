// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/postgres"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"
)

func TestBootstrapChallengeRepositoryConsumeOnce(t *testing.T) {
	ctx := context.Background()
	configRepo := postgres.NewConfigRepository(db, testLog)
	challengeRepo := postgres.NewBootstrapChallengeRepository(db)

	configID := uuid.Must(uuid.NewV4()).String()
	_, err := configRepo.Save(ctx, bootstrap.Config{
		ID: configID, WorkspaceID: uuid.Must(uuid.NewV4()).String(),
		ExternalID: uuid.Must(uuid.NewV4()).String(), ExternalKey: "encrypted-key",
		BootstrapKeyVersion: 2, Status: bootstrap.Active,
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	challenge := bootstrap.BootstrapChallenge{
		ID: uuid.Must(uuid.NewV4()).String(), ConfigID: configID, KeyVersion: 2,
		ServerNonce: make([]byte, bootstrap.BootstrapNonceSize),
		CreatedAt:   now, ExpiresAt: now.Add(time.Minute),
	}
	require.NoError(t, challengeRepo.Create(ctx, challenge))

	stored, err := challengeRepo.Retrieve(ctx, challenge.ID, configID)
	require.NoError(t, err)
	require.Equal(t, challenge.ID, stored.ID)
	require.Equal(t, challenge.KeyVersion, stored.KeyVersion)
	require.Equal(t, challenge.ServerNonce, stored.ServerNonce)
	require.Nil(t, stored.ConsumedAt)

	require.NoError(t, challengeRepo.Consume(ctx, challenge.ID, configID, now.Add(time.Second)))
	err = challengeRepo.Consume(ctx, challenge.ID, configID, now.Add(2*time.Second))
	require.ErrorIs(t, err, repoerr.ErrConflict)
}

func TestBootstrapChallengeRepositoryRejectsExpiredConsume(t *testing.T) {
	ctx := context.Background()
	configRepo := postgres.NewConfigRepository(db, testLog)
	challengeRepo := postgres.NewBootstrapChallengeRepository(db)

	configID := uuid.Must(uuid.NewV4()).String()
	_, err := configRepo.Save(ctx, bootstrap.Config{
		ID: configID, WorkspaceID: uuid.Must(uuid.NewV4()).String(),
		ExternalID: uuid.Must(uuid.NewV4()).String(), ExternalKey: "encrypted-key",
		BootstrapKeyVersion: 1, Status: bootstrap.Active,
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	challenge := bootstrap.BootstrapChallenge{
		ID: uuid.Must(uuid.NewV4()).String(), ConfigID: configID, KeyVersion: 1,
		ServerNonce: make([]byte, bootstrap.BootstrapNonceSize),
		CreatedAt:   now, ExpiresAt: now.Add(time.Second),
	}
	require.NoError(t, challengeRepo.Create(ctx, challenge))
	err = challengeRepo.Consume(ctx, challenge.ID, configID, now.Add(time.Second))
	require.ErrorIs(t, err, repoerr.ErrConflict)
}
