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
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxNameSize = 1024

var (
	invalidName     = strings.Repeat("m", maxNameSize+10)
	clientIdentity  = "client-identity@example.com"
	clientName      = "client name"
	invalidClientID = "invalidClientID"
	namesgen        = namegenerator.NewNameGenerator()
)

func TestClientsSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	uid := testsutil.GenerateUUID(t)

	cases := []struct {
		desc   string
		client clients.Client
		err    error
	}{
		{
			desc: "add new client successfully",
			client: clients.Client{
				ID:   uid,
				Name: clientName,
				Credentials: clients.Credentials{
					Identity: clientIdentity,
					Secret:   testsutil.GenerateUUID(t),
				},
				Metadata: clients.Metadata{},
				Status:   clients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add new client with an owner",
			client: clients.Client{
				ID:    testsutil.GenerateUUID(t),
				Owner: uid,
				Name:  clientName,
				Credentials: clients.Credentials{
					Identity: "withowner-client@example.com",
					Secret:   testsutil.GenerateUUID(t),
				},
				Metadata: clients.Metadata{},
				Status:   clients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add client with invalid client id",
			client: clients.Client{
				ID:   invalidName,
				Name: clientName,
				Credentials: clients.Credentials{
					Identity: "invalidid-client@example.com",
					Secret:   testsutil.GenerateUUID(t),
				},
				Metadata: clients.Metadata{},
				Status:   clients.EnabledStatus,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client name",
			client: clients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: invalidName,
				Credentials: clients.Credentials{
					Identity: "invalidname-client@example.com",
					Secret:   testsutil.GenerateUUID(t),
				},
				Metadata: clients.Metadata{},
				Status:   clients.EnabledStatus,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client owner",
			client: clients.Client{
				ID:    testsutil.GenerateUUID(t),
				Owner: invalidName,
				Credentials: clients.Credentials{
					Identity: "invalidowner-client@example.com",
					Secret:   testsutil.GenerateUUID(t),
				},
				Metadata: clients.Metadata{},
				Status:   clients.EnabledStatus,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client identity",
			client: clients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: clientName,
				Credentials: clients.Credentials{
					Identity: invalidName,
					Secret:   testsutil.GenerateUUID(t),
				},
				Metadata: clients.Metadata{},
				Status:   clients.EnabledStatus,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "add client with a missing client identity",
			client: clients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: clients.Credentials{
					Identity: "",
					Secret:   testsutil.GenerateUUID(t),
				},
				Metadata: clients.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add client with a missing client secret",
			client: clients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: clients.Credentials{
					Identity: "missing-client-secret@example.com",
					Secret:   "",
				},
				Metadata: clients.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add a client with invalid metadata",
			client: clients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namesgen.Generate(),
				Credentials: clients.Credentials{
					Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
					Secret:   testsutil.GenerateUUID(t),
				},
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
			},
			err: errors.ErrMalformedEntity,
		},
	}
	for _, tc := range cases {
		rClient, err := repo.Save(context.Background(), tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			rClient[0].Credentials.Secret = tc.client.Credentials.Secret
			assert.Equal(t, tc.client, rClient[0], fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, rClient[0]))
		}
	}
}

func TestClientsRetrieveBySecret(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	client := clients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: clientName,
		Credentials: clients.Credentials{
			Identity: clientIdentity,
			Secret:   testsutil.GenerateUUID(t),
		},
		Metadata: clients.Metadata{},
		Status:   clients.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		secret   string
		response clients.Client
		err      error
	}{
		{
			desc:     "retrieve client by secret successfully",
			secret:   client.Credentials.Secret,
			response: client,
			err:      nil,
		},
		{
			desc:     "retrieve client by invalid secret",
			secret:   "non-existent-secret",
			response: clients.Client{},
			err:      errors.ErrNotFound,
		},
		{
			desc:     "retrieve client by empty secret",
			secret:   "",
			response: clients.Client{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		res, err := repo.RetrieveBySecret(context.Background(), tc.secret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, res, tc.response, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, res))
	}
}

func TestDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	client := clients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: clientName,
		Credentials: clients.Credentials{
			Identity: clientIdentity,
			Secret:   testsutil.GenerateUUID(t),
		},
		Status: clients.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "delete client successfully",
			id:   client.ID,
			err:  nil,
		},
		{
			desc: "delete client with invalid id",
			id:   invalidClientID,
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "delete client with empty id",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := repo.Delete(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
