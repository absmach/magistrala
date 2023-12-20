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
	cpostgres "github.com/absmach/magistrala/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxNameSize = 254

var (
	invalidName = strings.Repeat("m", maxNameSize+10)
	password    = "$tr0ngPassw0rd"
	namesgen    = namegenerator.NewNameGenerator()
)

func TestClientsSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	uid := testsutil.GenerateUUID(t)

	name := namesgen.Generate()
	clientIdentity := name + "@example.com"

	cases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "add new client successfully",
			client: mgclients.Client{
				ID:   uid,
				Name: name,
				Credentials: mgclients.Credentials{
					Identity: clientIdentity,
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add new client with an owner",
			client: mgclients.Client{
				ID:    testsutil.GenerateUUID(t),
				Owner: uid,
				Name:  namesgen.Generate(),
				Credentials: mgclients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add client with duplicate client identity",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: mgclients.Credentials{
					Identity: clientIdentity,
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: errors.ErrConflict,
		},
		{
			desc: "add client with duplicate client name",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: name,
				Credentials: mgclients.Credentials{
					Identity: clientIdentity,
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: errors.ErrConflict,
		},
		{
			desc: "add client with invalid client id",
			client: mgclients.Client{
				ID:   invalidName,
				Name: namesgen.Generate(),
				Credentials: mgclients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with invalid client name",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: invalidName,
				Credentials: mgclients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with invalid client owner",
			client: mgclients.Client{
				ID:    testsutil.GenerateUUID(t),
				Owner: invalidName,
				Credentials: mgclients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with invalid client identity",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: mgclients.Credentials{
					Identity: invalidName,
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with a missing client name",
			client: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add client with a missing client identity",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: mgclients.Credentials{
					Secret: password,
				},
				Metadata: mgclients.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add client with a missing client secret",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: mgclients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
				},
				Metadata: mgclients.Metadata{},
			},
			err: nil,
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
	repo := cpostgres.NewRepository(database)

	cases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "authorize check for super user",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: mgclients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
				Role:     mgclients.AdminRole,
			},
			err: nil,
		},
		{
			desc: "unauthorize user",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: mgclients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
				Role:     mgclients.UserRole,
			},
			err: errors.ErrAuthorization,
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
	repo := cpostgres.NewRepository(database)

	client := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: namesgen.Generate(),
		Credentials: mgclients.Credentials{
			Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
			Secret:   password,
		},
		Metadata: mgclients.Metadata{},
		Status:   mgclients.EnabledStatus,
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
			err:      errors.ErrNotFound,
		},
		{
			desc:     "retrieve with empty client id",
			clientID: "",
			err:      errors.ErrNotFound,
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

	repo := cpostgres.NewRepository(database)

	ownerID := testsutil.GenerateUUID(t)
	num := 200
	var items []mgclients.Client
	for i := 0; i < num; i++ {
		client := mgclients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: namesgen.Generate(),
			Credentials: mgclients.Credentials{
				Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
				Secret:   "",
			},
			Metadata: mgclients.Metadata{},
			Status:   mgclients.EnabledStatus,
			Tags:     []string{"tag1"},
		}
		if i%50 == 0 {
			client.Owner = ownerID
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("failed to save client %s", client.ID))
		items = append(items, client)
	}

	cases := []struct {
		desc     string
		page     mgclients.Page
		response mgclients.ClientsPage
		err      error
	}{
		{
			desc: "retrieve first page of clients",
			page: mgclients.Page{
				Offset: 0,
				Limit:  50,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 0,
					Limit:  50,
				},
				Clients: items[0:50],
			},
			err: nil,
		},
		{
			desc: "retrieve second page of clients",
			page: mgclients.Page{
				Offset: 50,
				Limit:  200,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 50,
					Limit:  200,
				},
				Clients: items[50:200],
			},
			err: nil,
		},
		{
			desc: "retrieve invitations with limit",
			page: mgclients.Page{
				Offset: 0,
				Limit:  50,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  uint64(num),
					Offset: 0,
					Limit:  50,
				},
				Clients: items[:50],
			},
		},
		{
			desc: "retrieve with offset out of range",
			page: mgclients.Page{
				Offset: 1000,
				Limit:  200,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 1000,
					Limit:  200,
				},
				Clients: []mgclients.Client{},
			},
			err: nil,
		},
		{
			desc: "retrieve with limit out of range",
			page: mgclients.Page{
				Offset: 0,
				Limit:  1000,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 0,
					Limit:  1000,
				},
				Clients: items,
			},
			err: nil,
		},
		{
			desc: "retrieve with empty page",
			page: mgclients.Page{},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{},
			},
			err: nil,
		},
		{
			desc: "retrieve with client id",
			page: mgclients.Page{
				IDs:    []string{items[0].ID},
				Offset: 0,
				Limit:  3,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  3,
				},
				Clients: []mgclients.Client{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve with invalid client id",
			page: mgclients.Page{
				IDs:    []string{invalidName},
				Offset: 0,
				Limit:  3,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  3,
				},
				Clients: []mgclients.Client{},
			},
			err: nil,
		},
		{
			desc: "retrieve with client name",
			page: mgclients.Page{
				Name:   items[0].Name,
				Offset: 0,
				Limit:  3,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  3,
				},
				Clients: []mgclients.Client{items[0]},
			},
			err: nil,
		},
		{
			desc: "retrieve with status",
			page: mgclients.Page{
				Status: mgclients.EnabledStatus,
				Offset: 0,
				Limit:  200,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 0,
					Limit:  200,
				},
				Clients: items,
			},
			err: nil,
		},
		{
			desc: "retrieve by owner id",
			page: mgclients.Page{
				Owner:  ownerID,
				Offset: 0,
				Limit:  5,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  5,
				},
				Clients: []mgclients.Client{items[0], items[50], items[100], items[150]},
			},
			err: nil,
		},
		{
			desc: "retrieve by owner id with invalid owner id",
			page: mgclients.Page{
				Owner:  invalidName,
				Offset: 0,
				Limit:  200,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  200,
				},
				Clients: []mgclients.Client{},
			},
			err: nil,
		},
		{
			desc: "retrieve by tags",
			page: mgclients.Page{
				Offset: 0,
				Limit:  200,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  200,
					Offset: 0,
					Limit:  200,
				},
				Clients: items,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.page)
		assert.Equal(t, tc.response.Total, page.Total, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Total, page.Total))
		assert.Equal(t, tc.response.Offset, page.Offset, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Offset, page.Offset))
		assert.Equal(t, tc.response.Limit, page.Limit, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.response.Limit, page.Limit))
		assert.Equal(t, tc.response.Page, page.Page, fmt.Sprintf("%s: expected  %v, got %v", tc.desc, tc.response, page))
		assert.ElementsMatch(t, tc.response.Clients, page.Clients, fmt.Sprintf("%s: expected  %v, got %v", tc.desc, tc.response.Clients, page.Clients))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateRole(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := cpostgres.NewRepository(database)

	client := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: namesgen.Generate(),
		Credentials: mgclients.Credentials{
			Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
			Secret:   password,
		},
		Metadata: mgclients.Metadata{},
		Status:   mgclients.EnabledStatus,
		Role:     mgclients.UserRole,
	}

	_, err := repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("failed to save client %s", client.ID))

	cases := []struct {
		desc    string
		client  mgclients.Client
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
			client:  mgclients.Client{ID: invalidName},
			newRole: mgclients.AdminRole,
			err:     errors.ErrNotFound,
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
