// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var idProvider = uuid.New()

func TestUserSave(t *testing.T) {
	email := "user-save@example.com"

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{
			desc: "new user",
			user: users.User{
				ID:       uid,
				Email:    email,
				Password: "pass",
			},
			err: nil,
		},
		{
			desc: "duplicate user",
			user: users.User{
				ID:       uid,
				Email:    email,
				Password: "pass",
			},
			err: users.ErrConflict,
		},
	}

	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleUserRetrieval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	email := "user-retrieval@example.com"

	uid, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user := users.User{
		ID:       uid,
		Email:    email,
		Password: "pass",
	}

	_, err = repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		email string
		err   error
	}{
		"existing user":     {email, nil},
		"non-existing user": {"unknown@example.com", users.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := repo.RetrieveByEmail(context.Background(), tc.email)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	userRepo := postgres.NewUserRepo(dbMiddleware)
	metaNum := uint64(2)
	var nUsers = uint64(10)

	meta := users.Metadata{
		"admin": "true",
	}

	wrongMeta := users.Metadata{
		"wrong": "true",
	}

	var ids []string
	for i := uint64(0); i < nUsers; i++ {
		uid, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		email := fmt.Sprintf("TestRetrieveAll%d@example.com", i)
		user := users.User{
			ID:       uid,
			Email:    email,
			Password: "pass",
		}
		if i < metaNum {
			user.Metadata = meta
		}
		ids = append(ids, uid)
		_, err = userRepo.Save(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]struct {
		email    string
		offset   uint64
		limit    uint64
		size     uint64
		total    uint64
		ids      []string
		metadata users.Metadata
	}{
		"retrieve all users filtered by email": {
			email:  "All",
			offset: 0,
			limit:  nUsers,
			size:   nUsers,
			total:  nUsers,
			ids:    ids,
		},
		"retrieve all users by email with limit and offset": {
			email:  "All",
			offset: 2,
			limit:  5,
			size:   5,
			total:  nUsers,
			ids:    ids,
		},
		"retrieve all users by metadata": {
			email:    "All",
			offset:   0,
			limit:    nUsers,
			size:     metaNum,
			total:    nUsers,
			metadata: meta,
			ids:      ids,
		},
		"retrieve users by metadata and ids": {
			email:    "All",
			offset:   0,
			limit:    nUsers,
			size:     1,
			total:    nUsers,
			metadata: meta,
			ids:      []string{ids[0]},
		},
		"retrieve users by wrong metadata": {
			email:    "All",
			offset:   0,
			limit:    nUsers,
			size:     0,
			total:    nUsers,
			metadata: wrongMeta,
			ids:      ids,
		},
		"retrieve users by wrong metadata and ids": {
			email:    "All",
			offset:   0,
			limit:    nUsers,
			size:     0,
			total:    nUsers,
			metadata: wrongMeta,
			ids:      []string{ids[0]},
		},
		"retrieve all users by list of ids with limit and offset": {
			email:  "All",
			offset: 2,
			limit:  5,
			size:   5,
			total:  nUsers,
			ids:    ids,
		},
		"retrieve all users by list of ids with limit and offset and metadata": {
			email:    "All",
			offset:   1,
			limit:    5,
			size:     1,
			total:    nUsers,
			ids:      ids[0:5],
			metadata: meta,
		},
		"retrieve all users from empty ids": {
			email:  "All",
			offset: 0,
			limit:  nUsers,
			size:   nUsers,
			total:  nUsers,
			ids:    []string{},
		},
		"retrieve all users from empty ids with offset": {
			email:  "All",
			offset: 1,
			limit:  5,
			size:   5,
			total:  nUsers,
			ids:    []string{},
		},
	}
	for desc, tc := range cases {
		page, err := userRepo.RetrieveAll(context.Background(), tc.offset, tc.limit, tc.ids, tc.email, tc.metadata)
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}
