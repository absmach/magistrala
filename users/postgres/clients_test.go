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
	"github.com/absmach/magistrala/users/mocks"
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
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	storageClient := new(mocks.Storage)
	repo := cpostgres.NewRepository(database, storageClient)

	uid := testsutil.GenerateUUID(t)

	first_name := namesgen.Generate()
	last_name := namesgen.Generate()
	user_name := namesgen.Generate()

	clientIdentity := first_name + "@example.com"

	cases := []struct {
		desc   string
		client users.User
		err    error
	}{
		{
			desc: "add new user successfully",
			client: users.User{
				ID:        uid,
				FirstName: first_name,
				LastName:  last_name,
				Identity:  clientIdentity,
				Credentials: users.Credentials{
					UserName: user_name,
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         users.EnabledStatus,
				ProfilePicture: "",
			},
			err: nil,
		},
		{
			desc: "add user with duplicate user identity",
			client: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: first_name,
				LastName:  last_name,
				Identity:  clientIdentity,
				Credentials: users.Credentials{
					UserName: namesgen.Generate(),
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         users.EnabledStatus,
				ProfilePicture: "",
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add user with duplicate user name",
			client: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: namesgen.Generate(),
				LastName:  last_name,
				Identity:  namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					UserName: user_name,
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         users.EnabledStatus,
				ProfilePicture: "",
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add user with invalid user id",
			client: users.User{
				ID:        invalidName,
				FirstName: namesgen.Generate(),
				LastName:  namesgen.Generate(),
				Identity:  namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					UserName: user_name,
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         users.EnabledStatus,
				ProfilePicture: "",
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add user with invalid user name",
			client: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: invalidName,
				LastName:  namesgen.Generate(),
				Identity:  namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					UserName: user_name,
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         users.EnabledStatus,
				ProfilePicture: "",
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add user with a missing user name",
			client: users.User{
				ID:       testsutil.GenerateUUID(t),
				Identity: namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					Secret: password,
				},
				Metadata:       users.Metadata{},
				ProfilePicture: "",
			},
			err: nil,
		},
		{
			desc: "add user with a missing user secret",
			client: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: namesgen.Generate(),
				LastName:  namesgen.Generate(),
				Identity:  namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					UserName: namesgen.Generate(),
				},
				Metadata: users.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add a user with invalid metadata",
			client: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: namesgen.Generate(),
				Identity:  namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					UserName: user_name,
					Secret:   password,
				},
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				ProfilePicture: "",
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		rUser, err := repo.Save(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			rUser.Credentials.Secret = tc.client.Credentials.Secret
			assert.Equal(t, tc.client, rUser, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, rUser))
		}
	}
}

func TestIsPlatformAdmin(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	storageClient := new(mocks.Storage)

	repo := cpostgres.NewRepository(database, storageClient)

	cases := []struct {
		desc   string
		client users.User
		err    error
	}{
		{
			desc: "authorize check for super user",
			client: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: namesgen.Generate(),
				Identity:  namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					UserName: namesgen.Generate(),
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
			client: users.User{
				ID:        testsutil.GenerateUUID(t),
				FirstName: namesgen.Generate(),
				Identity:  namesgen.Generate() + "@example.com",
				Credentials: users.Credentials{
					UserName: namesgen.Generate(),
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
		_, err := repo.Save(context.Background(), tc.client)
		require.Nil(t, err, fmt.Sprintf("%s: save user unexpected error: %s", tc.desc, err))
		err = repo.CheckSuperAdmin(context.Background(), tc.client.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	storageClient := new(mocks.Storage)

	repo := cpostgres.NewRepository(database, storageClient)

	client := users.User{
		ID:        testsutil.GenerateUUID(t),
		FirstName: namesgen.Generate(),
		Credentials: users.Credentials{
			UserName: namesgen.Generate(),
			Secret:   password,
		},
		Metadata: users.Metadata{},
		Status:   users.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("failed to save client %s", client.ID))

	cases := []struct {
		desc     string
		clientID string
		err      error
	}{
		{
			desc:     "retrieve existing client",
			clientID: client.ID,
			err:      nil,
		},
		{
			desc:     "retrieve non-existing client",
			clientID: invalidName,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve with empty client id",
			clientID: "",
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.RetrieveByID(context.Background(), tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	storageClient := new(mocks.Storage)

	repo := cpostgres.NewRepository(database, storageClient)

	num := 200
	var items, enabledClients []users.User
	for i := 0; i < num; i++ {
		client := users.User{
			ID:        testsutil.GenerateUUID(t),
			FirstName: namesgen.Generate(),
			Identity:  namesgen.Generate() + "@example.com",
			Credentials: users.Credentials{
				UserName: namesgen.Generate(),
				Secret:   "",
			},
			Metadata: users.Metadata{},
			Status:   users.EnabledStatus,
			Tags:     []string{"tag1"},
			LastName: namesgen.Generate(),
		}
		if i%50 == 0 {
			client.Metadata = map[string]interface{}{
				"key": "value",
			}
			client.Role = users.AdminRole
			client.Status = users.DisabledStatus
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("failed to save client %s", client.ID))
		items = append(items, client)
		if client.Status == users.EnabledStatus {
			enabledClients = append(enabledClients, client)
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
					Total:  196, // No of enabled clients.
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
			desc: "retrieve with invalid client id",
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
			desc: "retrieve with client name",
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
			desc: "retrieve with client User Name",
			pageMeta: users.Page{
				UserName: items[0].Credentials.UserName,
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
				Users: enabledClients,
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
			desc: "retrieve with invalid client name",
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
