// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/auth/postgres"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	expTime    = time.Now().Add(5 * time.Minute)
	idProvider = uuid.New()
	invalidID  = strings.Repeat("a", 255)
)

func generateID(t *testing.T) string {
	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	return id
}

func TestKeySave(t *testing.T) {
	repo := postgres.New(database)

	keyID := generateID(t)
	issuer := generateID(t)

	cases := []struct {
		desc string
		key  auth.Key
		err  error
	}{
		{
			desc: "save a new key",
			key: auth.Key{
				ID:        keyID,
				Type:      auth.APIKey,
				Issuer:    issuer,
				Subject:   generateID(t),
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
			},
			err: nil,
		},
		{
			desc: "save with duplicate id",
			key: auth.Key{
				ID:        keyID,
				Type:      auth.APIKey,
				Issuer:    issuer,
				Subject:   generateID(t),
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "save with empty id",
			key: auth.Key{
				Type:      auth.APIKey,
				Issuer:    issuer,
				Subject:   generateID(t),
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
			},
			err: nil,
		},
		{
			desc: "save with empty subject",
			key: auth.Key{
				ID:        generateID(t),
				Type:      auth.APIKey,
				Issuer:    issuer,
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
			},
			err: nil,
		},
		{
			desc: "save with empty issuer",
			key: auth.Key{
				ID:        generateID(t),
				Type:      auth.APIKey,
				Issuer:    "",
				Subject:   generateID(t),
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
			},
			err: nil,
		},
		{
			desc: "save with empty issued at",
			key: auth.Key{
				ID:        generateID(t),
				Type:      auth.APIKey,
				Issuer:    issuer,
				Subject:   generateID(t),
				IssuedAt:  time.Time{},
				ExpiresAt: expTime,
			},
			err: nil,
		},
		{
			desc: "save with invalid id",
			key: auth.Key{
				ID:        invalidID,
				Type:      auth.APIKey,
				Issuer:    issuer,
				Subject:   generateID(t),
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "save with invalid subject",
			key: auth.Key{
				ID:        generateID(t),
				Type:      auth.APIKey,
				Issuer:    issuer,
				Subject:   invalidID,
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "save with invalid issuer",
			key: auth.Key{
				ID:        generateID(t),
				Type:      auth.APIKey,
				Issuer:    invalidID,
				Subject:   generateID(t),
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestKeyRetrieve(t *testing.T) {
	repo := postgres.New(database)

	key := auth.Key{
		ID:        generateID(t),
		Subject:   generateID(t),
		IssuedAt:  time.Now(),
		Issuer:    generateID(t),
		ExpiresAt: expTime,
	}
	_, err := repo.Save(context.Background(), key)
	assert.Nil(t, err, fmt.Sprintf("Storing Key expected to succeed: %s", err))

	cases := []struct {
		desc   string
		id     string
		issuer string
		err    error
	}{
		{
			desc:   "retrieve an existing key",
			id:     key.ID,
			issuer: key.Issuer,
			err:    nil,
		},
		{
			desc:   "retrieve key with empty issuer id",
			id:     key.ID,
			issuer: "",
			err:    repoerr.ErrNotFound,
		},
		{
			desc:   "retrieve non-existent key",
			id:     "",
			issuer: key.Issuer,
			err:    repoerr.ErrNotFound,
		},
		{
			desc:   "retrieve non-existent key with empty issuer id",
			id:     "",
			issuer: "",
			err:    repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.Retrieve(context.Background(), tc.issuer, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestKeyRemove(t *testing.T) {
	repo := postgres.New(database)

	key := auth.Key{
		ID:        generateID(t),
		Subject:   generateID(t),
		IssuedAt:  time.Now(),
		Issuer:    generateID(t),
		ExpiresAt: expTime,
	}
	_, err := repo.Save(context.Background(), key)
	assert.Nil(t, err, fmt.Sprintf("Storing Key expected to succeed: %s", err))

	cases := []struct {
		desc   string
		id     string
		issuer string
		err    error
	}{
		{
			desc:   "remove an existing key",
			id:     key.ID,
			issuer: key.Issuer,
			err:    nil,
		},
		{
			desc:   "remove key that has already been removed",
			id:     key.ID,
			issuer: key.Issuer,
			err:    nil,
		},
		{
			desc:   "remove key that does not exist",
			id:     generateID(t),
			issuer: generateID(t),
			err:    nil,
		},
		{
			desc:   "remove key with empty issuer id",
			id:     key.ID,
			issuer: "",
			err:    nil,
		},
		{
			desc:   "remove key with empty id",
			id:     "",
			issuer: key.Issuer,
			err:    nil,
		},
		{
			desc:   "remove key with empty id and issuer id",
			id:     "",
			issuer: "",
			err:    nil,
		},
	}

	for _, tc := range cases {
		err := repo.Remove(context.Background(), tc.issuer, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
