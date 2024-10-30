// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/users"
	cpostgres "github.com/absmach/magistrala/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxNameSize = 254

var (
	invalidName = strings.Repeat("m", maxNameSize+10)
	password    = "$tr0ngPassw0rd"
	namesgen    = namegenerator.NewGenerator()
)

func TestUsersSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})

	repo := cpostgres.NewRepository(database)

	uid := testsutil.GenerateUUID(t)

	first_name := namesgen.Generate()
	last_name := namesgen.Generate()
	username := namesgen.Generate()

	email := first_name + "@example.com"

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{
			desc: "add new user successfully",
			user: users.User{
				ID:        uid,
				FirstName: first_name,
				LastName:  last_name,
				Email:     email,
				Credentials: users.Credentials{
					Username: username,
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   users.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add user with duplicate user email",
			user: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: first_name,
				LastName:  last_name,
				Email:     email,
				Credentials: users.Credentials{
					Username: namesgen.Generate(),
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   users.EnabledStatus,
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add user with duplicate user name",
			user: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: namesgen.Generate(),
				LastName:  last_name,
				Email:     namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					Username: username,
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   users.EnabledStatus,
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add user with invalid user id",
			user: users.User{
				ID:        invalidName,
				FirstName: namesgen.Generate(),
				LastName:  namesgen.Generate(),
				Email:     namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					Username: username,
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   users.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add user with invalid user name",
			user: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: first_name,
				LastName:  last_name,
				Email:     namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					Username: invalidName,
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   users.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add user with a missing username",
			user: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: first_name,
				LastName:  last_name,
				Email:     namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					Secret: password,
				},
				Metadata: users.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add user with a missing user secret",
			user: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: namesgen.Generate(),
				LastName:  namesgen.Generate(),
				Email:     namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					Username: namesgen.Generate(),
				},
				Metadata: users.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add a user with invalid metadata",
			user: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: namesgen.Generate(),
				Email:     namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					Username: username,
					Secret:   password,
				},
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		rUser, err := repo.Save(context.Background(), tc.user)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			rUser.Credentials.Secret = tc.user.Credentials.Secret
			assert.Equal(t, tc.user, rUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.user, rUser))
		}
	}
}

func TestIsPlatformAdmin(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})

	repo := cpostgres.NewRepository(database)

	first_name := namesgen.Generate()
	last_name := namesgen.Generate()
	username := namesgen.Generate()
	email := first_name + "@example.com"

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{
			desc: "authorize check for super user",
			user: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: first_name,
				LastName:  last_name,
				Email:     email,
				Credentials: users.Credentials{
					Username: username,
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   users.EnabledStatus,
				Role:     users.AdminRole,
			},
			err: nil,
		},
		{
			desc: "unauthorize user",
			user: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: first_name,
				LastName:  last_name,
				Email:     namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					Username: namesgen.Generate(),
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   users.EnabledStatus,
				Role:     users.UserRole,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.user)
		require.Nil(t, err, fmt.Sprintf("%s: save user unexpected error: %s", tc.desc, err))
		err = repo.CheckSuperAdmin(context.Background(), tc.user.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})

	repo := cpostgres.NewRepository(database)

	user := users.User{
		ID:        testsutil.GenerateUUID(t),
		FirstName: namesgen.Generate(),
		LastName:  namesgen.Generate(),
		Email:     namesgen.Generate() + "@example.com",
		Credentials: users.Credentials{
			Username: namesgen.Generate(),
			Secret:   password,
		},
		Metadata: users.Metadata{},
		Status:   users.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("failed to save users %s", user.ID))

	cases := []struct {
		desc   string
		userID string
		err    error
	}{
		{
			desc:   "retrieve existing user",
			userID: user.ID,
			err:    nil,
		},
		{
			desc:   "retrieve non-existing user",
			userID: invalidName,
			err:    repoerr.ErrNotFound,
		},
		{
			desc:   "retrieve with empty user id",
			userID: "",
			err:    repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.RetrieveByID(context.Background(), tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})

	repo := cpostgres.NewRepository(database)

	num := 200
	var items, enabledUsers []users.User
	for i := 0; i < num; i++ {
		user := users.User{
			ID:        testsutil.GenerateUUID(t),
			FirstName: namesgen.Generate(),
			LastName:  namesgen.Generate(),
			Email:     namesgen.Generate() + "@example.com",
			Credentials: users.Credentials{
				Username: namesgen.Generate(),
				Secret:   "",
			},
			Metadata: users.Metadata{},
			Status:   users.EnabledStatus,
			Tags:     []string{"tag1"},
		}
		if i%50 == 0 {
			user.Metadata = map[string]interface{}{
				"key": "value",
			}
			user.Role = users.AdminRole
			user.Status = users.DisabledStatus
		}
		_, err := repo.Save(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("failed to save user %s", user.ID))
		items = append(items, user)
		if user.Status == users.EnabledStatus {
			enabledUsers = append(enabledUsers, user)
		}
	}

	cases := []struct {
		desc     string
		pageMeta users.Page
		page     users.UsersPage
		err      error
	}{
		{
			desc: "retrieve first page of users",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  50,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  200,
					Offset: 0,
					Limit:  50,
				},
				Users: items[0:50],
			},
			err: nil,
		},
		{
			desc: "retrieve second page of users",
			pageMeta: users.Page{
				Offset: 50,
				Limit:  200,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  200,
					Offset: 50,
					Limit:  200,
				},
				Users: items[50:200],
			},
			err: nil,
		},
		{
			desc: "retrieve users with limit",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  50,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  50,
				},
				Users: items[:50],
			},
		},
		{
			desc: "retrieve with offset out of range",
			pageMeta: users.Page{
				Offset: 1000,
				Limit:  200,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  200,
					Offset: 1000,
					Limit:  200,
				},
				Users: []users.User{},
			},
			err: nil,
		},
		{
			desc: "retrieve with limit out of range",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  1000,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  200,
					Offset: 0,
					Limit:  1000,
				},
				Users: items,
			},
			err: nil,
		},
		{
			desc:     "retrieve with empty page",
			pageMeta: users.Page{},
			page: users.UsersPage{
				Page: users.Page{
					Total:  196, // number of enabled users
					Offset: 0,
					Limit:  0,
				},
				Users: []users.User{},
			},
			err: nil,
		},
		{
			desc: "retrieve with user id",
			pageMeta: users.Page{
				IDs:    []string{items[0].ID},
				Offset: 0,
				Limit:  3,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  3,
				},
				Users: []users.User{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve with invalid user id",
			pageMeta: users.Page{
				IDs:    []string{invalidName},
				Offset: 0,
				Limit:  3,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  3,
				},
				Users: []users.User{},
			},
			err: nil,
		},
		{
			desc: "retrieve with first name",
			pageMeta: users.Page{
				FirstName: items[0].FirstName,
				Offset:    0,
				Limit:     3,
				Role:      users.AllRole,
				Status:    users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  3,
				},
				Users: []users.User{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve with username",
			pageMeta: users.Page{
				Username: items[0].Credentials.Username,
				Offset:   0,
				Limit:    3,
				Role:     users.AllRole,
				Status:   users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  3,
				},
				Users: []users.User{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve with enabled status",
			pageMeta: users.Page{
				Status: users.EnabledStatus,
				Offset: 0,
				Limit:  200,
				Role:   users.AllRole,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  196,
					Offset: 0,
					Limit:  200,
				},
				Users: enabledUsers,
			},
			err: nil,
		},
		{
			desc: "retrieve with disabled status",
			pageMeta: users.Page{
				Status: users.DisabledStatus,
				Offset: 0,
				Limit:  200,
				Role:   users.AllRole,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  4,
					Offset: 0,
					Limit:  200,
				},
				Users: []users.User{items[0], items[50], items[100], items[150]},
			},
		},
		{
			desc: "retrieve with all status",
			pageMeta: users.Page{
				Status: users.AllStatus,
				Offset: 0,
				Limit:  200,
				Role:   users.AllRole,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  200,
					Offset: 0,
					Limit:  200,
				},
				Users: items,
			},
		},
		{
			desc: "retrieve by tags",
			pageMeta: users.Page{
				Tag:    "tag1",
				Offset: 0,
				Limit:  200,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  200,
					Offset: 0,
					Limit:  200,
				},
				Users: items,
			},
			err: nil,
		},
		{
			desc: "retrieve with invalid first name",
			pageMeta: users.Page{
				FirstName: invalidName,
				Offset:    0,
				Limit:     3,
				Role:      users.AllRole,
				Status:    users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  3,
				},
				Users: []users.User{},
			},
		},
		{
			desc: "retrieve with metadata",
			pageMeta: users.Page{
				Metadata: map[string]interface{}{
					"key": "value",
				},
				Offset: 0,
				Limit:  200,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  4,
					Offset: 0,
					Limit:  200,
				},
				Users: []users.User{items[0], items[50], items[100], items[150]},
			},
			err: nil,
		},
		{
			desc: "retrieve with invalid metadata",
			pageMeta: users.Page{
				Metadata: map[string]interface{}{
					"key": "value1",
				},
				Offset: 0,
				Limit:  200,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  200,
				},
				Users: []users.User{},
			},
			err: nil,
		},
		{
			desc: "retrieve with role",
			pageMeta: users.Page{
				Role:   users.AdminRole,
				Offset: 0,
				Limit:  200,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  4,
					Offset: 0,
					Limit:  200,
				},
				Users: []users.User{items[0], items[50], items[100], items[150]},
			},
			err: nil,
		},
		{
			desc: "retrieve with invalid role",
			pageMeta: users.Page{
				Role:   users.AdminRole + 2,
				Offset: 0,
				Limit:  200,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  200,
				},
				Users: []users.User{},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.pageMeta)

		assert.Equal(t, tc.page.Total, page.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Total, page.Total))
		assert.Equal(t, tc.page.Offset, page.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Offset, page.Offset))
		assert.Equal(t, tc.page.Limit, page.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Limit, page.Limit))
		assert.Equal(t, tc.page.Page, page.Page, fmt.Sprintf("%s: expected  %v, got %v", tc.desc, tc.page, page))
		assert.ElementsMatch(t, tc.page.Users, page.Users, fmt.Sprintf("%s: expected %v, got %v", tc.desc, tc.page.Users, page.Users))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
