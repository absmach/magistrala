// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/authn"
	"github.com/mainflux/mainflux/authn/postgres"
	"github.com/mainflux/mainflux/pkg/errors"
	uuidProvider "github.com/mainflux/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
)

func TestKeySave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	email := "user-save@example.com"
	expTime := time.Now().Add(5 * time.Minute)
	id, _ := uuidProvider.New().ID()
	cases := []struct {
		desc string
		key  authn.Key
		err  error
	}{
		{
			desc: "save a new key",
			key: authn.Key{
				Issuer:    email,
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
				ID:        id,
			},
			err: nil,
		},
		{
			desc: "save with duplicate id",
			key: authn.Key{
				Issuer:    email,
				IssuedAt:  time.Now(),
				ExpiresAt: expTime,
				ID:        id,
			},
			err: authn.ErrConflict,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestKeyRetrieve(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	email := "user-save@example.com"
	expTime := time.Now().Add(5 * time.Minute)
	id, _ := uuidProvider.New().ID()
	key := authn.Key{
		Issuer:    email,
		IssuedAt:  time.Now(),
		ExpiresAt: expTime,
		ID:        id,
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
			desc:   "retrieve unauthorized",
			id:     key.ID,
			issuer: "",
			err:    authn.ErrNotFound,
		},
		{
			desc:   "retrieve unknown key",
			id:     "",
			issuer: key.Issuer,
			err:    authn.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.Retrieve(context.Background(), tc.issuer, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestKeyRemove(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.New(dbMiddleware)

	email := "user-save@example.com"
	expTime := time.Now().Add(5 * time.Minute)
	id, _ := uuidProvider.New().ID()
	key := authn.Key{
		Issuer:    email,
		IssuedAt:  time.Now(),
		ExpiresAt: expTime,
		ID:        id,
	}
	_, err := repo.Save(opentracing.ContextWithSpan(context.Background(), opentracing.StartSpan("")), key)
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
			desc:   "remove key that does not exist",
			id:     key.ID,
			issuer: key.Issuer,
			err:    nil,
		},
	}

	for _, tc := range cases {
		err := repo.Remove(context.Background(), tc.issuer, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
