// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/clients/postgres"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxNameSize = 1024
	password    = "$tr0ngPassw0rd"
	emailSuffix = "@example.com"
)

var (
	invalidName     = strings.Repeat("m", maxNameSize+10)
	clientIdentity  = "client-identity@example.com"
	clientName      = "client name"
	invalidDomainID = strings.Repeat("m", maxNameSize+10)
	namegen         = namegenerator.NewGenerator()
	validTimestamp  = time.Now().UTC().Truncate(time.Millisecond)
	validClient     = clients.Client{
		ID:        testsutil.GenerateUUID(&testing.T{}),
		Domain:    testsutil.GenerateUUID(&testing.T{}),
		Name:      namegen.Generate(),
		Metadata:  map[string]interface{}{"key": "value"},
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		Status:    clients.EnabledStatus,
	}
	invalidID = strings.Repeat("a", 37)
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
			err: repoerr.ErrConflict,
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
			err: repoerr.ErrConflict,
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
			err: repoerr.ErrMalformedEntity,
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
			err: repoerr.ErrMalformedEntity,
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
			err: repoerr.ErrMalformedEntity,
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
			err: repoerr.ErrMalformedEntity,
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
			err: repoerr.ErrMalformedEntity,
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

func TestUpdate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	_, err := repo.Save(context.Background(), validClient)
	require.Nil(t, err, fmt.Sprintf("save client unexpected error: %s", err))

	cases := []struct {
		desc   string
		update string
		client clients.Client
		err    error
	}{
		{
			desc:   "update client successfully",
			update: "all",
			client: clients.Client{
				ID:        validClient.ID,
				Name:      namegen.Generate(),
				Metadata:  map[string]interface{}{"key": "value"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update client name",
			update: "name",
			client: clients.Client{
				ID:        validClient.ID,
				Name:      namegen.Generate(),
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update client metadata",
			update: "metadata",
			client: clients.Client{
				ID:        validClient.ID,
				Metadata:  map[string]interface{}{"key1": "value1"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update client with invalid ID",
			update: "all",
			client: clients.Client{
				ID:        testsutil.GenerateUUID(t),
				Name:      namegen.Generate(),
				Metadata:  map[string]interface{}{"key": "value"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update client with empty ID",
			update: "all",
			client: clients.Client{
				Name:      namegen.Generate(),
				Metadata:  map[string]interface{}{"key": "value"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			client, err := repo.Update(context.Background(), tc.client)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.client.ID, client.ID, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.ID, client.ID))
				assert.Equal(t, tc.client.UpdatedAt, client.UpdatedAt, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.UpdatedAt, client.UpdatedAt))
				assert.Equal(t, tc.client.UpdatedBy, client.UpdatedBy, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.UpdatedBy, client.UpdatedBy))
				switch tc.update {
				case "all":
					assert.Equal(t, tc.client.Name, client.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.Name, client.Name))
					assert.Equal(t, tc.client.Metadata, client.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.Metadata, client.Metadata))
				case "name":
					assert.Equal(t, tc.client.Name, client.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.Name, client.Name))
				case "metadata":
					assert.Equal(t, tc.client.Metadata, client.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.Metadata, client.Metadata))
				}
			}
		})
	}
}

func TestUpdateTags(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	client1 := generateClient(t, clients.EnabledStatus, repo)
	client2 := generateClient(t, clients.DisabledStatus, repo)

	cases := []struct {
		desc   string
		client clients.Client
		err    error
	}{
		{
			desc: "for enabled client",
			client: clients.Client{
				ID:   client1.ID,
				Tags: namegen.GenerateMultiple(5),
			},
			err: nil,
		},
		{
			desc: "for disabled client",
			client: clients.Client{
				ID:   client2.ID,
				Tags: namegen.GenerateMultiple(5),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid client",
			client: clients.Client{
				ID:   testsutil.GenerateUUID(t),
				Tags: namegen.GenerateMultiple(5),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: clients.Client{},
			err:    repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.client.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.client.UpdatedBy = testsutil.GenerateUUID(t)
			expected, err := repo.UpdateTags(context.Background(), c.client)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.client.Tags, expected.Tags)
				assert.Equal(t, c.client.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.client.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestUpdateIdentity(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	client1 := generateClient(t, clients.EnabledStatus, repo)
	client2 := generateClient(t, clients.DisabledStatus, repo)

	cases := []struct {
		desc   string
		client clients.Client
		err    error
	}{
		{
			desc: "for enabled client",
			client: clients.Client{
				ID: client1.ID,
				Credentials: clients.Credentials{
					Identity: namegen.Generate() + emailSuffix,
				},
			},
			err: nil,
		},
		{
			desc: "for disabled client",
			client: clients.Client{
				ID: client2.ID,
				Credentials: clients.Credentials{
					Identity: namegen.Generate() + emailSuffix,
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid client",
			client: clients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: clients.Credentials{
					Identity: namegen.Generate() + emailSuffix,
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: clients.Client{},
			err:    repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.client.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.client.UpdatedBy = testsutil.GenerateUUID(t)
			expected, err := repo.UpdateIdentity(context.Background(), c.client)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.client.Credentials.Identity, expected.Credentials.Identity)
				assert.Equal(t, c.client.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.client.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestUpdateSecret(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	client1 := generateClient(t, clients.EnabledStatus, repo)
	client2 := generateClient(t, clients.DisabledStatus, repo)

	cases := []struct {
		desc   string
		client clients.Client
		err    error
	}{
		{
			desc: "for enabled client",
			client: clients.Client{
				ID: client1.ID,
				Credentials: clients.Credentials{
					Secret: "newpassword",
				},
			},
			err: nil,
		},
		{
			desc: "for disabled client",
			client: clients.Client{
				ID: client2.ID,
				Credentials: clients.Credentials{
					Secret: "newpassword",
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid client",
			client: clients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: clients.Credentials{
					Secret: "newpassword",
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: clients.Client{},
			err:    repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.client.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.client.UpdatedBy = testsutil.GenerateUUID(t)
			_, err := repo.UpdateSecret(context.Background(), c.client)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				rc, err := repo.RetrieveByID(context.Background(), c.client.ID)
				require.Nil(t, err, fmt.Sprintf("retrieve client by id during update of secret unexpected error: %s", err))
				assert.Equal(t, c.client.Credentials.Secret, rc.Credentials.Secret)
				assert.Equal(t, c.client.UpdatedAt, rc.UpdatedAt)
				assert.Equal(t, c.client.UpdatedBy, rc.UpdatedBy)
			}
		})
	}
}

func TestChangeStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	client1 := generateClient(t, clients.EnabledStatus, repo)
	client2 := generateClient(t, clients.DisabledStatus, repo)

	cases := []struct {
		desc   string
		client clients.Client
		err    error
	}{
		{
			desc: "for an enabled client",
			client: clients.Client{
				ID:     client1.ID,
				Status: clients.DisabledStatus,
			},
			err: nil,
		},
		{
			desc: "for a disabled client",
			client: clients.Client{
				ID:     client2.ID,
				Status: clients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "for invalid client",
			client: clients.Client{
				ID:     testsutil.GenerateUUID(t),
				Status: clients.DisabledStatus,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: clients.Client{},
			err:    repoerr.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.client.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.client.UpdatedBy = testsutil.GenerateUUID(t)
			expected, err := repo.ChangeStatus(context.Background(), c.client)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.client.Status, expected.Status)
				assert.Equal(t, c.client.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.client.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	nClients := uint64(200)

	expectedClients := []clients.Client{}
	disabledClients := []clients.Client{}
	for i := uint64(0); i < nClients; i++ {
		client := clients.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: testsutil.GenerateUUID(t),
			Name:   namegen.Generate(),
			Credentials: clients.Credentials{
				Identity: namegen.Generate() + emailSuffix,
				Secret:   testsutil.GenerateUUID(t),
			},
			Tags: namegen.GenerateMultiple(5),
			Metadata: clients.Metadata{
				"department": namegen.Generate(),
			},
			Status:    clients.EnabledStatus,
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		if i%50 == 0 {
			client.Status = clients.DisabledStatus
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))
		expectedClients = append(expectedClients, client)
		if client.Status == clients.DisabledStatus {
			disabledClients = append(disabledClients, client)
		}
	}

	cases := []struct {
		desc     string
		pm       clients.Page
		response clients.ClientsPage
		err      error
	}{
		{
			desc: "with empty page",
			pm:   clients.Page{},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  196,
					Offset: 0,
					Limit:  0,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with offset only",
			pm: clients.Page{
				Offset: 50,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 50,
					Limit:  0,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with limit only",
			pm: clients.Page{
				Limit:  50,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  50,
				},
				Clients: expectedClients[:50],
			},
		},
		{
			desc: "retrieve all clients",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with offset and limit",
			pm: clients.Page{
				Offset: 50,
				Limit:  50,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 50,
					Limit:  50,
				},
				Clients: expectedClients[50:100],
			},
		},
		{
			desc: "with offset out of range and limit",
			pm: clients.Page{
				Offset: 1000,
				Limit:  50,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 1000,
					Limit:  50,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with offset and limit out of range",
			pm: clients.Page{
				Offset: 170,
				Limit:  50,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 170,
					Limit:  50,
				},
				Clients: expectedClients[170:200],
			},
		},
		{
			desc: "with metadata",
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				Metadata: expectedClients[0].Metadata,
				Status:   clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong metadata",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Metadata: clients.Metadata{
					"faculty": namegen.Generate(),
				},
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with invalid metadata",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Metadata: clients.Metadata{
					"faculty": make(chan int),
				},
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  uint64(nClients),
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client(nil),
			},
			err: repoerr.ErrViewEntity,
		},
		{
			desc: "with name",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Name:   expectedClients[0].Name,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong name",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Name:   namegen.Generate(),
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with identity",
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				Identity: expectedClients[0].Credentials.Identity,
				Status:   clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong identity",
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				Identity: namegen.Generate(),
				Status:   clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with domain",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Domain: expectedClients[0].Domain,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong domain",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Domain: testsutil.GenerateUUID(t),
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with enabled status",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Status: clients.EnabledStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  196,
					Offset: 0,
					Limit:  10,
				},
				Clients: expectedClients[1:11],
			},
		},
		{
			desc: "with disabled status",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: clients.DisabledStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  4,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: disabledClients,
			},
		},
		{
			desc: "with combined status",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with the wrong status",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: 10,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with tag",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Tag:    expectedClients[0].Tags[0],
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  uint64(nClients),
				},
				Clients: []clients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong tags",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Tag:    namegen.Generate(),
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with multiple parameters",
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				Metadata: expectedClients[0].Metadata,
				Name:     expectedClients[0].Name,
				Tag:      expectedClients[0].Tags[0],
				Identity: expectedClients[0].Credentials.Identity,
				Domain:   expectedClients[0].Domain,
				Status:   clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{expectedClients[0]},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			page, err := repo.RetrieveAll(context.Background(), c.pm)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.response.Total, page.Total)
				assert.Equal(t, c.response.Offset, page.Offset)
				assert.Equal(t, c.response.Limit, page.Limit)
				expected := stripClientDetails(c.response.Clients)
				got := stripClientDetails(page.Clients)
				assert.ElementsMatch(t, expected, got)
			}
		})
	}
}

func TestSearchClients(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	name := namegen.Generate()

	nClients := uint64(200)
	expectedClients := []clients.Client{}
	for i := 0; i < int(nClients); i++ {
		username := name + strconv.Itoa(i) + emailSuffix
		client := clients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: username,
			Credentials: clients.Credentials{
				Identity: username,
				Secret:   testsutil.GenerateUUID(t),
			},
			Metadata:  clients.Metadata{},
			Status:    clients.EnabledStatus,
			CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("save client unexpected error: %s", err))

		expectedClients = append(expectedClients, clients.Client{
			ID:        client.ID,
			Name:      client.Name,
			CreatedAt: client.CreatedAt,
		})
	}

	page, err := repo.RetrieveAll(context.Background(), clients.Page{Offset: 0, Limit: nClients})
	require.Nil(t, err, fmt.Sprintf("retrieve all clients unexpected error: %s", err))
	assert.Equal(t, nClients, page.Total)

	cases := []struct {
		desc     string
		page     clients.Page
		response clients.ClientsPage
		err      error
	}{
		{
			desc: "with empty page",
			page: clients.Page{},
			response: clients.ClientsPage{
				Clients: []clients.Client(nil),
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  0,
				},
			},
			err: nil,
		},
		{
			desc: "with offset only",
			page: clients.Page{
				Offset: 50,
			},
			response: clients.ClientsPage{
				Clients: []clients.Client(nil),
				Page: clients.Page{
					Total:  nClients,
					Offset: 50,
					Limit:  0,
				},
			},
			err: nil,
		},
		{
			desc: "with limit only",
			page: clients.Page{
				Limit: 10,
				Order: "name",
				Dir:   "asc",
			},
			response: clients.ClientsPage{
				Clients: expectedClients[0:10],
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients",
			page: clients.Page{
				Offset: 0,
				Limit:  nClients,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with offset and limit",
			page: clients.Page{
				Offset: 10,
				Limit:  10,
				Order:  "name",
				Dir:    "asc",
			},
			response: clients.ClientsPage{
				Clients: expectedClients[10:20],
				Page: clients.Page{
					Total:  nClients,
					Offset: 10,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with offset out of range and limit",
			page: clients.Page{
				Offset: 1000,
				Limit:  50,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 1000,
					Limit:  50,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc: "with offset and limit out of range",
			page: clients.Page{
				Offset: 190,
				Limit:  50,
				Order:  "name",
				Dir:    "asc",
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 190,
					Limit:  50,
				},
				Clients: expectedClients[190:200],
			},
		},
		{
			desc: "with shorter name",
			page: clients.Page{
				Name:   expectedClients[0].Name[:4],
				Offset: 0,
				Limit:  10,
				Order:  "name",
				Dir:    "asc",
			},
			response: clients.ClientsPage{
				Clients: findClients(expectedClients, expectedClients[0].Name[:4], 0, 10),
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with longer name",
			page: clients.Page{
				Name:   expectedClients[0].Name,
				Offset: 0,
				Limit:  10,
			},
			response: clients.ClientsPage{
				Clients: []clients.Client{expectedClients[0]},
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with name SQL injected",
			page: clients.Page{
				Name:   fmt.Sprintf("%s' OR '1'='1", expectedClients[0].Name[:1]),
				Offset: 0,
				Limit:  10,
			},
			response: clients.ClientsPage{
				Clients: []clients.Client(nil),
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with shorter Identity",
			page: clients.Page{
				Identity: expectedClients[0].Name[:4],
				Offset:   0,
				Limit:    10,
				Order:    "name",
				Dir:      "asc",
			},
			response: clients.ClientsPage{
				Clients: findClients(expectedClients, expectedClients[0].Name[:4], 0, 10),
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with longer Identity",
			page: clients.Page{
				Identity: expectedClients[0].Name,
				Offset:   0,
				Limit:    10,
			},
			response: clients.ClientsPage{
				Clients: []clients.Client{expectedClients[0]},
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with Identity SQL injected",
			page: clients.Page{
				Identity: fmt.Sprintf("%s' OR '1'='1", expectedClients[0].Name[:1]),
				Offset:   0,
				Limit:    10,
			},
			response: clients.ClientsPage{
				Clients: []clients.Client(nil),
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with unknown name",
			page: clients.Page{
				Name:   namegen.Generate(),
				Offset: 0,
				Limit:  10,
			},
			response: clients.ClientsPage{
				Clients: []clients.Client(nil),
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with unknown name SQL injected",
			page: clients.Page{
				Name:   fmt.Sprintf("%s' OR '1'='1", namegen.Generate()),
				Offset: 0,
				Limit:  10,
			},
			response: clients.ClientsPage{
				Clients: []clients.Client(nil),
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with unknown identity",
			page: clients.Page{
				Identity: namegen.Generate(),
				Offset:   0,
				Limit:    10,
			},
			response: clients.ClientsPage{
				Clients: []clients.Client(nil),
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with name in asc order",
			page: clients.Page{
				Order:  "name",
				Dir:    "asc",
				Name:   expectedClients[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: clients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "with name in desc order",
			page: clients.Page{
				Order:  "name",
				Dir:    "desc",
				Name:   expectedClients[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: clients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "with identity in asc order",
			page: clients.Page{
				Order:    "identity",
				Dir:      "asc",
				Identity: expectedClients[0].Name[:1],
				Offset:   0,
				Limit:    10,
			},
			response: clients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "with identity in desc order",
			page: clients.Page{
				Order:    "identity",
				Dir:      "desc",
				Identity: expectedClients[0].Name[:1],
				Offset:   0,
				Limit:    10,
			},
			response: clients.ClientsPage{},
			err:      nil,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			switch response, err := repo.SearchClients(context.Background(), c.page); {
			case err == nil:
				if c.page.Order != "" && c.page.Dir != "" {
					c.response = response
				}
				assert.Nil(t, err)
				assert.Equal(t, c.response.Total, response.Total)
				assert.Equal(t, c.response.Limit, response.Limit)
				assert.Equal(t, c.response.Offset, response.Offset)
				assert.ElementsMatch(t, response.Clients, c.response.Clients)
			default:
				assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			}
		})
	}
}

func TestRetrieveByIDs(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	num := 10

	var items []clients.Client
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		client := clients.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: testsutil.GenerateUUID(t),
			Name:   name,
			Credentials: clients.Credentials{
				Identity: name + emailSuffix,
				Secret:   testsutil.GenerateUUID(t),
			},
			Tags:      namegen.GenerateMultiple(5),
			Metadata:  map[string]interface{}{"name": name},
			CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
			Status:    clients.EnabledStatus,
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))
		items = append(items, client)
	}

	cases := []struct {
		desc     string
		ids      []string
		response clients.ClientsPage
		err      error
	}{
		{
			desc: "successfully",
			ids:  getIDs(items[0:3]),
			response: clients.ClientsPage{
				Page: clients.Page{
					Total: 3,
				},
				Clients: items[0:3],
			},
			err: nil,
		},
		{
			desc: "successfully",
			ids:  getIDs(items[3:6]),
			response: clients.ClientsPage{
				Page: clients.Page{
					Total: 3,
				},
				Clients: items[3:6],
			},
			err: nil,
		},
		{
			desc: "with empty ids",
			ids:  []string{},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total: 0,
				},
				Clients: []clients.Client(nil),
			},
			err: nil,
		},
		{
			desc: "with valid and invalid ids",
			ids:  append(getIDs(items[0:3]), testsutil.GenerateUUID(t)),
			response: clients.ClientsPage{
				Page: clients.Page{
					Total: 3,
				},
				Clients: items[0:3],
			},
			err: nil,
		},
		{
			desc: "with invalid ids",
			ids:  []string{testsutil.GenerateUUID(t)},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total: 0,
				},
				Clients: []clients.Client(nil),
			},
			err: nil,
		},
	}

	for _, c := range cases {
		response, err := repo.RetrieveByIds(context.Background(), c.ids)
		assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("%s: expected %s got %s\n", c.desc, c.err, err))
		if err == nil {
			assert.Equal(t, c.response.Total, response.Total)
			expected := stripClientDetails(c.response.Clients)
			got := stripClientDetails(response.Clients)
			assert.ElementsMatch(t, expected, got)
		}
	}
}

func TestAddConnection(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	client := generateClient(t, clients.EnabledStatus, repo)

	validConnection := clients.Connection{
		ClientID:  client.ID,
		ChannelID: testsutil.GenerateUUID(t),
		DomainID:  client.Domain,
		Type:      connections.Publish,
	}

	cases := []struct {
		desc       string
		connection clients.Connection
		err        error
	}{
		{
			desc:       "add connection successfully",
			connection: validConnection,
			err:        nil,
		},
		{
			desc: "add connection with non-existent client",
			connection: clients.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  client.Domain,
				Type:      connections.Publish,
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add connection with non-existent domain",
			connection: clients.Connection{
				ClientID:  client.ID,
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: repoerr.ErrCreateEntity,
		},

		{
			desc: "add connection with invalid client ID",
			connection: clients.Connection{
				ClientID:  invalidID,
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add connection with invalid channel ID",
			connection: clients.Connection{
				ClientID:  client.ID,
				ChannelID: invalidID,
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: repoerr.ErrMalformedEntity,
		},
		{
			desc: "add connection with invalid domain ID",
			connection: clients.Connection{
				ClientID:  client.ID,
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  invalidID,
				Type:      connections.Publish,
			},
			err: repoerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.AddConnections(context.Background(), []clients.Connection{tc.connection})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRemoveConnection(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	client := generateClient(t, clients.EnabledStatus, repo)

	validConnection := clients.Connection{
		ClientID:  client.ID,
		ChannelID: testsutil.GenerateUUID(t),
		DomainID:  client.Domain,
		Type:      connections.Publish,
	}

	err := repo.AddConnections(context.Background(), []clients.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc       string
		connection clients.Connection
		err        error
	}{
		{
			desc:       "remove connection successfully",
			connection: validConnection,
			err:        nil,
		},
		{
			desc: "remove connection with non-existent channel",
			connection: clients.Connection{
				ClientID:  client.ID,
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  client.Domain,
				Type:      connections.Publish,
			},
			err: nil,
		},
		{
			desc: "remove connection with non-existent domain",
			connection: clients.Connection{
				ClientID:  client.ID,
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  testsutil.GenerateUUID(t),
				Type:      connections.Publish,
			},
			err: nil,
		},
		{
			desc: "remove connection with non-existent client",
			connection: clients.Connection{
				ClientID:  testsutil.GenerateUUID(t),
				ChannelID: testsutil.GenerateUUID(t),
				DomainID:  client.Domain,
				Type:      connections.Publish,
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveConnections(context.Background(), []clients.Connection{tc.connection})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	client := generateClient(t, clients.EnabledStatus, repo)

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
			id:   testsutil.GenerateUUID(t),
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

func TestSetParentGroup(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	validClient := generateClient(t, clients.EnabledStatus, repo)

	cases := []struct {
		desc          string
		id            string
		parentGroupID string
		err           error
	}{
		{
			desc:          "set parent group successfully",
			id:            validClient.ID,
			parentGroupID: testsutil.GenerateUUID(t),
			err:           nil,
		},
		{
			desc:          "set parent group with invalid ID",
			id:            invalidID,
			parentGroupID: testsutil.GenerateUUID(t),
			err:           repoerr.ErrNotFound,
		},
		{
			desc:          "set parent group with empty ID",
			id:            "",
			parentGroupID: testsutil.GenerateUUID(t),
			err:           repoerr.ErrNotFound,
		},
		{
			desc:          "set parent group with invalid parent group ID",
			id:            validClient.ID,
			parentGroupID: invalidID,
			err:           repoerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.SetParentGroup(context.Background(), clients.Client{
				ID:          tc.id,
				ParentGroup: tc.parentGroupID,
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				resp, err := repo.RetrieveByID(context.Background(), tc.id)
				require.Nil(t, err, fmt.Sprintf("retrieve client unexpected error: %s", err))
				assert.Equal(t, tc.id, resp.ID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.id, resp.ID))
				assert.Equal(t, tc.parentGroupID, resp.ParentGroup, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.parentGroupID, resp.ParentGroup))
			}
		})
	}
}

func TestRemoveParentGroup(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	validClient := generateClient(t, clients.EnabledStatus, repo)

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove parent group successfully",
			id:   validClient.ID,
			err:  nil,
		},
		{
			desc: "remove parent group with invalid ID",
			id:   invalidID,
			err:  repoerr.ErrNotFound,
		},
		{
			desc: "remove parent group with empty ID",
			id:   "",
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveParentGroup(context.Background(), clients.Client{
				ID: tc.id,
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				resp, err := repo.RetrieveByID(context.Background(), tc.id)
				require.Nil(t, err, fmt.Sprintf("retrieve client unexpected error: %s", err))
				assert.Equal(t, tc.id, resp.ID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.id, resp.ID))
				assert.Equal(t, "", resp.ParentGroup, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, "", resp.ParentGroup))
			}
		})
	}
}

func TestClientConnectionsCount(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	validClient := generateClient(t, clients.EnabledStatus, repo)

	rConnections := []clients.Connection{}
	for i := 0; i < 10; i++ {
		connection := clients.Connection{
			ClientID:  validClient.ID,
			ChannelID: testsutil.GenerateUUID(t),
			DomainID:  validClient.Domain,
			Type:      connections.Publish,
		}
		rConnections = append(rConnections, connection)
	}

	err := repo.AddConnections(context.Background(), rConnections)
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc     string
		clientID string
		count    uint64
		err      error
	}{
		{
			desc:     "get client connections count successfully",
			clientID: validClient.ID,
			count:    10,
			err:      nil,
		},
		{
			desc:     "get client connections count with non-existent client",
			clientID: testsutil.GenerateUUID(t),
			count:    0,
			err:      nil,
		},
		{
			desc:     "get client connections count with empty client ID",
			clientID: "",
			count:    0,
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			count, err := repo.ClientConnectionsCount(context.Background(), tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.count, count, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.count, count))
		})
	}
}

func TestDoesClientHaveConnections(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	validClient := generateClient(t, clients.EnabledStatus, repo)

	validConnection := clients.Connection{
		ClientID:  validClient.ID,
		ChannelID: testsutil.GenerateUUID(t),
		DomainID:  validClient.Domain,
		Type:      connections.Publish,
	}

	err := repo.AddConnections(context.Background(), []clients.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc     string
		clientID string
		has      bool
		err      error
	}{
		{
			desc:     "check if client has connections successfully",
			clientID: validClient.ID,
			has:      true,
			err:      nil,
		},
		{
			desc:     "check if client has connections with non-existent channel",
			clientID: testsutil.GenerateUUID(t),
			has:      false,
			err:      nil,
		},
		{
			desc:     "check if client has connections with empty channel ID",
			clientID: "",
			has:      false,
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			has, err := repo.DoesClientHaveConnections(context.Background(), tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.has, has, fmt.Sprintf("%s: expected %t got %t\n", tc.desc, tc.has, has))
		})
	}
}

func TestRemoveClientConnections(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	validClient := generateClient(t, clients.EnabledStatus, repo)

	validConnection := clients.Connection{
		ClientID:  validClient.ID,
		ChannelID: testsutil.GenerateUUID(t),
		DomainID:  validClient.Domain,
		Type:      connections.Publish,
	}

	err := repo.AddConnections(context.Background(), []clients.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc     string
		clientID string
		err      error
	}{
		{
			desc:     "remove client connections successfully",
			clientID: validConnection.ClientID,
			err:      nil,
		},
		{
			desc:     "remove client connections with non-existent client",
			clientID: testsutil.GenerateUUID(t),
			err:      nil,
		},
		{
			desc:     "remove client connections with empty client ID",
			clientID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveClientConnections(context.Background(), tc.clientID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRemoveChannelConnections(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM connections")
		require.Nil(t, err, fmt.Sprintf("clean connections unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	validClient := generateClient(t, clients.EnabledStatus, repo)

	validConnection := clients.Connection{
		ClientID:  validClient.ID,
		ChannelID: testsutil.GenerateUUID(t),
		DomainID:  validClient.Domain,
		Type:      connections.Publish,
	}

	err := repo.AddConnections(context.Background(), []clients.Connection{validConnection})
	require.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))

	cases := []struct {
		desc      string
		channelID string
		err       error
	}{
		{
			desc:      "remove channel connections successfully",
			channelID: validConnection.ChannelID,
			err:       nil,
		},
		{
			desc:      "remove channel connections with non-existent channel",
			channelID: testsutil.GenerateUUID(t),
			err:       nil,
		},
		{
			desc:      "remove channel connections with empty channel ID",
			channelID: "",
			err:       nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveChannelConnections(context.Background(), tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func TestRetrieveParentGroupClients(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	var items []clients.Client
	parentID := testsutil.GenerateUUID(t)
	for i := 0; i < 10; i++ {
		name := namegen.Generate()
		client := clients.Client{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			ParentGroup: parentID,
			Name:        name,
			Metadata:    map[string]interface{}{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      clients.EnabledStatus,
		}
		items = append(items, client)
	}

	_, err := repo.Save(context.Background(), items...)
	require.Nil(t, err, fmt.Sprintf("create client unexpected error: %s", err))

	cases := []struct {
		desc          string
		parentGroupID string
		resp          []clients.Client
		err           error
	}{
		{
			desc:          "retrieve parent group clients successfully",
			parentGroupID: parentID,
			resp:          items[:10],
			err:           nil,
		},
		{
			desc:          "retrieve parent group clients with non-existent client",
			parentGroupID: testsutil.GenerateUUID(t),
			resp:          []clients.Client(nil),
			err:           nil,
		},
		{
			desc:          "retrieve parent group clients with empty client ID",
			parentGroupID: "",
			resp:          []clients.Client(nil),
			err:           nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			clients, err := repo.RetrieveParentGroupClients(context.Background(), tc.parentGroupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				got := stripClientDetails(clients)
				expected := stripClientDetails(tc.resp)
				assert.Equal(t, len(tc.resp), len(clients), fmt.Sprintf("%s: expected %d got %d\n", tc.desc, len(tc.resp), len(clients)))
				assert.ElementsMatch(t, expected, got, fmt.Sprintf("%s: expected %+v got %+v\n", tc.desc, expected, got))
			}
		})
	}
}

func TestUnsetParentGroupFromClients(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	var items []clients.Client
	parentID := testsutil.GenerateUUID(t)
	for i := 0; i < 10; i++ {
		name := namegen.Generate()
		client := clients.Client{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			ParentGroup: parentID,
			Name:        name,
			Metadata:    map[string]interface{}{"name": name},
			CreatedAt:   time.Now().UTC().Truncate(time.Microsecond),
			Status:      clients.EnabledStatus,
		}
		items = append(items, client)
	}

	_, err := repo.Save(context.Background(), items...)
	require.Nil(t, err, fmt.Sprintf("create client unexpected error: %s", err))

	cases := []struct {
		desc          string
		parentGroupID string
		err           error
	}{
		{
			desc:          "unset parent group from clients successfully",
			parentGroupID: parentID,
			err:           nil,
		},
		{
			desc:          "unset parent group from clients with non-existent id",
			parentGroupID: testsutil.GenerateUUID(t),
			err:           nil,
		},
		{
			desc:          "unset parent group from clients with empty client ID",
			parentGroupID: "",
			err:           nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.UnsetParentGroupFromClient(context.Background(), tc.parentGroupID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		})
	}
}

func generateClient(t *testing.T, status clients.Status, repo clients.Repository) clients.Client {
	client := clients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: namegen.Generate(),
		Credentials: clients.Credentials{
			Identity: namegen.Generate() + emailSuffix,
			Secret:   testsutil.GenerateUUID(t),
		},
		Tags: namegen.GenerateMultiple(5),
		Metadata: clients.Metadata{
			"name": namegen.Generate(),
		},
		Status:    status,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	_, err := repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))

	return client
}

func getIDs(clis []clients.Client) []string {
	var ids []string
	for _, client := range clis {
		ids = append(ids, client.ID)
	}

	return ids
}

func stripClientDetails(clients []clients.Client) []clients.Client {
	for i := range clients {
		clients[i].CreatedAt = validTimestamp
		clients[i].Credentials.Secret = ""
	}

	return clients
}

func findClients(clis []clients.Client, query string, offset, limit uint64) []clients.Client {
	rclients := []clients.Client{}
	for _, client := range clis {
		if strings.Contains(client.Name, query) {
			rclients = append(rclients, client)
		}
	}

	if offset > uint64(len(rclients)) {
		return []clients.Client{}
	}

	if limit > uint64(len(rclients)) {
		return rclients[offset:]
	}

	return rclients[offset:limit]
}
