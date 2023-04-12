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
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		user     users.User
		response string
		err      error
	}{
		{
			desc: "new user",
			user: users.User{
				ID:       uid,
				Email:    email,
				Password: "pass",
				Status:   users.EnabledStatusKey,
			},
			response: uid,
			err:      nil,
		},
		{
			desc: "duplicate user",
			user: users.User{
				ID:       uid,
				Email:    email,
				Password: "pass",
				Status:   users.EnabledStatusKey,
			},
			response: "",
			err:      errors.ErrConflict,
		},
		{
			desc: "invalid user status",
			user: users.User{
				ID:       uid,
				Email:    email,
				Password: "pass",
				Status:   "invalid",
			},
			response: "",
			err:      errors.ErrMalformedEntity,
		},
	}

	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	for _, tc := range cases {
		resp, err := repo.Save(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, resp, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.response, resp))
	}
}

func TestSingleUserRetrieval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewUserRepo(dbMiddleware)

	email := "user-retrieval@example.com"

	uid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	user := users.User{
		ID:       uid,
		Email:    email,
		Password: "pass",
		Status:   users.EnabledStatusKey,
		Metadata: make(users.Metadata),
	}

	_, err = repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		email    string
		response users.User
		err      error
	}{
		{
			desc:     "existing user",
			email:    email,
			response: user,
			err:      nil,
		},
		{
			desc:     "non-existing user",
			email:    "unknown@example.com",
			response: users.User{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		resp, err := repo.RetrieveByEmail(context.Background(), tc.email)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response.ID, resp.ID, fmt.Sprintf("%s: got incorrect user from RetrieveByEmail", tc.desc))
		assert.Equal(t, tc.response.Email, resp.Email, fmt.Sprintf("%s: got incorrect user from RetrieveByEmail", tc.desc))
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
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		email := fmt.Sprintf("TestRetrieveAll%d@example.com", i)
		user := users.User{
			ID:       uid,
			Email:    email,
			Password: "pass",
			Status:   users.EnabledStatusKey,
		}
		if i < metaNum {
			user.Metadata = meta
		}
		ids = append(ids, uid)
		_, err = userRepo.Save(context.Background(), user)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := []struct {
		desc     string
		email    string
		offset   uint64
		limit    uint64
		size     uint64
		total    uint64
		ids      []string
		metadata users.Metadata
	}{
		{
			desc:   "retrieve all users filtered by email",
			email:  "All",
			offset: 0,
			limit:  nUsers,
			size:   nUsers,
			total:  nUsers,
			ids:    ids,
		},
		{
			desc:   "retrieve all users by email with limit and offset",
			email:  "All",
			offset: 2,
			limit:  5,
			size:   5,
			total:  nUsers,
			ids:    ids,
		},
		{
			desc:     "retrieve all users by metadata",
			email:    "All",
			offset:   0,
			limit:    nUsers,
			size:     metaNum,
			total:    nUsers,
			metadata: meta,
			ids:      ids,
		},
		{
			desc:     "retrieve users by metadata and ids",
			email:    "All",
			offset:   0,
			limit:    nUsers,
			size:     1,
			total:    nUsers,
			metadata: meta,
			ids:      []string{ids[0]},
		},
		{
			desc:     "retrieve users by wrong metadata",
			email:    "All",
			offset:   0,
			limit:    nUsers,
			size:     0,
			total:    nUsers,
			metadata: wrongMeta,
			ids:      ids,
		},
		{
			desc:     "retrieve users by wrong metadata and ids",
			email:    "All",
			offset:   0,
			limit:    nUsers,
			size:     0,
			total:    nUsers,
			metadata: wrongMeta,
			ids:      []string{ids[0]},
		},
		{
			desc:   "retrieve all users by list of ids with limit and offset",
			email:  "All",
			offset: 2,
			limit:  5,
			size:   5,
			total:  nUsers,
			ids:    ids,
		},
		{
			desc:     "retrieve all users by list of ids with limit and offset and metadata",
			email:    "All",
			offset:   1,
			limit:    5,
			size:     1,
			total:    nUsers,
			ids:      ids[0:5],
			metadata: meta,
		},
		{
			desc:   "retrieve all users from empty ids",
			email:  "All",
			offset: 0,
			limit:  nUsers,
			size:   nUsers,
			total:  nUsers,
			ids:    []string{},
		},
		{
			desc:   "retrieve all users from empty ids with offset",
			email:  "All",
			offset: 1,
			limit:  5,
			size:   5,
			total:  nUsers,
			ids:    []string{},
		},
	}
	for _, tc := range cases {
		pm := users.PageMetadata{
			Offset:   tc.offset,
			Limit:    tc.limit,
			Email:    tc.email,
			Metadata: tc.metadata,
			Status:   users.EnabledStatusKey,
		}

		page, err := userRepo.RetrieveAll(context.Background(), tc.ids, pm)
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", tc.desc, err))
	}
}
