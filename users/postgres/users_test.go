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

func TestUserSave(t *testing.T) {
	email := "user-save@example.com"

	uid, err := uuid.New().ID()
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

	uid, err := uuid.New().ID()
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

func TestRetrieveMembers(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	groupRepo := postgres.NewGroupRepo(dbMiddleware)
	userRepo := postgres.NewUserRepo(dbMiddleware)
	var nUsers = uint64(10)
	var usrs []users.User
	for i := uint64(0); i < nUsers; i++ {
		uid, err := uuid.New().ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		email := fmt.Sprintf("TestRetrieveMembers%d@example.com", i)
		user := users.User{
			ID:       uid,
			Email:    email,
			Password: "pass",
		}
		_, err = userRepo.Save(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("saving user error: %s", err))
		u, _ := userRepo.RetrieveByEmail(context.Background(), user.Email)
		usrs = append(usrs, u)
	}
	uid, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("user uuid error: %s", err))
	group := users.Group{
		ID:   uid,
		Name: "TestMembers",
	}

	g, err := groupRepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("group save got unexpected error: %s", err))

	for _, u := range usrs {
		err := groupRepo.Assign(context.Background(), u.ID, g.ID)
		require.Nil(t, err, fmt.Sprintf("group user assign got unexpected error: %s", err))
	}

	cases := map[string]struct {
		group    string
		offset   uint64
		limit    uint64
		size     uint64
		total    uint64
		metadata users.Metadata
	}{
		"retrieve all users for existing group": {
			group:  g.ID,
			offset: 0,
			limit:  nUsers,
			size:   nUsers,
			total:  nUsers,
		},
	}

	for desc, tc := range cases {
		page, err := userRepo.RetrieveMembers(context.Background(), tc.group, tc.offset, tc.limit, tc.metadata)
		size := uint64(len(usrs))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	userRepo := postgres.NewUserRepo(dbMiddleware)
	var nUsers = uint64(10)

	for i := uint64(0); i < nUsers; i++ {
		uid, err := uuid.New().ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		email := fmt.Sprintf("TestRetrieveAll%d@example.com", i)
		user := users.User{
			ID:       uid,
			Email:    email,
			Password: "pass",
		}
		_, err = userRepo.Save(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	cases := map[string]struct {
		email    string
		offset   uint64
		limit    uint64
		size     uint64
		total    uint64
		metadata users.Metadata
	}{
		"retrieve all users filtered by email": {
			email:  "All",
			offset: 0,
			limit:  nUsers,
			size:   nUsers,
			total:  nUsers,
		},
		"retrieve all users by email with limit and offset": {
			email:  "All",
			offset: 2,
			limit:  5,
			size:   5,
			total:  nUsers,
		},
	}

	for desc, tc := range cases {
		page, err := userRepo.RetrieveAll(context.Background(), tc.offset, tc.limit, tc.email, tc.metadata)
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}
