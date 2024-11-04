// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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
	emailSuffix = "@example.com"
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

func TestSearch(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	nUsers := uint64(200)
	expectedUsers := []users.User{}
	for i := 0; i < int(nUsers); i++ {
		user := generateUser(t, users.EnabledStatus, repo)

		expectedUsers = append(expectedUsers, users.User{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Credentials: users.Credentials{
				Username: user.Credentials.Username,
			},
			CreatedAt: user.CreatedAt,
		})
	}

	page, err := repo.RetrieveAll(context.Background(), users.Page{Offset: 0, Limit: nUsers})
	require.Nil(t, err, fmt.Sprintf("retrieve all users unexpected error: %s", err))
	assert.Equal(t, nUsers, page.Total)

	cases := []struct {
		desc     string
		page     users.Page
		response users.UsersPage
		err      error
	}{
		{
			desc: "with empty page",
			page: users.Page{},
			response: users.UsersPage{
				Users: []users.User(nil),
				Page: users.Page{
					Total:  nUsers,
					Offset: 0,
					Limit:  0,
				},
			},
			err: nil,
		},
		{
			desc: "with offset only",
			page: users.Page{
				Offset: 50,
			},
			response: users.UsersPage{
				Users: []users.User(nil),
				Page: users.Page{
					Total:  nUsers,
					Offset: 50,
					Limit:  0,
				},
			},
			err: nil,
		},
		{
			desc: "with limit only",
			page: users.Page{
				Limit: 10,
				Order: "name",
				Dir:   "asc",
			},
			response: users.UsersPage{
				Users: expectedUsers[0:10],
				Page: users.Page{
					Total:  nUsers,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all users",
			page: users.Page{
				Offset: 0,
				Limit:  nUsers,
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  nUsers,
					Offset: 0,
					Limit:  nUsers,
				},
				Users: expectedUsers,
			},
		},
		{
			desc: "with offset and limit",
			page: users.Page{
				Offset: 10,
				Limit:  10,
				Order:  "name",
				Dir:    "asc",
			},
			response: users.UsersPage{
				Users: expectedUsers[10:20],
				Page: users.Page{
					Total:  nUsers,
					Offset: 10,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with offset out of range and limit",
			page: users.Page{
				Offset: 1000,
				Limit:  50,
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  nUsers,
					Offset: 1000,
					Limit:  50,
				},
				Users: []users.User(nil),
			},
		},
		{
			desc: "with offset and limit out of range",
			page: users.Page{
				Offset: 190,
				Limit:  50,
				Order:  "name",
				Dir:    "asc",
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  nUsers,
					Offset: 190,
					Limit:  50,
				},
				Users: expectedUsers[190:200],
			},
		},
		{
			desc: "with shorter name",
			page: users.Page{
				FirstName: expectedUsers[0].FirstName[:4],
				Offset:    0,
				Limit:     10,
				Order:     "first_name",
				Dir:       "asc",
			},
			response: users.UsersPage{
				Users: findUsers(expectedUsers, expectedUsers[0].FirstName[:4], 0, 10),
				Page: users.Page{
					Total:  nUsers,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with longer name",
			page: users.Page{
				FirstName: expectedUsers[0].FirstName,
				Offset:    0,
				Limit:     10,
			},
			response: users.UsersPage{
				Users: []users.User{expectedUsers[0]},
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with name SQL injected",
			page: users.Page{
				FirstName: fmt.Sprintf("%s' OR '1'='1", expectedUsers[0].FirstName[:1]),
				Offset:    0,
				Limit:     10,
			},
			response: users.UsersPage{
				Users: []users.User(nil),
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with shorter email",
			page: users.Page{
				Email:  expectedUsers[0].FirstName[:4],
				Offset: 0,
				Limit:  10,
				Order:  "first_name",
				Dir:    "asc",
			},
			response: users.UsersPage{
				Users: findUsers(expectedUsers, expectedUsers[0].FirstName[:4], 0, 10),
				Page: users.Page{
					Total:  nUsers,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with Identity SQL injected",
			page: users.Page{
				Email:  fmt.Sprintf("%s' OR '1'='1", expectedUsers[0].FirstName[:1]),
				Offset: 0,
				Limit:  10,
			},
			response: users.UsersPage{
				Users: []users.User(nil),
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with unknown name",
			page: users.Page{
				FirstName: namesgen.Generate(),
				Offset:    0,
				Limit:     10,
			},
			response: users.UsersPage{
				Users: []users.User(nil),
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with unknown email",
			page: users.Page{
				Email:  namesgen.Generate(),
				Offset: 0,
				Limit:  10,
			},
			response: users.UsersPage{
				Users: []users.User(nil),
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with name in asc order",
			page: users.Page{
				Order:     "first_name",
				Dir:       "asc",
				FirstName: expectedUsers[0].FirstName[:1],
				Offset:    0,
				Limit:     10,
			},
			response: users.UsersPage{},
			err:      nil,
		},
		{
			desc: "with name in desc order",
			page: users.Page{
				Order:     "first_name",
				Dir:       "desc",
				FirstName: expectedUsers[0].FirstName[:1],
				Offset:    0,
				Limit:     10,
			},
			response: users.UsersPage{},
			err:      nil,
		},
		{
			desc: "with last name in asc order",
			page: users.Page{
				LastName: expectedUsers[0].LastName[:1],
				Order:    "last_name",
				Dir:      "asc",
			},
			response: users.UsersPage{
				Users: []users.User{expectedUsers[0]},
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  1,
				},
			},
			err: nil,
		},
		{
			desc: "with username in asc order",
			page: users.Page{
				Username: expectedUsers[0].Credentials.Username[:1],
				Order:    "username",
				Dir:      "asc",
			},
			response: users.UsersPage{
				Users: []users.User{expectedUsers[0]},
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  1,
				},
			},
			err: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			switch response, err := repo.SearchUsers(context.Background(), c.page); {
			case err == nil:
				if c.page.Order != "" && c.page.Dir != "" {
					c.response = response
				}
				assert.Nil(t, err)
				assert.Equal(t, c.response.Total, response.Total)
				assert.Equal(t, c.response.Limit, response.Limit)
				assert.Equal(t, c.response.Offset, response.Offset)
				assert.ElementsMatch(t, response.Users, c.response.Users)
			default:
				assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	user1 := generateUser(t, users.EnabledStatus, repo)
	user2 := generateUser(t, users.DisabledStatus, repo)

	cases := []struct {
		desc   string
		update string
		user   users.User
		err    error
	}{
		{
			desc:   "update metadata for enabled user",
			update: "metadata",
			user: users.User{
				ID: user1.ID,
				Metadata: users.Metadata{
					"update": namesgen.Generate(),
				},
			},
			err: nil,
		},
		{
			desc:   "update malformed metadata for enabled user",
			update: "metadata",
			user: users.User{
				ID: user1.ID,
				Metadata: users.Metadata{
					"update": make(chan int),
				},
			},
			err: repoerr.ErrUpdateEntity,
		},
		{
			desc:   "update metadata for disabled user",
			update: "metadata",
			user: users.User{
				ID: user2.ID,
				Metadata: users.Metadata{
					"update": namesgen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update first name for enabled user",
			update: "first_name",
			user: users.User{
				ID:        user1.ID,
				FirstName: namesgen.Generate(),
			},
			err: nil,
		},
		{
			desc:   "update first name for disabled user",
			update: "first_name",
			user: users.User{
				ID:        user2.ID,
				FirstName: namesgen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update metadata for invalid user",
			update: "metadata",
			user: users.User{
				ID: testsutil.GenerateUUID(t),
				Metadata: users.Metadata{
					"update": namesgen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update first name for empty user",
			update: "first_name",
			user: users.User{
				FirstName: namesgen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update last name for enabled user",
			update: "last_name",
			user: users.User{
				ID:       user1.ID,
				LastName: namesgen.Generate(),
			},
			err: nil,
		},
		{
			desc:   "update last name for disabled user",
			update: "last_name",
			user: users.User{
				ID:       user2.ID,
				LastName: namesgen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update last name for invalid user",
			update: "last_name",
			user: users.User{
				ID:       testsutil.GenerateUUID(t),
				LastName: namesgen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update tags for enabled user",
			user: users.User{
				ID:   user1.ID,
				Tags: namesgen.GenerateMultiple(5),
			},
			err: nil,
		},
		{
			desc: "update tags for disabled user",
			user: users.User{
				ID:   user2.ID,
				Tags: namesgen.GenerateMultiple(5),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update tags for invalid user",
			user: users.User{
				ID:   testsutil.GenerateUUID(t),
				Tags: namesgen.GenerateMultiple(5),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update profile picture for enabled user",
			user: users.User{
				ID:             user1.ID,
				ProfilePicture: namesgen.Generate(),
			},
			err: nil,
		},
		{
			desc: "update profile picture for disabled user",
			user: users.User{
				ID:             user2.ID,
				ProfilePicture: namesgen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update profile picture for invalid user",
			user: users.User{
				ID:             testsutil.GenerateUUID(t),
				ProfilePicture: namesgen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update role for enabled user",
			user: users.User{
				ID:   user1.ID,
				Role: users.AdminRole,
			},
			err: nil,
		},
		{
			desc: "update role for disabled user",
			user: users.User{
				ID:   user2.ID,
				Role: users.AdminRole,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update role for invalid user",
			user: users.User{
				ID:   testsutil.GenerateUUID(t),
				Role: users.AdminRole,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update email for enabled user",
			user: users.User{
				ID:    user1.ID,
				Email: namesgen.Generate() + emailSuffix,
			},
			err: nil,
		},
		{
			desc: "update email for disabled user",
			user: users.User{
				ID:    user2.ID,
				Email: namesgen.Generate() + emailSuffix,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update email for invalid user",
			user: users.User{
				ID:    testsutil.GenerateUUID(t),
				Email: namesgen.Generate() + emailSuffix,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.user.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.user.UpdatedBy = testsutil.GenerateUUID(t)
			expected, err := repo.Update(context.Background(), c.user)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				switch c.update {
				case "metadata":
					assert.Equal(t, c.user.Metadata, expected.Metadata)
				case "first_name":
					assert.Equal(t, c.user.FirstName, expected.FirstName)
				case "last_name":
					assert.Equal(t, c.user.LastName, expected.LastName)
				case "tags":
					assert.Equal(t, c.user.Tags, expected.Tags)
				case "profile_picture":
					assert.Equal(t, c.user.ProfilePicture, expected.ProfilePicture)
				case "role":
					assert.Equal(t, c.user.Role, expected.Role)
				case "email":
					assert.Equal(t, c.user.Email, expected.Email)
				}
				assert.Equal(t, c.user.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.user.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestUpdateUsername(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	user1 := generateUser(t, users.EnabledStatus, repo)
	user2 := generateUser(t, users.DisabledStatus, repo)

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{
			desc: "for enabled user",
			user: users.User{
				ID: user1.ID,
				Credentials: users.Credentials{
					Username: namesgen.Generate(),
				},
			},
			err: nil,
		},
		{
			desc: "for enabled user with existing username",
			user: users.User{
				ID: user1.ID,
				Credentials: users.Credentials{
					Username: user2.Credentials.Username,
				},
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "for disabled user",
			user: users.User{
				ID: user2.ID,
				Credentials: users.Credentials{
					Username: namesgen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid user",
			user: users.User{
				ID: testsutil.GenerateUUID(t),
				Credentials: users.Credentials{
					Username: namesgen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for empty user",
			user: users.User{},
			err:  repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.user.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.user.UpdatedBy = testsutil.GenerateUUID(t)
			expected, err := repo.UpdateUsername(context.Background(), c.user)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.user.Credentials.Username, expected.Credentials.Username)
				assert.Equal(t, c.user.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.user.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestUpdateSecret(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	user1 := generateUser(t, users.EnabledStatus, repo)
	user2 := generateUser(t, users.DisabledStatus, repo)

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{
			desc: "for enabled user",
			user: users.User{
				ID: user1.ID,
				Credentials: users.Credentials{
					Secret: "newpassword",
				},
			},
			err: nil,
		},
		{
			desc: "for disabled user",
			user: users.User{
				ID: user2.ID,
				Credentials: users.Credentials{
					Secret: "newpassword",
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid user",
			user: users.User{
				ID: testsutil.GenerateUUID(t),
				Credentials: users.Credentials{
					Secret: "newpassword",
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for empty user",
			user: users.User{},
			err:  repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.user.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.user.UpdatedBy = testsutil.GenerateUUID(t)
			_, err := repo.UpdateSecret(context.Background(), c.user)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				rc, err := repo.RetrieveByID(context.Background(), c.user.ID)
				require.Nil(t, err, fmt.Sprintf("retrieve user by id during update of secret unexpected error: %s", err))
				assert.Equal(t, c.user.Credentials.Secret, rc.Credentials.Secret)
				assert.Equal(t, c.user.UpdatedAt, rc.UpdatedAt)
				assert.Equal(t, c.user.UpdatedBy, rc.UpdatedBy)
			}
		})
	}
}

func TestChangeStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	user1 := generateUser(t, users.EnabledStatus, repo)
	user2 := generateUser(t, users.DisabledStatus, repo)

	cases := []struct {
		desc string
		user users.User
		err  error
	}{
		{
			desc: "for an enabled user",
			user: users.User{
				ID:     user1.ID,
				Status: users.DisabledStatus,
			},
			err: nil,
		},
		{
			desc: "for a disabled user",
			user: users.User{
				ID:     user2.ID,
				Status: users.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "for invalid user",
			user: users.User{
				ID:     testsutil.GenerateUUID(t),
				Status: users.DisabledStatus,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for empty user",
			user: users.User{},
			err:  repoerr.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.user.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.user.UpdatedBy = testsutil.GenerateUUID(t)
			expected, err := repo.ChangeStatus(context.Background(), c.user)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.user.Status, expected.Status)
				assert.Equal(t, c.user.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.user.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	user := generateUser(t, users.EnabledStatus, repo)

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "delete user successfully",
			id:   user.ID,
			err:  nil,
		},
		{
			desc: "delete user with invalid id",
			id:   testsutil.GenerateUUID(t),
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "delete user with empty id",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := repo.Delete(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByIDs(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	num := 200

	var items []users.User
	for i := 0; i < num; i++ {
		user := generateUser(t, users.EnabledStatus, repo)
		items = append(items, user)
	}

	page, err := repo.RetrieveAll(context.Background(), users.Page{Offset: 0, Limit: uint64(num)})
	require.Nil(t, err, fmt.Sprintf("retrieve all users unexpected error: %s", err))
	assert.Equal(t, uint64(num), page.Total)

	cases := []struct {
		desc     string
		page     users.Page
		response users.UsersPage
		err      error
	}{
		{
			desc: "successfully",
			page: users.Page{
				Offset: 0,
				Limit:  10,
				IDs:    getIDs(items[0:3]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  3,
					Offset: 0,
					Limit:  10,
				},
				Users: items[0:3],
			},
			err: nil,
		},
		{
			desc: "with empty ids",
			page: users.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{},
			},
			response: users.UsersPage{
				Page: users.Page{
					Offset: 0,
					Limit:  10,
				},
				Users: []users.User(nil),
			},
			err: nil,
		},
		{
			desc: "with offset only",
			page: users.Page{
				Offset: 10,
				IDs:    getIDs(items[0:20]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  20,
					Offset: 10,
					Limit:  0,
				},
				Users: []users.User(nil),
			},
			err: nil,
		},
		{
			desc: "with limit only",
			page: users.Page{
				Limit: 10,
				IDs:   getIDs(items[0:20]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  20,
					Offset: 0,
					Limit:  10,
				},
				Users: items[0:10],
			},
			err: nil,
		},
		{
			desc: "with offset out of range",
			page: users.Page{
				Offset: 1000,
				Limit:  50,
				IDs:    getIDs(items[0:20]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  20,
					Offset: 1000,
					Limit:  50,
				},
				Users: []users.User(nil),
			},
			err: nil,
		},
		{
			desc: "with offset and limit out of range",
			page: users.Page{
				Offset: 15,
				Limit:  10,
				IDs:    getIDs(items[0:20]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  20,
					Offset: 15,
					Limit:  10,
				},
				Users: items[15:20],
			},
			err: nil,
		},
		{
			desc: "with limit out of range",
			page: users.Page{
				Offset: 0,
				Limit:  1000,
				IDs:    getIDs(items[0:20]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  20,
					Offset: 0,
					Limit:  1000,
				},
				Users: items[:20],
			},
			err: nil,
		},
		{
			desc: "with first name",
			page: users.Page{
				Offset:    0,
				Limit:     10,
				FirstName: items[0].FirstName,
				IDs:       getIDs(items[0:20]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Users: []users.User{items[0]},
			},
			err: nil,
		},
		{
			desc: "with metadata",
			page: users.Page{
				Offset:   0,
				Limit:    10,
				Metadata: items[0].Metadata,
				IDs:      getIDs(items[0:20]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Users: []users.User{items[0]},
			},
			err: nil,
		},
		{
			desc: "with invalid metadata",
			page: users.Page{
				Offset: 0,
				Limit:  10,
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				IDs: getIDs(items[0:20]),
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Users: []users.User(nil),
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, c := range cases {
		switch response, err := repo.RetrieveAllByIDs(context.Background(), c.page); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", c.desc, c.err, err))
			assert.Equal(t, c.response.Total, response.Total)
			assert.Equal(t, c.response.Limit, response.Limit)
			assert.Equal(t, c.response.Offset, response.Offset)
			assert.ElementsMatch(t, response.Users, c.response.Users)
		default:
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
		}
	}
}

func TestRetrieveByEmail(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	user := generateUser(t, users.EnabledStatus, repo)

	cases := []struct {
		desc     string
		email    string
		response users.User
		err      error
	}{
		{
			desc:     "successfully",
			email:    user.Email,
			response: user,
			err:      nil,
		},
		{
			desc:     "with invalid user id",
			email:    testsutil.GenerateUUID(t),
			response: users.User{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "with empty user id",
			email:    "",
			response: users.User{},
			err:      repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			usr, err := repo.RetrieveByEmail(context.Background(), c.email)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s got %s\n", c.err, err))
			if err == nil {
				assert.Equal(t, user.ID, usr.ID)
				assert.Equal(t, user.FirstName, usr.FirstName)
				assert.Equal(t, user.LastName, usr.LastName)
				assert.Equal(t, user.Metadata, usr.Metadata)
				assert.Equal(t, user.Email, usr.Email)
				assert.Equal(t, user.Credentials.Username, usr.Credentials.Username)
				assert.Equal(t, user.Status, usr.Status)
			}
		})
	}
}

func TestRetrieveByUsername(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	user := generateUser(t, users.EnabledStatus, repo)

	cases := []struct {
		desc     string
		username string
		response users.User
		err      error
	}{
		{
			desc:     "successfully",
			username: user.Credentials.Username,
			response: user,
			err:      nil,
		},
		{
			desc:     "with invalid user id",
			username: testsutil.GenerateUUID(t),
			response: users.User{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "with empty user id",
			username: "",
			response: users.User{},
			err:      repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			usr, err := repo.RetrieveByUsername(context.Background(), c.username)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s got %s\n", c.err, err))
			if err == nil {
				assert.Equal(t, user.ID, usr.ID)
				assert.Equal(t, user.FirstName, usr.FirstName)
				assert.Equal(t, user.LastName, usr.LastName)
				assert.Equal(t, user.Metadata, usr.Metadata)
				assert.Equal(t, user.Email, usr.Email)
				assert.Equal(t, user.Credentials.Username, usr.Credentials.Username)
				assert.Equal(t, user.Status, usr.Status)
			}
		})
	}
}

func findUsers(usrs []users.User, query string, offset, limit uint64) []users.User {
	rUsers := []users.User{}
	for _, user := range usrs {
		if strings.Contains(user.FirstName, query) {
			rUsers = append(rUsers, user)
		}
	}

	if offset > uint64(len(rUsers)) {
		return []users.User{}
	}

	if limit > uint64(len(rUsers)) {
		return rUsers[offset:]
	}

	return rUsers[offset:limit]
}

func generateUser(t *testing.T, status users.Status, repo users.Repository) users.User {
	usr := users.User{
		ID:        testsutil.GenerateUUID(t),
		FirstName: namesgen.Generate(),
		LastName:  namesgen.Generate(),
		Email:     namesgen.Generate() + emailSuffix,
		Credentials: users.Credentials{
			Username: namesgen.Generate(),
			Secret:   testsutil.GenerateUUID(t),
		},
		Tags: namesgen.GenerateMultiple(5),
		Metadata: users.Metadata{
			"name": namesgen.Generate(),
		},
		Status:    status,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	user, err := repo.Save(context.Background(), usr)
	require.Nil(t, err, fmt.Sprintf("add new user: expected nil got %s\n", err))

	return user
}

func getIDs(usrs []users.User) []string {
	var ids []string
	for _, user := range usrs {
		ids = append(ids, user.ID)
	}

	return ids
}
