// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/postgres"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/stretchr/testify/require"
)

func TestDomainTransportKeyRepository(t *testing.T) {
	repo := postgres.NewDomainTransportKeyRepository(db)
	domainID := testsutil.GenerateUUID(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	current := bootstrap.DomainTransportKey{
		DomainID: domainID, KeyID: testsutil.GenerateUUID(t),
		EncryptedSecret: "dbv1.primary.encrypted", WrappingKeyID: "primary",
		Status: bootstrap.TransportKeyActive, CreatedAt: now, UpdatedAt: now,
	}

	require.NoError(t, repo.Create(context.Background(), current))
	stored, err := repo.RetrieveCurrent(context.Background(), domainID)
	require.NoError(t, err)
	require.Equal(t, current.KeyID, stored.KeyID)
	require.Equal(t, current.EncryptedSecret, stored.EncryptedSecret)

	duplicate := current
	duplicate.KeyID = testsutil.GenerateUUID(t)
	err = repo.Create(context.Background(), duplicate)
	require.True(t, errors.Contains(err, repoerr.ErrConflict))

	next := current
	next.KeyID = testsutil.GenerateUUID(t)
	next.EncryptedSecret = "dbv1.primary.next-encrypted"
	next.CreatedAt = now.Add(time.Second)
	next.UpdatedAt = next.CreatedAt
	retireAt := now.Add(24 * time.Hour)
	require.NoError(t, repo.Rotate(context.Background(), current.KeyID, next, retireAt))

	active, err := repo.RetrieveCurrent(context.Background(), domainID)
	require.NoError(t, err)
	require.Equal(t, next.KeyID, active.KeyID)
	retiring, err := repo.Retrieve(context.Background(), domainID, current.KeyID)
	require.NoError(t, err)
	require.Equal(t, bootstrap.TransportKeyRetiring, retiring.Status)
	require.NotNil(t, retiring.RetireAt)

	expiresAt := now.Add(5 * time.Minute)
	require.NoError(t, repo.ConsumeRequestID(context.Background(), domainID, next.KeyID, "request-id", expiresAt))
	err = repo.ConsumeRequestID(context.Background(), domainID, next.KeyID, "request-id", expiresAt)
	require.True(t, errors.Contains(err, repoerr.ErrConflict))
}
