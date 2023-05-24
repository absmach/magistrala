// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/auth/postgres"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/ulid"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const email = "user-save@example.com"

var (
	expTime      = time.Now().Add(5 * time.Minute)
	idProvider   = uuid.New()
	ulidProvider = ulid.New()
)

func TestKeySave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	id, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		key  auth.Key
		err  error
	}{
		{
			desc: "save a new key",
			key: auth.Key{
				Subject:   email,
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
				ID:        id,
				IssuerID:  id,
			},
			err: nil,
		},
		{
			desc: "save with duplicate id",
			key: auth.Key{
				Subject:   email,
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
				ID:        id,
				IssuerID:  id,
			},
			err: errors.ErrConflict,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestKeyRetrieveByID(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	id, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	key := auth.Key{
		Subject:   email,
		IssuedAt:  time.Now(),
		ExpiresAt: expTime,
		ID:        id,
		IssuerID:  id,
	}
	_, err = repo.Save(context.Background(), key)
	assert.Nil(t, err, fmt.Sprintf("Storing Key expected to succeed: %s", err))
	cases := []struct {
		desc  string
		id    string
		owner string
		err   error
	}{
		{
			desc:  "retrieve an existing key",
			id:    key.ID,
			owner: key.IssuerID,
			err:   nil,
		},
		{
			desc:  "retrieve key with empty issuer id",
			id:    key.ID,
			owner: "",
			err:   errors.ErrNotFound,
		},
		{
			desc:  "retrieve non-existent key",
			id:    "",
			owner: key.IssuerID,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.RetrieveByID(context.Background(), tc.owner, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestKeyRetrieveAll(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	issuerID1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	issuerID2, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	n := uint64(100)
	for i := uint64(0); i < n; i++ {
		id, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		key := auth.Key{
			Subject:   fmt.Sprintf("key-%d@email.com", i),
			IssuedAt:  time.Now(),
			ExpiresAt: expTime,
			ID:        id,
			IssuerID:  issuerID1,
			Type:      auth.LoginKey,
		}
		if i%10 == 0 {
			key.Type = auth.APIKey
		}
		if i == n-1 {
			key.IssuerID = issuerID2
		}
		_, err = repo.Save(context.Background(), key)
		assert.Nil(t, err, fmt.Sprintf("Storing Key expected to succeed: %s", err))
	}

	cases := map[string]struct {
		owner        string
		pageMetadata auth.PageMetadata
		size         uint64
	}{
		"retrieve all keys": {
			owner: issuerID1,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			size: n - 1,
		},
		"retrieve all keys with different issuer ID": {
			owner: issuerID2,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			size: 1,
		},
		"retrieve subset of keys with existing owner": {
			owner: issuerID1,
			pageMetadata: auth.PageMetadata{
				Offset: n/2 - 1,
				Limit:  n,
				Total:  n,
			},
			size: n / 2,
		},
		"retrieve keys with existing subject": {
			owner: issuerID1,
			pageMetadata: auth.PageMetadata{
				Offset:  0,
				Limit:   n,
				Subject: "key-10@email.com",
			},
			size: 1,
		},
		"retrieve keys with non-existing subject": {
			owner: issuerID1,
			pageMetadata: auth.PageMetadata{
				Offset:  0,
				Limit:   n,
				Subject: "wrong",
				Total:   0,
			},
			size: 0,
		},
		"retrieve keys with existing type": {
			owner: issuerID1,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Type:   auth.APIKey,
			},
			size: 10,
		},
		"retrieve keys with non-existing type": {
			owner: issuerID1,
			pageMetadata: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  0,
				Type:   uint32(9),
			},
			size: 0,
		},
		"retrieve all keys with existing subject and type": {
			owner: issuerID1,
			pageMetadata: auth.PageMetadata{
				Offset:  0,
				Limit:   n,
				Subject: "key-10@email.com",
				Type:    auth.APIKey,
			},
			size: 1,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.owner, tc.pageMetadata)
		size := uint64(len(page.Keys))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		// assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestKeyRemove(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	id, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	key := auth.Key{
		Subject:   email,
		IssuedAt:  time.Now(),
		ExpiresAt: expTime,
		ID:        id,
		IssuerID:  id,
	}
	_, err = repo.Save(opentracing.ContextWithSpan(context.Background(), opentracing.StartSpan("")), key)
	assert.Nil(t, err, fmt.Sprintf("Storing Key expected to succeed: %s", err))
	cases := []struct {
		desc  string
		id    string
		owner string
		err   error
	}{
		{
			desc:  "remove an existing key",
			id:    key.ID,
			owner: key.IssuerID,
			err:   nil,
		},
		{
			desc:  "remove key that does not exist",
			id:    key.ID,
			owner: key.IssuerID,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := repo.Remove(context.Background(), tc.owner, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
