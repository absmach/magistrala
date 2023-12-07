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

var (
	password       = "$tr0ngPassw0rd"
	clientIdentity = "client-identity@example.com"
	clientName     = "client name"
	wrongName      = "wrong-name"
	wrongID        = "wrong-id"
	namesgen       = namegenerator.NewNameGenerator()
)

func TestClientsRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	client := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: clientName,
		Credentials: mgclients.Credentials{
			Identity: clientIdentity,
			Secret:   password,
		},
		Status: mgclients.EnabledStatus,
	}

	clients, err := repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	client = clients

	cases := map[string]struct {
		ID  string
		err error
	}{
		"retrieve existing client":     {client.ID, nil},
		"retrieve non-existing client": {wrongID, errors.ErrNotFound},
	}

	for desc, tc := range cases {
		cli, err := repo.RetrieveByID(context.Background(), tc.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		if err == nil {
			assert.Equal(t, client.ID, cli.ID, fmt.Sprintf("retrieve client by ID : client ID : expected %s got %s\n", client.ID, cli.ID))
			assert.Equal(t, client.Name, cli.Name, fmt.Sprintf("retrieve client by ID : client Name : expected %s got %s\n", client.Name, cli.Name))
			assert.Equal(t, client.Credentials.Identity, cli.Credentials.Identity, fmt.Sprintf("retrieve client by ID : client Identity : expected %s got %s\n", client.Credentials.Identity, cli.Credentials.Identity))
			assert.Equal(t, client.Status, cli.Status, fmt.Sprintf("retrieve client by ID : client Status : expected %d got %d\n", client.Status, cli.Status))
		}
	}
}

func TestClientsRetrieveByIdentity(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	client := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: clientName,
		Credentials: mgclients.Credentials{
			Identity: clientIdentity,
			Secret:   password,
		},
		Status: mgclients.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		identity string
		err      error
	}{
		"retrieve existing client":     {clientIdentity, nil},
		"retrieve non-existing client": {wrongID, errors.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := repo.RetrieveByIdentity(context.Background(), tc.identity)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestClientsRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	nClients := uint64(200)
	ownerID := testsutil.GenerateUUID(t)

	meta := mgclients.Metadata{
		"admin": true,
	}
	wrongMeta := mgclients.Metadata{
		"admin": false,
	}
	expectedClients := []mgclients.Client{}

	for i := uint64(0); i < nClients; i++ {
		identity := fmt.Sprintf("TestRetrieveAll%d@example.com", i)
		client := mgclients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: identity,
			Credentials: mgclients.Credentials{
				Identity: identity,
				Secret:   password,
			},
			Metadata: mgclients.Metadata{},
			Status:   mgclients.EnabledStatus,
		}
		if i%10 == 0 {
			client.Owner = ownerID
			client.Metadata = meta
			client.Tags = []string{"Test"}
		}
		if i%50 == 0 {
			client.Status = mgclients.DisabledStatus
		}
		client, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		expectedClients = append(expectedClients, client)
	}

	cases := map[string]struct {
		size     uint64
		pm       mgclients.Page
		response []mgclients.Client
	}{
		"retrieve all clients empty page": {
			pm:       mgclients.Page{},
			response: []mgclients.Client{},
			size:     0,
		},
		"retrieve all clients": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: mgclients.AllStatus,
			},
			response: expectedClients,
			size:     200,
		},
		"retrieve all clients with limit": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  50,
				Status: mgclients.AllStatus,
			},
			response: expectedClients[0:50],
			size:     50,
		},
		"retrieve all clients with offset": {
			pm: mgclients.Page{
				Offset: 50,
				Limit:  nClients,
				Status: mgclients.AllStatus,
			},
			response: expectedClients[50:200],
			size:     150,
		},
		"retrieve all clients with limit and offset": {
			pm: mgclients.Page{
				Offset: 50,
				Limit:  50,
				Status: mgclients.AllStatus,
			},
			response: expectedClients[50:100],
			size:     50,
		},
		"retrieve all clients with limit and offset not full": {
			pm: mgclients.Page{
				Offset: 170,
				Limit:  50,
				Status: mgclients.AllStatus,
			},
			response: expectedClients[170:200],
			size:     30,
		},
		"retrieve all clients by metadata": {
			pm: mgclients.Page{
				Offset:   0,
				Limit:    nClients,
				Total:    nClients,
				Metadata: meta,
				Status:   mgclients.AllStatus,
			},
			response: []mgclients.Client{
				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
			},
			size: 20,
		},
		"retrieve clients by wrong metadata": {
			pm: mgclients.Page{
				Offset:   0,
				Limit:    nClients,
				Total:    nClients,
				Metadata: wrongMeta,
				Status:   mgclients.AllStatus,
			},
			response: []mgclients.Client{},
			size:     0,
		},
		"retrieve all clients by name": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Name:   "TestRetrieveAll3@example.com",
				Status: mgclients.AllStatus,
			},
			response: []mgclients.Client{expectedClients[3]},
			size:     1,
		},
		"retrieve clients by wrong name": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Name:   wrongName,
				Status: mgclients.AllStatus,
			},
			response: []mgclients.Client{},
			size:     0,
		},
		"retrieve all clients by owner": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Owner:  ownerID,
				Status: mgclients.AllStatus,
			},
			response: []mgclients.Client{
				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
			},
			size: 20,
		},
		"retrieve clients by wrong owner": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Owner:  wrongID,
				Status: mgclients.AllStatus,
			},
			response: []mgclients.Client{},
			size:     0,
		},
		"retrieve all clients shared by and owned by": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Owner:  ownerID,
				Status: mgclients.AllStatus,
			},
			response: []mgclients.Client{
				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
			},
			size: 20,
		},
		"retrieve all clients by disabled status": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Status: mgclients.DisabledStatus,
			},
			response: []mgclients.Client{expectedClients[0], expectedClients[50], expectedClients[100], expectedClients[150]},
			size:     4,
		},
		"retrieve all clients by combined status": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Status: mgclients.AllStatus,
			},
			response: expectedClients,
			size:     200,
		},
		"retrieve clients by the wrong status": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Status: 10,
			},
			response: []mgclients.Client{},
			size:     0,
		},
		"retrieve all clients by tags": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Tag:    "Test",
				Status: mgclients.AllStatus,
			},
			response: []mgclients.Client{
				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
			},
			size: 20,
		},
		"retrieve clients by wrong tags": {
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Total:  nClients,
				Tag:    "wrongTags",
				Status: mgclients.AllStatus,
			},
			response: []mgclients.Client{},
			size:     0,
		},
	}
	for desc, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.pm)
		size := uint64(len(page.Clients))
		assert.ElementsMatch(t, page.Clients, tc.response, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.response, page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestClientsUpdateMetadata(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	client1 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "enabled-client",
		Credentials: mgclients.Credentials{
			Identity: "client1-update@example.com",
			Secret:   password,
		},
		Metadata: mgclients.Metadata{
			"name": "enabled-client",
		},
		Tags:   []string{"enabled", "tag1"},
		Status: mgclients.EnabledStatus,
	}

	client2 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "disabled-client",
		Credentials: mgclients.Credentials{
			Identity: "client2-update@example.com",
			Secret:   password,
		},
		Metadata: mgclients.Metadata{
			"name": "disabled-client",
		},
		Tags:   []string{"disabled", "tag1"},
		Status: mgclients.DisabledStatus,
	}

	clients1, err := repo.Save(context.Background(), client1)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client with metadata: expected %v got %s\n", nil, err))
	clients2, err := repo.Save(context.Background(), client2)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
	client1 = clients1
	client2 = clients2

	ucases := []struct {
		desc   string
		update string
		client mgclients.Client
		err    error
	}{
		{
			desc:   "update metadata for enabled client",
			update: "metadata",
			client: mgclients.Client{
				ID: client1.ID,
				Metadata: mgclients.Metadata{
					"update": "metadata",
				},
			},
			err: nil,
		},
		{
			desc:   "update metadata for disabled client",
			update: "metadata",
			client: mgclients.Client{
				ID: client2.ID,
				Metadata: mgclients.Metadata{
					"update": "metadata",
				},
			},
			err: errors.ErrNotFound,
		},
		{
			desc:   "update name for enabled client",
			update: "name",
			client: mgclients.Client{
				ID:   client1.ID,
				Name: "updated name",
			},
			err: nil,
		},
		{
			desc:   "update name for disabled client",
			update: "name",
			client: mgclients.Client{
				ID:   client2.ID,
				Name: "updated name",
			},
			err: errors.ErrNotFound,
		},
		{
			desc:   "update name and metadata for enabled client",
			update: "both",
			client: mgclients.Client{
				ID:   client1.ID,
				Name: "updated name and metadata",
				Metadata: mgclients.Metadata{
					"update": "name and metadata",
				},
			},
			err: nil,
		},
		{
			desc:   "update name and metadata for a disabled client",
			update: "both",
			client: mgclients.Client{
				ID:   client2.ID,
				Name: "updated name and metadata",
				Metadata: mgclients.Metadata{
					"update": "name and metadata",
				},
			},
			err: errors.ErrNotFound,
		},
		{
			desc:   "update metadata for invalid client",
			update: "metadata",
			client: mgclients.Client{
				ID: wrongID,
				Metadata: mgclients.Metadata{
					"update": "metadata",
				},
			},
			err: errors.ErrNotFound,
		},
		{
			desc:   "update name for invalid client",
			update: "name",
			client: mgclients.Client{
				ID:   wrongID,
				Name: "updated name",
			},
			err: errors.ErrNotFound,
		},
		{
			desc:   "update name and metadata for invalid client",
			update: "both",
			client: mgclients.Client{
				ID:   client2.ID,
				Name: "updated name and metadata",
				Metadata: mgclients.Metadata{
					"update": "name and metadata",
				},
			},
			err: errors.ErrNotFound,
		},
	}
	for _, tc := range ucases {
		expected, err := repo.Update(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			if tc.client.Name != "" {
				assert.Equal(t, expected.Name, tc.client.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected.Name, tc.client.Name))
			}
			if tc.client.Metadata != nil {
				assert.Equal(t, expected.Metadata, tc.client.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected.Metadata, tc.client.Metadata))
			}
		}
	}
}

func TestClientsUpdateTags(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	client1 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "enabled-client-with-tags",
		Credentials: mgclients.Credentials{
			Identity: "client1-update-tags@example.com",
			Secret:   password,
		},
		Tags:   []string{"test", "enabled"},
		Status: mgclients.EnabledStatus,
	}
	client2 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "disabled-client-with-tags",
		Credentials: mgclients.Credentials{
			Identity: "client2-update-tags@example.com",
			Secret:   password,
		},
		Tags:   []string{"test", "disabled"},
		Status: mgclients.DisabledStatus,
	}

	clients1, err := repo.Save(context.Background(), client1)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client with tags: expected %v got %s\n", nil, err))
	if err == nil {
		assert.Equal(t, client1.ID, client1.ID, fmt.Sprintf("add new client with tags: expected %v got %s\n", nil, err))
	}
	client1 = clients1
	clients2, err := repo.Save(context.Background(), client2)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client with tags: expected %v got %s\n", nil, err))
	if err == nil {
		assert.Equal(t, client2.ID, client2.ID, fmt.Sprintf("add new disabled client with tags: expected %v got %s\n", nil, err))
	}
	client2 = clients2
	ucases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "update tags for enabled client",
			client: mgclients.Client{
				ID:   client1.ID,
				Tags: []string{"updated"},
			},
			err: nil,
		},
		{
			desc: "update tags for disabled client",
			client: mgclients.Client{
				ID:   client2.ID,
				Tags: []string{"updated"},
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update tags for invalid client",
			client: mgclients.Client{
				ID:   wrongID,
				Tags: []string{"updated"},
			},
			err: errors.ErrNotFound,
		},
	}
	for _, tc := range ucases {
		expected, err := repo.UpdateTags(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.client.Tags, expected.Tags, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Tags, expected.Tags))
		}
	}
}

func TestClientsUpdateSecret(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	client1 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "enabled-client",
		Credentials: mgclients.Credentials{
			Identity: "client1-update@example.com",
			Secret:   password,
		},
		Status: mgclients.EnabledStatus,
	}
	client2 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "disabled-client",
		Credentials: mgclients.Credentials{
			Identity: "client2-update@example.com",
			Secret:   password,
		},
		Status: mgclients.DisabledStatus,
	}

	rClients1, err := repo.Save(context.Background(), client1)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
	if err == nil {
		assert.Equal(t, client1.ID, rClients1.ID, fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
	}
	rClients2, err := repo.Save(context.Background(), client2)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
	if err == nil {
		assert.Equal(t, client2.ID, rClients2.ID, fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
	}

	ucases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "update secret for enabled client",
			client: mgclients.Client{
				ID: client1.ID,
				Credentials: mgclients.Credentials{
					Identity: "client1-update@example.com",
					Secret:   "newpassword",
				},
			},
			err: nil,
		},
		{
			desc: "update secret for disabled client",
			client: mgclients.Client{
				ID: client2.ID,
				Credentials: mgclients.Credentials{
					Identity: "client2-update@example.com",
					Secret:   "newpassword",
				},
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update secret for invalid client",
			client: mgclients.Client{
				ID: wrongID,
				Credentials: mgclients.Credentials{
					Identity: "client3-update@example.com",
					Secret:   "newpassword",
				},
			},
			err: errors.ErrNotFound,
		},
	}
	for _, tc := range ucases {
		_, err := repo.UpdateSecret(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			c, err := repo.RetrieveByIdentity(context.Background(), tc.client.Credentials.Identity)
			require.Nil(t, err, fmt.Sprintf("retrieve client by id during update of secret unexpected error: %s", err))
			assert.Equal(t, tc.client.Credentials.Secret, c.Credentials.Secret, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Credentials.Secret, c.Credentials.Secret))
		}
	}
}

func TestClientsUpdateIdentity(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	client1 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "enabled-client",
		Credentials: mgclients.Credentials{
			Identity: "client1-update@example.com",
			Secret:   password,
		},
		Status: mgclients.EnabledStatus,
	}
	client2 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "disabled-client",
		Credentials: mgclients.Credentials{
			Identity: "client2-update@example.com",
			Secret:   password,
		},
		Status: mgclients.DisabledStatus,
	}

	rClients1, err := repo.Save(context.Background(), client1)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
	if err == nil {
		assert.Equal(t, client1.ID, rClients1.ID, fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
	}
	rClients2, err := repo.Save(context.Background(), client2)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
	if err == nil {
		assert.Equal(t, client2.ID, rClients2.ID, fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
	}

	ucases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "update identity for enabled client",
			client: mgclients.Client{
				ID: client1.ID,
				Credentials: mgclients.Credentials{
					Identity: "client1-updated@example.com",
				},
			},
			err: nil,
		},
		{
			desc: "update identity for disabled client",
			client: mgclients.Client{
				ID: client2.ID,
				Credentials: mgclients.Credentials{
					Identity: "client2-updated@example.com",
				},
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update identity for invalid client",
			client: mgclients.Client{
				ID: wrongID,
				Credentials: mgclients.Credentials{
					Identity: "client3-updated@example.com",
				},
			},
			err: errors.ErrNotFound,
		},
	}
	for _, tc := range ucases {
		expected, err := repo.UpdateIdentity(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.client.Credentials.Identity, expected.Credentials.Identity, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Credentials.Identity, expected.Credentials.Identity))
		}
	}
}

func TestClientsUpdateOwner(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	client1 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "enabled-client-with-owner",
		Credentials: mgclients.Credentials{
			Identity: "client1-update-owner@example.com",
			Secret:   password,
		},
		Owner:  testsutil.GenerateUUID(t),
		Status: mgclients.EnabledStatus,
	}
	client2 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "disabled-client-with-owner",
		Credentials: mgclients.Credentials{
			Identity: "client2-update-owner@example.com",
			Secret:   password,
		},
		Owner:  testsutil.GenerateUUID(t),
		Status: mgclients.DisabledStatus,
	}

	clients1, err := repo.Save(context.Background(), client1)
	client1 = clients1
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client with owner: expected %v got %s\n", nil, err))
	if err == nil {
		assert.Equal(t, client1.ID, client1.ID, fmt.Sprintf("add new client with owner: expected %v got %s\n", nil, err))
	}
	clients2, err := repo.Save(context.Background(), client2)
	client2 = clients2
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client with owner: expected %v got %s\n", nil, err))
	if err == nil {
		assert.Equal(t, client2.ID, client2.ID, fmt.Sprintf("add new disabled client with owner: expected %v got %s\n", nil, err))
	}
	ucases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "update owner for enabled client",
			client: mgclients.Client{
				ID:    client1.ID,
				Owner: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "update owner for disabled client",
			client: mgclients.Client{
				ID:    client2.ID,
				Owner: testsutil.GenerateUUID(t),
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update owner for invalid client",
			client: mgclients.Client{
				ID:    wrongID,
				Owner: testsutil.GenerateUUID(t),
			},
			err: errors.ErrNotFound,
		},
	}
	for _, tc := range ucases {
		expected, err := repo.UpdateOwner(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.client.Owner, expected.Owner, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Owner, expected.Owner))
		}
	}
}

func TestClientsChangeStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	client1 := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "enabled-client",
		Credentials: mgclients.Credentials{
			Identity: "client1-update@example.com",
			Secret:   password,
		},
		Status: mgclients.EnabledStatus,
	}

	clients1, err := repo.Save(context.Background(), client1)
	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
	client1 = clients1

	ucases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "change client status for an enabled client",
			client: mgclients.Client{
				ID:     client1.ID,
				Status: 0,
			},
			err: nil,
		},
		{
			desc: "change client status for a disabled client",
			client: mgclients.Client{
				ID:     client1.ID,
				Status: 1,
			},
			err: nil,
		},
		{
			desc: "change client status for non-existing client",
			client: mgclients.Client{
				ID:     "invalid",
				Status: 2,
			},
			err: errors.ErrNotFound,
		},
	}

	for _, tc := range ucases {
		expected, err := repo.ChangeStatus(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.client.Status, expected.Status, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.client.Status, expected.Status))
		}
	}
}

func TestClientsRetrieveAllBasicInfo(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	nusers := 100
	users := make([]mgclients.Client, nusers)

	name := namesgen.Generate()

	for i := 0; i < nusers; i++ {
		username := fmt.Sprintf("%s-%d@example.com", name, i)
		client := mgclients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: username,
			Credentials: mgclients.Credentials{
				Identity: username,
				Secret:   password,
			},
			Metadata: mgclients.Metadata{},
			Status:   mgclients.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("save client unexpected error: %s", err))

		users[i] = mgclients.Client{
			ID:   client.ID,
			Name: client.Name,
		}
	}

	cases := []struct {
		desc     string
		page     mgclients.Page
		response mgclients.ClientsPage
		err      error
	}{
		{
			desc: "retrieve all clients",
			page: mgclients.Page{
				Offset: 0,
				Limit:  uint64(nusers),
			},
			response: mgclients.ClientsPage{
				Clients: users,
				Page: mgclients.Page{
					Total:  uint64(nusers),
					Offset: 0,
					Limit:  uint64(nusers),
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with offset",
			page: mgclients.Page{
				Offset: 10,
				Limit:  uint64(nusers),
			},
			response: mgclients.ClientsPage{
				Clients: users[10:],
				Page: mgclients.Page{
					Total:  uint64(nusers),
					Offset: 10,
					Limit:  uint64(nusers),
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with limit",
			page: mgclients.Page{
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{
				Clients: users[:10],
				Page: mgclients.Page{
					Total:  uint64(nusers),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with offset and limit",
			page: mgclients.Page{
				Offset: 10,
				Limit:  10,
			},
			response: mgclients.ClientsPage{
				Clients: users[10:20],
				Page: mgclients.Page{
					Total:  uint64(nusers),
					Offset: 10,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with name",
			page: mgclients.Page{
				Name:   users[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{
				Clients: findClients(users, users[0].Name[:1], 0, 10),
				Page: mgclients.Page{
					Total:  uint64(nusers),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with name",
			page: mgclients.Page{
				Name:   users[0].Name[:4],
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{
				Clients: findClients(users, users[0].Name[:4], 0, 10),
				Page: mgclients.Page{
					Total:  uint64(nusers),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with name with SQL injection",
			page: mgclients.Page{
				Name:   fmt.Sprintf("%s' OR '1'='1", users[0].Name[:1]),
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client(nil),
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with Identity",
			page: mgclients.Page{
				Identity: users[0].Name[:1],
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{
				Clients: findClients(users, users[0].Name[:1], 0, 10),
				Page: mgclients.Page{
					Total:  uint64(nusers),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with Identity",
			page: mgclients.Page{
				Identity: users[0].Name[:4],
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{
				Clients: findClients(users, users[0].Name[:4], 0, 10),
				Page: mgclients.Page{
					Total:  uint64(nusers),
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with Identity with SQL injection",
			page: mgclients.Page{
				Identity: fmt.Sprintf("%s' OR '1'='1", users[0].Name[:1]),
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client(nil),
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients unknown name",
			page: mgclients.Page{
				Name:   "unknown",
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client(nil),
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients unknown name with SQL injection",
			page: mgclients.Page{
				Name:   "unknown' OR '1'='1",
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client(nil),
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients unknown identity",
			page: mgclients.Page{
				Identity: "unknown",
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client(nil),
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients with order",
			page: mgclients.Page{
				Order:  "name",
				Dir:    "asc",
				Name:   users[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "retrieve all clients with order",
			page: mgclients.Page{
				Order:  "name",
				Dir:    "desc",
				Name:   users[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "retrieve all clients with order",
			page: mgclients.Page{
				Order:    "identity",
				Dir:      "asc",
				Identity: users[0].Name[:1],
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "retrieve all clients with order",
			page: mgclients.Page{
				Order:    "identity",
				Dir:      "desc",
				Identity: users[0].Name[:1],
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{},
			err:      nil,
		},
	}
	for _, tc := range cases {
		resp, err := repo.RetrieveAllBasicInfo(context.Background(), tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			if tc.page.Order != "" && tc.page.Dir != "" {
				tc.response = resp
			}
			assert.Equal(t, tc.response, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, resp))
		}
	}
}

func findClients(clients []mgclients.Client, query string, offset, limit uint64) []mgclients.Client {
	clis := []mgclients.Client{}
	for _, client := range clients {
		if strings.Contains(client.Name, query) {
			clis = append(clis, client)
		}
	}

	if offset > uint64(len(clis)) {
		return []mgclients.Client{}
	}

	if limit > uint64(len(clis)) {
		return clis[offset:]
	}

	return clis[offset:limit]
}
