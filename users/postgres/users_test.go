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
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/users"
	cpostgres "github.com/absmach/supermq/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxNameSize = 254
	defOrder    = "created_at"
	defDir      = "asc"
)

var (
	invalidName    = strings.Repeat("m", maxNameSize+10)
	password       = "$tr0ngPassw0rd"
	namesgen       = namegenerator.NewGenerator()
	emailSuffix    = "@example.com"
	validTimestamp = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	ascDir         = "asc"
	descDir        = "desc"
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

	externalUser := users.User{
		ID:              testsutil.GenerateUUID(t),
		FirstName:       namesgen.Generate(),
		LastName:        namesgen.Generate(),
		PrivateMetadata: users.Metadata{},
		Metadata:        users.Metadata{},
		Credentials: users.Credentials{
			Username: namesgen.Generate(),
		},
		Email:        namesgen.Generate() + "@example.com",
		AuthProvider: "external",
	}
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
				PrivateMetadata: users.Metadata{
					"organization": namesgen.Generate(),
				},
				Metadata: users.Metadata{
					"address": namesgen.Generate(),
				},
				Status: users.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add new external user successfully",
			user: externalUser,
			err:  nil,
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
				PrivateMetadata: users.Metadata{
					"organization": namesgen.Generate(),
				},
				Metadata: users.Metadata{
					"address": namesgen.Generate(),
				},
				Status: users.EnabledStatus,
			},
			err: errors.ErrEmailAlreadyExists,
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
				PrivateMetadata: users.Metadata{
					"organization": namesgen.Generate(),
				},
				Metadata: users.Metadata{
					"address": namesgen.Generate(),
				},
				Status: users.EnabledStatus,
			},
			err: errors.ErrUsernameNotAvailable,
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
				PrivateMetadata: users.Metadata{
					"organization": namesgen.Generate(),
				},
				Metadata: users.Metadata{
					"address": namesgen.Generate(),
				},
				Status: users.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
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
				PrivateMetadata: users.Metadata{
					"organization": namesgen.Generate(),
				},
				Metadata: users.Metadata{
					"address": namesgen.Generate(),
				},
				Status: users.EnabledStatus,
			},
			err: repoerr.ErrCreateEntity,
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
				PrivateMetadata: users.Metadata{
					"organization": namesgen.Generate(),
				},
				Metadata: users.Metadata{
					"address": namesgen.Generate(),
				},
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
				PrivateMetadata: users.Metadata{
					"organization": namesgen.Generate(),
				},
				Metadata: users.Metadata{
					"address": namesgen.Generate(),
				},
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
				PrivateMetadata: map[string]any{
					"key": make(chan int),
				},
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rUser, err := repo.Save(context.Background(), tc.user)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				rUser.Credentials.Secret = tc.user.Credentials.Secret
				assert.Equal(t, tc.user, rUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.user, rUser))
			}
		})
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
				PrivateMetadata: users.Metadata{},
				Metadata:        users.Metadata{},
				Status:          users.EnabledStatus,
				Role:            users.AdminRole,
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
				PrivateMetadata: users.Metadata{},
				Metadata:        users.Metadata{},
				Status:          users.EnabledStatus,
				Role:            users.UserRole,
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
		PrivateMetadata: users.Metadata{
			"organization": namesgen.Generate(),
		},
		Metadata: users.Metadata{
			"address": namesgen.Generate(),
		},
		Status: users.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), user)
	require.Nil(t, err, fmt.Sprintf("failed to save users %s", user.ID))

	externalUser := users.User{
		ID:              testsutil.GenerateUUID(t),
		FirstName:       namesgen.Generate(),
		LastName:        namesgen.Generate(),
		PrivateMetadata: users.Metadata{},
		Metadata:        users.Metadata{},
		Credentials: users.Credentials{
			Username: namesgen.Generate(),
		},
		Email:        namesgen.Generate() + "@example.com",
		AuthProvider: "external",
	}

	_, err = repo.Save(context.Background(), externalUser)
	require.Nil(t, err, fmt.Sprintf("failed to save users %s", user.ID))

	cases := []struct {
		desc   string
		userID string
		user   users.User
		err    error
	}{
		{
			desc:   "retrieve existing user",
			userID: user.ID,
			user:   user,
			err:    nil,
		},

		{
			desc:   "retrieve existing oauth user",
			userID: externalUser.ID,
			user:   externalUser,
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
		rUser, err := repo.RetrieveByID(context.Background(), tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.user, rUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.user, rUser))
		}
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
	baseTime := time.Now().UTC().Truncate(time.Millisecond)
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
			Metadata:  users.Metadata{},
			Status:    users.EnabledStatus,
			Tags:      []string{"tag1"},
			CreatedAt: baseTime.Add(time.Duration(i) * time.Millisecond),
			UpdatedAt: baseTime.Add(time.Duration(i) * time.Millisecond),
		}
		if i%50 == 0 {
			user.Metadata = map[string]any{
				"key": "value",
			}
			user.Role = users.AdminRole
			user.Status = users.DisabledStatus
		}
		if i%99 == 0 {
			user.Tags = []string{"tag1", "tag2"}
		}
		_, err := repo.Save(context.Background(), user)
		require.Nil(t, err, fmt.Sprintf("failed to save user %s", user.ID))
		items = append(items, user)
		if user.Status == users.EnabledStatus {
			enabledUsers = append(enabledUsers, user)
		}
	}

	reversedUsers := []users.User{}
	for i := len(items) - 1; i >= 0; i-- {
		reversedUsers = append(reversedUsers, items[i])
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
				Limit:  1,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "created_at",
				Dir:    ascDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  200,
					Offset: 0,
					Limit:  1,
				},
				Users: items[0:1],
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order:     "created_at",
				Dir:       ascDir,
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
				Order:    "created_at",
				Dir:      ascDir,
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order:  "created_at",
				Dir:    ascDir,
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
			desc: "retrieve by tags with OR operator",
			pageMeta: users.Page{
				Tags:   users.TagsQuery{Operator: users.OrOp, Elements: []string{"tag1"}},
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
			desc: "retrieve by tags with OR operator no match",
			pageMeta: users.Page{
				Tags:   users.TagsQuery{Operator: users.OrOp, Elements: []string{"non-existing-tag"}},
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
			desc: "retrieve by tags with AND operator",
			pageMeta: users.Page{
				Tags:   users.TagsQuery{Operator: users.AndOp, Elements: []string{"tag1", "tag2"}},
				Offset: 0,
				Limit:  200,
				Role:   users.AllRole,
				Status: users.AllStatus,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  3,
					Offset: 0,
					Limit:  200,
				},
				Users: []users.User{items[0], items[99], items[198]},
			},
			err: nil,
		},
		{
			desc: "retrieve by tags with AND operator no match",
			pageMeta: users.Page{
				Tags:   users.TagsQuery{Operator: users.AndOp, Elements: []string{"tag1", "non-existing-tag"}},
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
				Metadata: map[string]any{
					"key": "value",
				},
				Offset: 0,
				Limit:  200,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "created_at",
				Dir:    ascDir,
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
				Metadata: map[string]any{
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
				Order:  "created_at",
				Dir:    ascDir,
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
		{
			desc: "retrieve users with order by first_name ascending",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "first_name",
				Dir:    ascDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve users with order by first_name descending",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "first_name",
				Dir:    descDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve users with order by username ascending",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "username",
				Dir:    ascDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve users with order by username descending",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "username",
				Dir:    descDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve users with order by created_at ascending",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "created_at",
				Dir:    ascDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Users: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve users with order by created_at descending",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "created_at",
				Dir:    descDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Users: reversedUsers[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve users with order by updated_at ascending",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "updated_at",
				Dir:    ascDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Users: items[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve users with order by updated_at descending",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Role:   users.AllRole,
				Status: users.AllStatus,
				Order:  "updated_at",
				Dir:    descDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  10,
				},
				Users: reversedUsers[:10],
			},
			err: nil,
		},
		{
			desc: "retrieve users created from specific time",
			pageMeta: users.Page{
				CreatedFrom: baseTime.Add(50 * time.Millisecond),
				Offset:      0,
				Limit:       200,
				Role:        users.AllRole,
				Status:      users.AllStatus,
				Order:       "created_at",
				Dir:         ascDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  150,
					Offset: 0,
					Limit:  200,
				},
				Users: items[50:200],
			},
			err: nil,
		},
		{
			desc: "retrieve users created to specific time",
			pageMeta: users.Page{
				CreatedTo: baseTime.Add(49 * time.Millisecond),
				Offset:    0,
				Limit:     200,
				Role:      users.AllRole,
				Status:    users.AllStatus,
				Order:     "created_at",
				Dir:       ascDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  50,
					Offset: 0,
					Limit:  200,
				},
				Users: items[0:50],
			},
			err: nil,
		},
		{
			desc: "retrieve users created within time range",
			pageMeta: users.Page{
				CreatedFrom: baseTime.Add(50 * time.Millisecond),
				CreatedTo:   baseTime.Add(99 * time.Millisecond),
				Offset:      0,
				Limit:       200,
				Role:        users.AllRole,
				Status:      users.AllStatus,
				Order:       "created_at",
				Dir:         ascDir,
			},
			page: users.UsersPage{
				Page: users.Page{
					Total:  50,
					Offset: 0,
					Limit:  200,
				},
				Users: items[50:100],
			},
			err: nil,
		},
		{
			desc: "retrieve users with time range outside of all records",
			pageMeta: users.Page{
				CreatedFrom: baseTime.Add(300 * time.Millisecond),
				CreatedTo:   baseTime.Add(400 * time.Millisecond),
				Offset:      0,
				Limit:       200,
				Role:        users.AllRole,
				Status:      users.AllStatus,
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
		if len(tc.page.Users) > 0 {
			assert.ElementsMatch(t, stripUserDetails(tc.page.Users), stripUserDetails(page.Users), fmt.Sprintf("%s: expected %v, got %v", tc.desc, tc.page.Users, page.Users))
		}
		verifyUsersOrdering(t, page.Users, tc.pageMeta.Order, tc.pageMeta.Dir)
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
	baseTime := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < int(nUsers); i++ {
		user := generateUserWithTime(t, users.EnabledStatus, repo, baseTime.Add(time.Duration(i)*time.Millisecond))

		expectedUsers = append(expectedUsers, users.User{
			ID:        user.ID,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Credentials: users.Credentials{
				Username: user.Credentials.Username,
			},
			Metadata:  user.Metadata,
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
				Dir:   ascDir,
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
				Limit:  10,
				Order:  defOrder,
				Dir:    defDir,
			},
			response: users.UsersPage{
				Page: users.Page{
					Total:  nUsers,
					Offset: 0,
					Limit:  10,
				},
				Users: expectedUsers[:10],
			},
		},
		{
			desc: "with offset and limit",
			page: users.Page{
				Offset: 10,
				Limit:  10,
				Order:  "name",
				Dir:    ascDir,
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
				Dir:    ascDir,
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
				Dir:       ascDir,
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
				Dir:    ascDir,
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
				Dir:       ascDir,
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
				Dir:       descDir,
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
				Dir:      ascDir,
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
				Dir:      ascDir,
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
				assert.ElementsMatch(t, response.Users, c.response.Users, fmt.Sprintf("expected %v got %v\n", c.response.Users, response.Users))
			default:
				assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			}
		})
	}
}

func TestUpdateRole(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)
	user1 := generateUser(t, users.EnabledStatus, repo)
	user2 := generateUser(t, users.DisabledStatus, repo)
	adminRole := users.AdminRole
	userRole := users.UserRole

	cases := []struct {
		desc    string
		update  string
		userID  string
		userReq users.User
		err     error
	}{
		{
			desc: "update role of user to admin",
			userReq: users.User{
				ID:   user1.ID,
				Role: adminRole,
			},
			err: nil,
		},
		{
			desc: "update role of admin to user",
			userReq: users.User{
				ID:   user1.ID,
				Role: userRole,
			},
			err: nil,
		},
		{
			desc: "update role for disabled user",
			userReq: users.User{
				ID:   user2.ID,
				Role: adminRole,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update role for invalid user",
			userReq: users.User{
				ID:   testsutil.GenerateUUID(t),
				Role: adminRole,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			updatedAt := time.Now().UTC().Truncate(time.Microsecond)
			updatedBy := testsutil.GenerateUUID(t)
			c.userReq.UpdatedAt = updatedAt
			c.userReq.UpdatedBy = updatedBy
			expected, err := repo.UpdateRole(context.Background(), c.userReq)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.userReq.Role, expected.Role)
				assert.Equal(t, c.userReq.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.userReq.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestUpdateEmail(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM users")
		require.Nil(t, err, fmt.Sprintf("clean users unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	user1 := generateUser(t, users.EnabledStatus, repo)
	user2 := generateUser(t, users.DisabledStatus, repo)
	user3 := generateUser(t, users.EnabledStatus, repo)

	updatedEmail := namesgen.Generate() + emailSuffix
	emptyName := ""

	cases := []struct {
		desc    string
		update  string
		userReq users.User
		err     error
	}{
		{
			desc: "update email for enabled user",
			userReq: users.User{
				ID:    user1.ID,
				Email: updatedEmail,
			},

			err: nil,
		},
		{
			desc: "update empty email for enabled user",
			userReq: users.User{
				ID:    user3.ID,
				Email: emptyName,
			},
			err: nil,
		},
		{
			desc: "update email for disabled user",
			userReq: users.User{
				ID:    user2.ID,
				Email: updatedEmail,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update email for invalid user",
			userReq: users.User{
				ID:    testsutil.GenerateUUID(t),
				Email: updatedEmail,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			updatedAt := time.Now().UTC().Truncate(time.Microsecond)
			updatedBy := testsutil.GenerateUUID(t)
			c.userReq.UpdatedAt = updatedAt
			c.userReq.UpdatedBy = updatedBy
			expected, err := repo.UpdateEmail(context.Background(), c.userReq)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.userReq.Email, expected.Email)
				assert.Equal(t, c.userReq.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.userReq.UpdatedBy, expected.UpdatedBy)
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
	user3 := generateUser(t, users.EnabledStatus, repo)

	updatedMetadata := users.Metadata{"update": namesgen.Generate()}
	malformedMetadata := users.Metadata{"update": make(chan int)}
	updatedLastName := namesgen.Generate()
	updatedFirstName := namesgen.Generate()
	updateTags := namesgen.GenerateMultiple(5)
	updatedProfilePicture := namesgen.Generate()
	emptyName := ""
	emptyTags := []string{}

	cases := []struct {
		desc    string
		update  string
		userID  string
		userReq users.UserReq
		userRes users.User
		err     error
	}{
		{
			desc:   "update metadata for enabled user",
			update: "metadata",
			userID: user1.ID,
			userReq: users.UserReq{
				Metadata: &updatedMetadata,
			},
			userRes: users.User{
				Metadata: updatedMetadata,
			},
			err: nil,
		},
		{
			desc:   "update private metadata for enabled user",
			update: "private_metadata",
			userID: user1.ID,
			userReq: users.UserReq{
				PrivateMetadata: &updatedMetadata,
			},
			userRes: users.User{
				PrivateMetadata: updatedMetadata,
			},
			err: nil,
		},
		{
			desc:   "update malformed private metadata for enabled user",
			update: "private_metadata",
			userID: user1.ID,
			userReq: users.UserReq{
				PrivateMetadata: &malformedMetadata,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc:   "update empty metadata for enabled user",
			update: "metadata",
			userID: user3.ID,
			userReq: users.UserReq{
				Metadata: &users.Metadata{},
			},
			userRes: users.User{
				Metadata: users.Metadata{},
			},
			err: nil,
		},
		{
			desc:   "update metadata for disabled user",
			update: "metadata",
			userID: user2.ID,
			userReq: users.UserReq{
				Metadata: &updatedMetadata,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update first name for enabled user",
			update: "first_name",
			userID: user1.ID,
			userReq: users.UserReq{
				FirstName: &updatedFirstName,
			},
			userRes: users.User{
				FirstName: updatedFirstName,
			},
			err: nil,
		},
		{
			desc:   "update empty first name for enabled user",
			update: "first_name",
			userID: user3.ID,
			userReq: users.UserReq{
				FirstName: &emptyName,
			},
			userRes: user3,
			err:     nil,
		},
		{
			desc:   "update first name for disabled user",
			update: "first_name",
			userID: user2.ID,
			userReq: users.UserReq{
				FirstName: &updatedFirstName,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update private metadata for invalid user",
			update: "private_metadata",
			userID: testsutil.GenerateUUID(t),
			userReq: users.UserReq{
				PrivateMetadata: &updatedMetadata,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update first name for empty user",
			update: "first_name",
			userID: "",
			userReq: users.UserReq{
				FirstName: &updatedFirstName,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update last name for enabled user",
			update: "last_name",
			userID: user1.ID,
			userReq: users.UserReq{
				LastName: &updatedLastName,
			},
			userRes: users.User{
				LastName: updatedLastName,
			},
			err: nil,
		},
		{
			desc:   "update empty last name for enabled user",
			update: "last_name",
			userID: user3.ID,
			userReq: users.UserReq{
				LastName: &emptyName,
			},
			userRes: user3,
			err:     nil,
		},
		{
			desc:   "update last name for disabled user",
			update: "last_name",
			userID: user2.ID,
			userReq: users.UserReq{
				LastName: &updatedLastName,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update last name for invalid user",
			update: "last_name",
			userID: testsutil.GenerateUUID(t),
			userReq: users.UserReq{
				LastName: &updatedLastName,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update tags for enabled user",
			userID: user1.ID,
			userReq: users.UserReq{
				Tags: &updateTags,
			},
			userRes: users.User{
				Tags: updateTags,
			},
			err: nil,
		},
		{
			desc:   "update empty tags for enabled user",
			userID: user3.ID,
			userReq: users.UserReq{
				Tags: &emptyTags,
			},
			userRes: users.User{
				Tags: []string{},
			},
			err: nil,
		},
		{
			desc:   "update tags for disabled user",
			userID: user2.ID,
			userReq: users.UserReq{
				Tags: &updateTags,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update tags for invalid user",
			userID: testsutil.GenerateUUID(t),
			userReq: users.UserReq{
				Tags: &updateTags,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update profile picture for enabled user",
			userID: user1.ID,
			userReq: users.UserReq{
				ProfilePicture: &updatedProfilePicture,
			},
			userRes: users.User{
				ProfilePicture: updatedProfilePicture,
			},
			err: nil,
		},
		{
			desc:   "update empty profile picture for enabled user",
			userID: user3.ID,
			userReq: users.UserReq{
				ProfilePicture: &emptyName,
			},
			userRes: users.User{
				ProfilePicture: emptyName,
			},
			err: nil,
		},
		{
			desc:   "update profile picture for disabled user",
			userID: user2.ID,
			userReq: users.UserReq{
				ProfilePicture: &updatedProfilePicture,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update profile picture for invalid user",
			userID: testsutil.GenerateUUID(t),
			userReq: users.UserReq{
				ProfilePicture: &updatedProfilePicture,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			updatedAt := time.Now().UTC().Truncate(time.Microsecond)
			updatedBy := testsutil.GenerateUUID(t)
			c.userReq.UpdatedAt = &updatedAt
			c.userReq.UpdatedBy = &updatedBy
			c.userRes.UpdatedAt = updatedAt
			c.userRes.UpdatedBy = updatedBy
			expected, err := repo.Update(context.Background(), c.userID, c.userReq)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				switch c.update {
				case "private_metadata":
					assert.Equal(t, c.userRes.PrivateMetadata, expected.PrivateMetadata)
				case "metadata":
					assert.Equal(t, c.userRes.Metadata, expected.Metadata)
				case "first_name":
					assert.Equal(t, c.userRes.FirstName, expected.FirstName)
				case "last_name":
					assert.Equal(t, c.userRes.LastName, expected.LastName)
				case "tags":
					assert.Equal(t, c.userRes.Tags, expected.Tags)
				case "profile_picture":
					assert.Equal(t, c.userRes.ProfilePicture, expected.ProfilePicture)
				case "role":
					assert.Equal(t, c.userRes.Role, expected.Role)
				case "email":
					assert.Equal(t, c.userRes.Email, expected.Email)
				}
				assert.Equal(t, c.userRes.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.userRes.UpdatedBy, expected.UpdatedBy)
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
			err: errors.ErrUsernameNotAvailable,
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
			c.user.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
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
			c.user.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
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
			c.user.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
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
	baseTime := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < num; i++ {
		user := generateUserWithTime(t, users.EnabledStatus, repo, baseTime.Add(time.Duration(i)*time.Millisecond))
		user.PrivateMetadata = nil
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order: "created_at",
				Dir:   ascDir,
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
				Order:  "created_at",
				Dir:    ascDir,
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
				Order:     "created_at",
				Dir:       ascDir,
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
				Metadata: map[string]any{
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
			err: repoerr.ErrViewEntity,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
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
		})
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
				assert.Equal(t, user.PrivateMetadata, usr.PrivateMetadata)
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
				assert.Equal(t, user.PrivateMetadata, usr.PrivateMetadata)
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
	return generateUserWithTime(t, status, repo, time.Now().UTC().Truncate(time.Millisecond))
}

func generateUserWithTime(t *testing.T, status users.Status, repo users.Repository, createdAt time.Time) users.User {
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
		PrivateMetadata: users.Metadata{
			"organization": namesgen.Generate(),
		},
		Metadata: users.Metadata{
			"address": namesgen.Generate(),
		},
		Status:    status,
		CreatedAt: createdAt,
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

func stripUserDetails(users []users.User) []users.User {
	for i := range users {
		users[i].CreatedAt = validTimestamp
		users[i].UpdatedAt = validTimestamp
	}
	return users
}

func verifyUsersOrdering(t *testing.T, users []users.User, order, dir string) {
	if order == "" || len(users) <= 1 {
		return
	}

	for i := 0; i < len(users)-1; i++ {
		switch order {
		case "first_name":
			if dir == ascDir {
				assert.LessOrEqual(t, users[i].FirstName, users[i+1].FirstName, fmt.Sprintf("Users not ordered by first_name ascending at index %d: %s > %s", i, users[i].FirstName, users[i+1].FirstName))
				continue
			}
			assert.GreaterOrEqual(t, users[i].FirstName, users[i+1].FirstName, fmt.Sprintf("Users not ordered by first_name descending at index %d: %s < %s", i, users[i].FirstName, users[i+1].FirstName))
		case "username":
			if dir == ascDir {
				assert.LessOrEqual(t, users[i].Credentials.Username, users[i+1].Credentials.Username, fmt.Sprintf("Users not ordered by username ascending at index %d: %s > %s", i, users[i].Credentials.Username, users[i+1].Credentials.Username))
				continue
			}
			assert.GreaterOrEqual(t, users[i].Credentials.Username, users[i+1].Credentials.Username, fmt.Sprintf("Users not ordered by username descending at index %d: %s < %s", i, users[i].Credentials.Username, users[i+1].Credentials.Username))
		case "created_at":
			if dir == ascDir {
				assert.False(t, users[i].CreatedAt.After(users[i+1].CreatedAt), fmt.Sprintf("Users not ordered by created_at ascending at index %d: %v > %v", i, users[i].CreatedAt, users[i+1].CreatedAt))
				continue
			}
			assert.False(t, users[i].CreatedAt.Before(users[i+1].CreatedAt), fmt.Sprintf("Users not ordered by created_at descending at index %d: %v < %v", i, users[i].CreatedAt, users[i+1].CreatedAt))
		case "updated_at":
			if dir == ascDir {
				assert.False(t, users[i].UpdatedAt.After(users[i+1].UpdatedAt), fmt.Sprintf("Users not ordered by updated_at ascending at index %d: %v > %v", i, users[i].UpdatedAt, users[i+1].UpdatedAt))
				continue
			}
			assert.False(t, users[i].UpdatedAt.Before(users[i+1].UpdatedAt), fmt.Sprintf("Users not ordered by updated_at descending at index %d: %v < %v", i, users[i].UpdatedAt, users[i+1].UpdatedAt))
		}
	}
}
