// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/clients/postgres"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxNameSize = 1024

var (
	invalidName     = strings.Repeat("m", maxNameSize+10)
	clientIdentity  = "client-identity@example.com"
	clientName      = "client name"
	invalidDomainID = strings.Repeat("m", maxNameSize+10)
	namegen         = namegenerator.NewGenerator()
)

func TestClientsSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	uid := testsutil.GenerateUUID(t)
	domainID := testsutil.GenerateUUID(t)
	secret := testsutil.GenerateUUID(t)

	cases := []struct {
		desc    string
		clients []clients.Client
		err     error
	}{
		{
			desc: "add new client successfully",
			clients: []clients.Client{
				{
					ID:     uid,
					Domain: domainID,
					Name:   clientName,
					Credentials: clients.Credentials{
						Identity: clientIdentity,
						Secret:   secret,
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: nil,
		},
		{
			desc: "add multiple clients successfully",
			clients: []clients.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: nil,
		},
		{
			desc: "add new client with duplicate secret",
			clients: []clients.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: domainID,
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Identity: clientIdentity,
						Secret:   secret,
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add multiple clients with one client having duplicate secret",
			clients: []clients.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: domainID,
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Identity: clientIdentity,
						Secret:   secret,
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add new client without domain id",
			clients: []clients.Client{
				{
					ID:   testsutil.GenerateUUID(t),
					Name: clientName,
					Credentials: clients.Credentials{
						Identity: "withoutdomain-client@example.com",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: nil,
		},
		{
			desc: "add client with invalid client id",
			clients: []clients.Client{
				{
					ID:     invalidName,
					Domain: domainID,
					Name:   clientName,
					Credentials: clients.Credentials{
						Identity: "invalidid-client@example.com",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add multiple clients with one client having invalid client id",
			clients: []clients.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
				{
					ID:     invalidName,
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client name",
			clients: []clients.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Name:   invalidName,
					Domain: domainID,
					Credentials: clients.Credentials{
						Identity: "invalidname-client@example.com",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client domain id",
			clients: []clients.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: invalidDomainID,
					Credentials: clients.Credentials{
						Identity: "invaliddomainid-client@example.com",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client identity",
			clients: []clients.Client{
				{
					ID:   testsutil.GenerateUUID(t),
					Name: clientName,
					Credentials: clients.Credentials{
						Identity: invalidName,
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
					Status:   clients.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add client with a missing client identity",
			clients: []clients.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   "missing-client-identity",
					Credentials: clients.Credentials{
						Identity: "",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: clients.Metadata{},
				},
			},
			err: nil,
		},
		{
			desc: "add client with a missing client secret",
			clients: []clients.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Credentials: clients.Credentials{
						Identity: "missing-client-secret@example.com",
						Secret:   "",
					},
					Metadata: clients.Metadata{},
				},
			},
			err: nil,
		},
		{
			desc: "add a client with invalid metadata",
			clients: []clients.Client{
				{
					ID:   testsutil.GenerateUUID(t),
					Name: namegen.Generate(),
					Credentials: clients.Credentials{
						Identity: fmt.Sprintf("%s@example.com", namegen.Generate()),
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: map[string]interface{}{
						"key": make(chan int),
					},
				},
			},
			err: errors.ErrMalformedEntity,
		},
	}
	for _, tc := range cases {
		rClients, err := repo.Save(context.Background(), tc.clients...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			for i := range rClients {
				tc.clients[i].Credentials.Secret = rClients[i].Credentials.Secret
			}
			assert.Equal(t, tc.clients, rClients, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.clients, rClients))
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
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve client by empty secret",
			secret:   "",
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		res, err := repo.RetrieveBySecret(context.Background(), tc.secret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, res, tc.response, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, res))
	}
}

func TestRetrieveByID(t *testing.T) {
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
		id       string
		response clients.Client
		err      error
	}{
		{
			desc:     "successfully",
			id:       client.ID,
			response: client,
			err:      nil,
		},
		{
			desc:     "with invalid id",
			id:       testsutil.GenerateUUID(t),
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "with empty id",
			id:       "",
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cli, err := repo.RetrieveByID(context.Background(), c.id)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s got %s\n", c.err, err))
			if err == nil {
				assert.Equal(t, client.ID, cli.ID)
				assert.Equal(t, client.Name, cli.Name)
				assert.Equal(t, client.Metadata, cli.Metadata)
				assert.Equal(t, client.Credentials.Identity, cli.Credentials.Identity)
				assert.Equal(t, client.Credentials.Secret, cli.Credentials.Secret)
				assert.Equal(t, client.Status, cli.Status)
			}
		})
	}
}
