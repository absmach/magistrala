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
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/users"
	cpostgres "github.com/absmach/magistrala/users/postgres"
	"github.com/absmach/magistrala/users/storage"
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

	storageClient, err := storage.NewStorageClient()
	require.Nil(t, err, fmt.Sprintf("failed to create storage client: %s", err))

	repo := cpostgres.NewRepository(database, storageClient)

	uid := testsutil.GenerateUUID(t)

	name := namesgen.Generate()
	clientIdentity := name + "@example.com"

	cases := []struct {
		desc   string
		client users.User
		err    error
	}{
		{
			desc: "add new client successfully",
			client: users.User{
				ID:   uid,
				Name: name,
				Credentials: users.Credentials{
					Identity: clientIdentity,
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         mgclients.EnabledStatus,
				ProfilePicture: "",
			},
			err: nil,
		},
		{
			desc: "add client with duplicate client identity",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: users.Credentials{
					Identity: clientIdentity,
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         mgclients.EnabledStatus,
				ProfilePicture: "",
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add client with duplicate client name",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: name,
				Credentials: users.Credentials{
					Identity: clientIdentity,
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         mgclients.EnabledStatus,
				ProfilePicture: "",
			},
			err: repoerr.ErrConflict,
		},
		{
			desc: "add client with invalid client id",
			client: users.User{
				ID:   invalidName,
				Name: namesgen.Generate(),
				Credentials: users.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         mgclients.EnabledStatus,
				ProfilePicture: "",
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with invalid client name",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: invalidName,
				Credentials: users.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         mgclients.EnabledStatus,
				ProfilePicture: "",
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with invalid client identity",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: users.Credentials{
					Identity: invalidName,
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				Status:         mgclients.EnabledStatus,
				ProfilePicture: "",
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with a missing client name",
			client: users.User{
				ID: testsutil.GenerateUUID(t),
				Credentials: users.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata:       users.Metadata{},
				ProfilePicture: "",
			},
			err: nil,
		},
		{
			desc: "add client with a missing client identity",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: users.Credentials{
					Secret: password,
				},
				Metadata:       users.Metadata{},
				ProfilePicture: "",
			},
			err: nil,
		},
		{
			desc: "add client with a missing client secret",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: users.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
				},
				Metadata:       users.Metadata{},
				ProfilePicture: "",
			},
			err: nil,
		},
		{
			desc: "add a client with invalid metadata",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: users.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
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
		rClient, err := repo.Save(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			rClient.Credentials.Secret = tc.client.Credentials.Secret
			assert.Equal(t, tc.client, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, rClient))
		}
	}
}

func TestIsPlatformAdmin(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	storageClient, err := storage.NewStorageClient()
	require.Nil(t, err, fmt.Sprintf("failed to create storage client: %s", err))

	repo := cpostgres.NewRepository(database, storageClient)

	cases := []struct {
		desc   string
		client users.User
		err    error
	}{
		{
			desc: "authorize check for super user",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: users.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   mgclients.EnabledStatus,
				Role:     mgclients.AdminRole,
			},
			err: nil,
		},
		{
			desc: "unauthorize user",
			client: users.User{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: users.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: users.Metadata{},
				Status:   mgclients.EnabledStatus,
				Role:     mgclients.UserRole,
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.client)
		require.Nil(t, err, fmt.Sprintf("%s: save client unexpected error: %s", tc.desc, err))
		err = repo.CheckSuperAdmin(context.Background(), tc.client.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	storageClient, err := storage.NewStorageClient()
	require.Nil(t, err, fmt.Sprintf("failed to create storage client: %s", err))

	repo := cpostgres.NewRepository(database, storageClient)

	client := users.User{
		ID:   testsutil.GenerateUUID(t),
		Name: namesgen.Generate(),
		Credentials: users.Credentials{
			Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
			Secret:   password,
		},
		Metadata: users.Metadata{},
		Status:   mgclients.EnabledStatus,
	}

	_, err = repo.Save(context.Background(), client)
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

// fails with listB being empty and listA having 196 users. why
func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	storageClient, err := storage.NewStorageClient()
	require.Nil(t, err, fmt.Sprintf("failed to create storage client: %s", err))

	repo := cpostgres.NewRepository(database, storageClient)

	num := 200
	var items []users.User
	for i := 0; i < num; i++ {
		client := users.User{
			ID:   testsutil.GenerateUUID(t),
			Name: namesgen.Generate(),
			Credentials: users.Credentials{
				Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
				Secret:   "",
			},
			Metadata:  users.Metadata{},
			Status:    mgclients.EnabledStatus,
			Tags:      []string{"tag1"},
			UserName:  namesgen.Generate(),
			FirstName: namesgen.Generate(),
			LastName:  namesgen.Generate(),
		}
		if i%50 == 0 {
			client.Metadata = map[string]interface{}{
				"key": "value",
			}
			client.Role = mgclients.AdminRole
			client.Status = mgclients.DisabledStatus
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("failed to save client %s", client.ID))
		items = append(items, client)
		// if client.Status == mgclients.EnabledStatus {
		// 	enabledClients = append(enabledClients, client)
		// }
	}

	cases := []struct {
		desc     string
		pageMeta mgclients.Page
		page     users.UsersPage
		err      error
	}{
		{
			desc: "retrieve first page of clients",
			pageMeta: mgclients.Page{
				Offset: 0,
				Limit:  50,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 0,
					Limit:  50,
				},
				Users: items[0:50],
			},
			err: nil,
		},
		{
			desc: "retrieve second page of clients",
			pageMeta: mgclients.Page{
				Offset: 50,
				Limit:  200,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 50,
					Limit:  200,
				},
				Users: items[50:200],
			},
			err: nil,
		},
		{
			desc: "retrieve clients with limit",
			pageMeta: mgclients.Page{
				Offset: 0,
				Limit:  50,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  50,
				},
				Users: items[:50],
			},
		},
		{
			desc: "retrieve with offset out of range",
			pageMeta: mgclients.Page{
				Offset: 1000,
				Limit:  200,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
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
			pageMeta: mgclients.Page{
				Offset: 0,
				Limit:  1000,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 0,
					Limit:  1000,
				},
				Users: items,
			},
			err: nil,
		},
		// {
		// 	desc:     "retrieve with empty page",
		// 	pageMeta: mgclients.Page{},
		// 	page: users.UsersPage{
		// 		Page: mgclients.Page{
		// 			Total:  196, // No of enabled clients.
		// 			Offset: 0,
		// 			Limit:  0,
		// 		},
		// 		Users: []users.User{},
		// 	},
		// 	err: nil,
		// },
		{
			desc: "retrieve with client id",
			pageMeta: mgclients.Page{
				IDs:    []string{items[0].ID},
				Offset: 0,
				Limit:  3,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
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
			pageMeta: mgclients.Page{
				IDs:    []string{invalidName},
				Offset: 0,
				Limit:  3,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
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
			pageMeta: mgclients.Page{
				Name:   items[0].Name,
				Offset: 0,
				Limit:  3,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  3,
				},
				Users: []users.User{items[0]},
			},
			err: nil,
		},
		// {
		// add username to pagequery
		// 	desc: "retrieve with client User Name",
		// 	pageMeta: mgclients.Page{
		// 		Name:   items[0].UserName,
		// 		Offset: 0,
		// 		Limit:  3,
		// 		Role:   mgclients.AllRole,
		// 		Status: mgclients.AllStatus,
		// 	},
		// 	page: users.UsersPage{
		// 		Page: mgclients.Page{
		// 			Total:  1,
		// 			Offset: 0,
		// 			Limit:  3,
		// 		},
		// 		Users: []users.User{items[0]},
		// 	},
		// 	err: nil,
		// },
		// {
		// 	desc: "retrieve with enabled status",
		// 	pageMeta: mgclients.Page{
		// 		Status: mgclients.EnabledStatus,
		// 		Offset: 0,
		// 		Limit:  200,
		// 		Role:   mgclients.AllRole,
		// 	},
		// 	page: users.UsersPage{
		// 		Page: mgclients.Page{
		// 			Total:  196,
		// 			Offset: 0,
		// 			Limit:  200,
		// 		},
		// 		Users: enabledClients,
		// 	},
		// 	err: nil,
		// },
		// {
		// 	desc: "retrieve with disabled status",
		// 	pageMeta: mgclients.Page{
		// 		Status: mgclients.DisabledStatus,
		// 		Offset: 0,
		// 		Limit:  200,
		// 		Role:   mgclients.AllRole,
		// 	},
		// 	page: users.UsersPage{
		// 		Page: mgclients.Page{
		// 			Total:  4,
		// 			Offset: 0,
		// 			Limit:  200,
		// 		},
		// 		Users: []users.User{items[0], items[50], items[100], items[150]},
		// 	},
		// },
		{
			desc: "retrieve with all status",
			pageMeta: mgclients.Page{
				Status: mgclients.AllStatus,
				Offset: 0,
				Limit:  200,
				Role:   mgclients.AllRole,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 0,
					Limit:  200,
				},
				Users: items,
			},
		},
		{
			desc: "retrieve by tags",
			pageMeta: mgclients.Page{
				Tag:    "tag1",
				Offset: 0,
				Limit:  200,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
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
			pageMeta: mgclients.Page{
				Name:   invalidName,
				Offset: 0,
				Limit:  3,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  3,
				},
				Users: []users.User{},
			},
		},
		{
			desc: "retrieve with metadata",
			pageMeta: mgclients.Page{
				Metadata: map[string]interface{}{
					"key": "value",
				},
				Offset: 0,
				Limit:  200,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
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
			pageMeta: mgclients.Page{
				Metadata: map[string]interface{}{
					"key": "value1",
				},
				Offset: 0,
				Limit:  200,
				Role:   mgclients.AllRole,
				Status: mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  200,
				},
				Users: []users.User{},
			},
			err: nil,
		},
		// {
		// 	desc: "retrieve with role",
		// 	pageMeta: mgclients.Page{
		// 		Role:   mgclients.AdminRole,
		// 		Offset: 0,
		// 		Limit:  200,
		// 		Status: mgclients.AllStatus,
		// 	},
		// 	page: users.UsersPage{
		// 		Page: mgclients.Page{
		// 			Total:  4,
		// 			Offset: 0,
		// 			Limit:  200,
		// 		},
		// 		Users: []users.User{items[0], items[50], items[100], items[150]},
		// 	},
		// 	err: nil,
		// },
		// {
		// 	desc: "retrieve with invalid role",
		// 	pageMeta: mgclients.Page{
		// 		Role:   mgclients.AdminRole + 2,
		// 		Offset: 0,
		// 		Limit:  200,
		// 		Status: mgclients.AllStatus,
		// 	},
		// 	page: users.UsersPage{
		// 		Page: mgclients.Page{
		// 			Total:  0,
		// 			Offset: 0,
		// 			Limit:  200,
		// 		},
		// 		Users: []users.User{},
		// 	},
		// 	err: nil,
		// },
		{
			desc: "retrieve with identity",
			pageMeta: mgclients.Page{
				Identity: items[0].Credentials.Identity,
				Offset:   0,
				Limit:    3,
				Role:     mgclients.AllRole,
				Status:   mgclients.AllStatus,
			},
			page: users.UsersPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  3,
				},
				Users: []users.User{items[0]},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		// fmt.Printf("Expected users: %v\n", tc.page.Users)

		page, err := repo.RetrieveAll(context.Background(), tc.pageMeta)

		// fmt.Printf("Actual users: %v\n", page.Users)
		assert.Equal(t, tc.page.Total, page.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Total, page.Total))
		assert.Equal(t, tc.page.Offset, page.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Offset, page.Offset))
		assert.Equal(t, tc.page.Limit, page.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.page.Limit, page.Limit))
		assert.Equal(t, tc.page.Page, page.Page, fmt.Sprintf("%s: expected  %v, got %v", tc.desc, tc.page, page))
		assert.ElementsMatch(t, tc.page.Users, page.Users, fmt.Sprintf("%s: expected %v, got %v", tc.desc, tc.page.Users, page.Users))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateRole(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	storageClient, err := storage.NewStorageClient()
	require.Nil(t, err, fmt.Sprintf("failed to create storage client: %s", err))

	repo := cpostgres.NewRepository(database, storageClient)

	client := users.User{
		ID:   testsutil.GenerateUUID(t),
		Name: namesgen.Generate(),
		Credentials: users.Credentials{
			Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
			Secret:   password,
		},
		Metadata: users.Metadata{},
		Status:   mgclients.EnabledStatus,
		Role:     mgclients.UserRole,
	}

	_, err = repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("failed to save client %s", client.ID))

	cases := []struct {
		desc    string
		client  users.User
		newRole mgclients.Role
		err     error
	}{
		{
			desc:    "update role to admin",
			client:  client,
			newRole: mgclients.AdminRole,
			err:     nil,
		},
		{
			desc:    "update role to user",
			client:  client,
			newRole: mgclients.UserRole,
			err:     nil,
		},
		{
			desc:    "update role with invalid client id",
			client:  users.User{ID: invalidName},
			newRole: mgclients.AdminRole,
			err:     repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		tc.client.Role = tc.newRole
		client, err := repo.UpdateRole(context.Background(), tc.client)
		if err != nil {
			assert.Equal(t, err, tc.err, fmt.Sprintf("%s: expected error %v, got %v", tc.desc, tc.err, err))
		} else {
			assert.Equal(t, tc.newRole, client.Role, fmt.Sprintf("%s: expected role %v, got %v", tc.desc, tc.newRole, client.Role))
		}
	}
}
