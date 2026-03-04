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
	"github.com/absmach/supermq/domains"
	dpostgres "github.com/absmach/supermq/domains/postgres"
	"github.com/absmach/supermq/groups"
	gpostgres "github.com/absmach/supermq/groups/postgres"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxNameSize = 1024
	emailSuffix = "@example.com"
	defOrder    = "created_at"
	ascDir      = "asc"
	descDir     = "desc"
)

var (
	invalidName     = strings.Repeat("m", maxNameSize+10)
	clientIdentity  = "client-identity@example.com"
	clientName      = "client name"
	invalidDomainID = strings.Repeat("m", maxNameSize+10)
	namegen         = namegenerator.NewGenerator()
	validTimestamp  = time.Now().UTC().Truncate(time.Millisecond)
	validClient     = clients.Client{
		ID:              testsutil.GenerateUUID(&testing.T{}),
		Domain:          testsutil.GenerateUUID(&testing.T{}),
		Name:            namegen.Generate(),
		Metadata:        map[string]any{"key": "value"},
		PrivateMetadata: map[string]any{"key": "value"},
		CreatedAt:       time.Now().UTC().Truncate(time.Microsecond),
		Status:          clients.EnabledStatus,
	}
	invalidID         = strings.Repeat("a", 37)
	directAccess      = "direct"
	directGroupAccess = "direct_group"
	domainAccess      = "domain"
	availableActions  = []string{
		"delete",
		"membership",
		"read",
		"update",
	}
	domainAvailableActions = []string{
		"client_add_role_users",
		"client_connect_to_channel",
		"client_create",
		"client_delete",
		"client_manage_role",
		"client_read",
		"client_remove_role_users",
		"client_set_parent_group",
		"client_update",
		"client_view_role_users",
	}
	groupAvailableActions = []string{
		"client_add_role_users",
		"client_connect_to_channel",
		"client_create",
		"client_delete",
		"client_manage_role",
		"client_read",
		"client_remove_role_users",
		"client_set_parent_group",
		"client_update",
		"client_view_role_users",
		"subgroup_client_add_role_users",
		"subgroup_client_connect_to_channel",
		"subgroup_client_create",
		"subgroup_client_delete",
		"subgroup_client_manage_role",
		"subgroup_client_read",
		"subgroup_client_remove_role_users",
		"subgroup_client_set_parent_group",
		"subgroup_client_update",
		"subgroup_client_view_role_users",
		"subgroup_manage_role",
		"subgroup_membership",
		"subgroup_read",
		"subgroup_remove_role_users",
		"subgroup_set_child",
		"subgroup_set_parent",
		"subgroup_update",
	}
	errClientSecretNotAvailable = errors.New("client key is not available")
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

	duplicateClientID := testsutil.GenerateUUID(t)

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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
				},
			},
			err: errClientSecretNotAvailable,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: domainID,
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Identity: clientIdentity,
						Secret:   secret,
					},
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
				},
			},
			err: errClientSecretNotAvailable,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
				},
				{
					ID:     invalidName,
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: clients.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
					Status:          clients.EnabledStatus,
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
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
					PrivateMetadata: map[string]any{"key": "value"},
					Metadata:        map[string]any{"key": "value"},
				},
			},
			err: nil,
		},
		{
			desc: "add a client with invalid private metadata",
			clients: []clients.Client{
				{
					ID:   testsutil.GenerateUUID(t),
					Name: namegen.Generate(),
					Credentials: clients.Credentials{
						Identity: fmt.Sprintf("%s@example.com", namegen.Generate()),
						Secret:   testsutil.GenerateUUID(t),
					},
					PrivateMetadata: map[string]any{
						"key": make(chan int),
					},
				},
			},
			err: errors.ErrMalformedEntity,
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
					PrivateMetadata: map[string]any{
						"key": make(chan int),
					},
				},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with duplicate name",
			clients: []clients.Client{
				{
					ID:              duplicateClientID,
					Domain:          validClient.Domain,
					Name:            validClient.Name,
					PrivateMetadata: map[string]any{"key": "different_value"},
					Metadata:        map[string]any{},
					CreatedAt:       validTimestamp,
					Status:          clients.EnabledStatus,
				},
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			rClients, err := repo.Save(context.Background(), tc.clients...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				for i := range rClients {
					tc.clients[i].Credentials.Secret = rClients[i].Credentials.Secret
				}
				assert.Equal(t, tc.clients, rClients, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.clients, rClients))
			}
		})
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
		Domain:          testsutil.GenerateUUID(t),
		Metadata:        clients.Metadata{},
		PrivateMetadata: clients.Metadata{},
		Status:          clients.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), client)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		secret   string
		id       string
		response clients.Client
		prefix   authn.AuthPrefix
		err      error
	}{
		{
			desc:     "retrieve client by secret with no id",
			secret:   client.Credentials.Secret,
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve client by client ID and secret successfully",
			secret:   client.Credentials.Secret,
			id:       client.ID,
			prefix:   authn.BasicAuth,
			response: client,
			err:      nil,
		},
		{
			desc:     "retrieve client by client ID invalid secret",
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
		{
			desc:     "retrieve client by client ID and secret with an invalid ID type",
			secret:   client.Credentials.Secret,
			id:       client.ID,
			prefix:   authn.DomainAuth,
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve client by domain ID and secret successfully",
			secret:   client.Credentials.Secret,
			id:       client.Domain,
			prefix:   authn.DomainAuth,
			response: client,
			err:      nil,
		},
		{
			desc:     "retrieve client by domain ID and secret with an invalid ID type",
			secret:   client.Credentials.Secret,
			id:       client.Domain,
			prefix:   authn.BasicAuth,
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res, err := repo.RetrieveBySecret(context.Background(), tc.secret, tc.id, tc.prefix)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, res, tc.response, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, res))
		})
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
		PrivateMetadata: clients.Metadata{
			"key": "value",
		},
		Metadata: clients.Metadata{
			"key": "value",
		},
		Status: clients.EnabledStatus,
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
				assert.Equal(t, client.PrivateMetadata, cli.PrivateMetadata)
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
				ID:              validClient.ID,
				Name:            namegen.Generate(),
				PrivateMetadata: map[string]any{"key": "value"},
				Metadata:        map[string]any{"key": "value"},
				UpdatedAt:       validTimestamp,
				UpdatedBy:       testsutil.GenerateUUID(t),
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
			desc:   "update client private metadata",
			update: "private_metadata",
			client: clients.Client{
				ID:              validClient.ID,
				PrivateMetadata: map[string]any{"key1": "value1"},
				UpdatedAt:       validTimestamp,
				UpdatedBy:       testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update client metadata",
			update: "metadata",
			client: clients.Client{
				ID:        validClient.ID,
				Metadata:  map[string]any{"key1": "value1"},
				UpdatedAt: validTimestamp,
				UpdatedBy: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc:   "update client with invalid ID",
			update: "all",
			client: clients.Client{
				ID:              testsutil.GenerateUUID(t),
				Name:            namegen.Generate(),
				PrivateMetadata: map[string]any{"key": "value"},
				Metadata:        map[string]any{"key": "value"},
				UpdatedAt:       validTimestamp,
				UpdatedBy:       testsutil.GenerateUUID(t),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update client with empty ID",
			update: "all",
			client: clients.Client{
				Name:            namegen.Generate(),
				PrivateMetadata: map[string]any{"key": "value"},
				Metadata:        map[string]any{"key": "value"},
				UpdatedAt:       validTimestamp,
				UpdatedBy:       testsutil.GenerateUUID(t),
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
					assert.Equal(t, tc.client.PrivateMetadata, client.PrivateMetadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.PrivateMetadata, client.PrivateMetadata))
					assert.Equal(t, tc.client.Metadata, client.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.Metadata, client.Metadata))
				case "name":
					assert.Equal(t, tc.client.Name, client.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.Name, client.Name))
				case "private_metadata":
					assert.Equal(t, tc.client.PrivateMetadata, client.PrivateMetadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client.PrivateMetadata, client.PrivateMetadata))
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

func TestRetrieveByIDsWithRoles(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	nClients := uint64(10)

	domainID := testsutil.GenerateUUID(t)
	userID := testsutil.GenerateUUID(t)
	expectedClients := []clients.Client{}
	for range nClients {
		client := clients.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: domainID,
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
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))
		newRolesProvision := []roles.RoleProvision{
			{
				Role: roles.Role{
					ID:        testsutil.GenerateUUID(t) + "_" + client.ID,
					Name:      "admin",
					EntityID:  client.ID,
					CreatedAt: validTimestamp,
					CreatedBy: userID,
				},
				OptionalActions: availableActions,
				OptionalMembers: []string{userID},
			},
		}
		npr, err := repo.AddRoles(context.Background(), newRolesProvision)
		require.Nil(t, err, fmt.Sprintf("add roles unexpected error: %s", err))
		expectedClient := client
		expectedClient.Roles = []roles.MemberRoleActions{
			{
				RoleID:     npr[0].Role.ID,
				RoleName:   npr[0].Role.Name,
				Actions:    npr[0].OptionalActions,
				AccessType: directAccess,
			},
		}
		expectedClients = append(expectedClients, expectedClient)
	}

	cases := []struct {
		desc     string
		clientID string
		userID   string
		response clients.Client
		err      error
	}{
		{
			desc:     "retrieve client with role successfully",
			clientID: expectedClients[0].ID,
			userID:   userID,
			response: expectedClients[0],
			err:      nil,
		},
		{
			desc:     "retrieve another client with role successfully",
			clientID: expectedClients[1].ID,
			userID:   userID,
			response: expectedClients[1],
			err:      nil,
		},
		{
			desc:     "retrieve client with invalid client id",
			clientID: testsutil.GenerateUUID(t),
			userID:   userID,
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve client with empty client id",
			clientID: "",
			userID:   userID,
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve client with invalid user id",
			clientID: expectedClients[0].ID,
			userID:   testsutil.GenerateUUID(t),
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve client with empty user id",
			clientID: expectedClients[0].ID,
			userID:   "",
			response: clients.Client{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			client, err := repo.RetrieveByIDWithRoles(context.Background(), tc.clientID, tc.userID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected %s to contain %s\n", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.response, client, fmt.Sprintf("expected %v got %v\n", tc.response, client))
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
	connectedClient := clients.Client{}

	channelID := testsutil.GenerateUUID(t)
	expectedClients := []clients.Client{}
	disabledClients := []clients.Client{}
	reversedClients := []clients.Client{}
	baseTime := time.Now().UTC().Truncate(time.Millisecond)
	for i := uint64(0); i < nClients; i++ {
		client := clients.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: testsutil.GenerateUUID(t),
			Name:   namegen.Generate(),
			Credentials: clients.Credentials{
				Identity: namegen.Generate() + emailSuffix,
				Secret:   testsutil.GenerateUUID(t),
			},
			Tags: []string{"tag1", "tag2"},
			Metadata: clients.Metadata{
				"department": namegen.Generate(),
			},
			Status:    clients.EnabledStatus,
			CreatedAt: baseTime.Add(time.Duration(i) * time.Millisecond),
			UpdatedAt: baseTime.Add(time.Duration(i) * time.Millisecond),
		}
		if i%50 == 0 {
			client.Status = clients.DisabledStatus
		}
		if i%99 == 0 {
			client.Tags = []string{"tag1", "tag3"}
		}
		_, err := repo.Save(context.Background(), client)
		if i == 0 {
			conn := clients.Connection{
				ClientID:  client.ID,
				ChannelID: channelID,
				DomainID:  client.Domain,
				Type:      connections.Publish,
			}
			err = repo.AddConnections(context.Background(), []clients.Connection{conn})
			assert.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))
			connectedClient = client
			connectedClient.ConnectionTypes = []connections.ConnType{connections.Publish}
		}
		require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))
		expectedClients = append(expectedClients, client)
		if client.Status == clients.DisabledStatus {
			disabledClients = append(disabledClients, client)
		}
	}

	for i := len(expectedClients) - 1; i >= 0; i-- {
		reversedClients = append(reversedClients, expectedClients[i])
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Limit:  10,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
				Clients: expectedClients[:10],
			},
		},
		{
			desc: "retrieve all clients",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:    defOrder,
				Dir:      ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:    defOrder,
				Dir:      ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
				Order:  defOrder,
				Dir:    ascDir,
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
			desc: "with single tag",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Tags:   clients.TagsQuery{Elements: []string{"tag1"}, Operator: clients.OrOp},
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  200,
					Offset: 0,
					Limit:  uint64(nClients),
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with multiple tags and OR operator",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Tags:   clients.TagsQuery{Elements: []string{"tag2", "tag3"}, Operator: clients.OrOp},
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  200,
					Offset: 0,
					Limit:  uint64(nClients),
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with multiple tags and AND operator",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Tags:   clients.TagsQuery{Elements: []string{"tag1", "tag3"}, Operator: clients.AndOp},
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  3,
					Offset: 0,
					Limit:  uint64(nClients),
				},
				Clients: []clients.Client{expectedClients[0], expectedClients[99], expectedClients[198]},
			},
		},
		{
			desc: "with wrong tags",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Tags:   clients.TagsQuery{Elements: []string{namegen.Generate(), namegen.Generate()}, Operator: clients.OrOp},
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
				Tags:     clients.TagsQuery{Elements: []string{expectedClients[0].Tags[0]}, Operator: clients.OrOp},
				Identity: expectedClients[0].Credentials.Identity,
				Domain:   expectedClients[0].Domain,
				Status:   clients.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
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
			desc: "with id",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				ID:     expectedClients[0].ID,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc: "with wrong id",
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				ID:     testsutil.GenerateUUID(t),
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
			desc: "with channel id",
			pm: clients.Page{
				Offset:  0,
				Limit:   nClients,
				Channel: channelID,
				Status:  clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{connectedClient},
			},
		},
		{
			desc: "with order by name ascending",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Order:  "name",
				Dir:    ascDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
		},
		{
			desc: "with order by name descending",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Order:  "name",
				Dir:    descDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
		},
		{
			desc: "with order by identity ascending",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Order:  "identity",
				Dir:    ascDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
		},
		{
			desc: "with order by identity descending",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Order:  "identity",
				Dir:    descDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
		},
		{
			desc: "with order by created_at ascending",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Order:  defOrder,
				Dir:    ascDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
				Clients: expectedClients[:10],
			},
		},
		{
			desc: "with order by created_at descending",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Order:  defOrder,
				Dir:    descDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
				Clients: reversedClients[:10],
			},
		},
		{
			desc: "with order by updated_at ascending",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Order:  "updated_at",
				Dir:    ascDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
		},
		{
			desc: "with order by updated_at descending",
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Order:  "updated_at",
				Dir:    descDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
		},
		{
			desc: "with created_from",
			pm: clients.Page{
				Offset:      0,
				Limit:       nClients,
				Status:      clients.AllStatus,
				CreatedFrom: baseTime.Add(100 * time.Millisecond),
				Order:       defOrder,
				Dir:         ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  100,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients[100:],
			},
		},
		{
			desc: "with created_to",
			pm: clients.Page{
				Offset:    0,
				Limit:     nClients,
				Status:    clients.AllStatus,
				CreatedTo: baseTime.Add(99 * time.Millisecond),
				Order:     defOrder,
				Dir:       ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  100,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients[:100],
			},
		},
		{
			desc: "with both created_from and created_to",
			pm: clients.Page{
				Offset:      0,
				Limit:       nClients,
				Status:      clients.AllStatus,
				CreatedFrom: baseTime.Add(50 * time.Millisecond),
				CreatedTo:   baseTime.Add(149 * time.Millisecond),
				Order:       defOrder,
				Dir:         ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  100,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients[50:150],
			},
		},
		{
			desc: "with created_from  returning no results",
			pm: clients.Page{
				Offset:      0,
				Limit:       nClients,
				Status:      clients.AllStatus,
				CreatedFrom: baseTime.Add(500 * time.Millisecond),
				Order:       defOrder,
				Dir:         ascDir,
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
			desc: "with created_to returning no results",
			pm: clients.Page{
				Offset:    0,
				Limit:     nClients,
				Status:    clients.AllStatus,
				CreatedTo: baseTime.Add(-10 * time.Millisecond),
				Order:     defOrder,
				Dir:       ascDir,
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
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			page, err := repo.RetrieveAll(context.Background(), c.pm)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.response.Total, page.Total)
				assert.Equal(t, c.response.Offset, page.Offset)
				assert.Equal(t, c.response.Limit, page.Limit)
				if len(c.response.Clients) > 0 {
					expected := stripClientDetails(c.response.Clients)
					got := stripClientDetails(page.Clients)
					assert.ElementsMatch(t, expected, got, fmt.Sprintf("expected %v got %v\n", expected, got))
				}
				verifyClientsOrdering(t, page.Clients, c.pm.Order, c.pm.Dir)
			}
		})
	}
}

func TestRetrieveUserClients(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM groups")
		require.Nil(t, err, fmt.Sprintf("clean groups unexpected error: %s", err))
		_, err = db.Exec("DELETE FROM domains")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})

	repo := postgres.NewRepository(database)

	nClients := uint64(10)

	emptyGroupParam := ""
	userID := testsutil.GenerateUUID(t)
	domainMemberID := testsutil.GenerateUUID(t)
	groupMemberID := testsutil.GenerateUUID(t)
	channelID := testsutil.GenerateUUID(t)
	domain := generateDomain(t, userID, domainMemberID)
	group := generateGroup(t, userID, groupMemberID, domain.ID)
	groupClient := clients.Client{}
	parentGroupClient := clients.Client{}
	connectedClient := clients.Client{}
	directClients := []clients.Client{}
	domainClients := []clients.Client{}
	baseTime := time.Now().UTC().Truncate(time.Microsecond)
	for i := range nClients {
		client := clients.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: domain.ID,
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
			CreatedAt: baseTime.Add(time.Duration(i) * time.Microsecond),
			UpdatedAt: baseTime.Add(time.Duration(i) * time.Microsecond),
		}
		if i == 1 {
			client.ParentGroup = group.ID
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))
		newRolesProvision := []roles.RoleProvision{
			{
				Role: roles.Role{
					ID:        testsutil.GenerateUUID(t) + "_" + client.ID,
					Name:      "admin",
					EntityID:  client.ID,
					CreatedAt: validTimestamp,
					CreatedBy: userID,
				},
				OptionalActions: availableActions,
				OptionalMembers: []string{userID},
			},
		}
		npr, err := repo.AddRoles(context.Background(), newRolesProvision)
		require.Nil(t, err, fmt.Sprintf("add roles unexpected error: %s", err))
		directClient := client
		directClient.RoleID = npr[0].Role.ID
		directClient.RoleName = npr[0].Role.Name
		directClient.AccessType = directAccess
		directClient.AccessProviderRoleActions = []string{}
		if i == 1 {
			directClient.ParentGroupPath = group.ID
		}
		directClients = append(directClients, directClient)
		if i == 1 {
			parentGroupClient = directClient
			parentGroupClient.ParentGroupPath = group.ID
			client.ParentGroupPath = group.ID
			groupClient = client
			groupClient.AccessType = directGroupAccess
			groupClient.AccessProviderId = group.ID
			groupClient.AccessProviderRoleId = group.Roles[0].RoleID
			groupClient.AccessProviderRoleName = group.Roles[0].RoleName
			groupClient.AccessProviderRoleActions = groupAvailableActions
		}
		if i == 2 {
			conn := clients.Connection{
				ClientID:  client.ID,
				ChannelID: channelID,
				DomainID:  client.Domain,
				Type:      connections.Publish,
			}
			err = repo.AddConnections(context.Background(), []clients.Connection{conn})
			assert.Nil(t, err, fmt.Sprintf("add connection unexpected error: %s", err))
			connectedClient = client
			connectedClient.RoleID = npr[0].Role.ID
			connectedClient.RoleName = npr[0].Role.Name
			connectedClient.AccessType = directAccess
			connectedClient.AccessProviderRoleActions = []string{}
			connectedClient.ConnectionTypes = []connections.ConnType{connections.Publish}
		}
		domainClient := client
		domainClient.AccessType = domainAccess
		domainClient.AccessProviderId = domain.ID
		domainClient.AccessProviderRoleId = domain.Roles[0].RoleID
		domainClient.AccessProviderRoleName = domain.Roles[0].RoleName
		domainClient.AccessProviderRoleActions = domainAvailableActions
		domainClients = append(domainClients, domainClient)
	}

	reversedDirectClients := []clients.Client{}
	for i := len(directClients) - 1; i >= 0; i-- {
		reversedDirectClients = append(reversedDirectClients, directClients[i])
	}

	cases := []struct {
		desc     string
		domainID string
		userID   string
		pm       clients.Page
		response clients.ClientsPage
		err      error
	}{
		{
			desc:     "retrieve clients with empty page",
			domainID: domain.ID,
			userID:   userID,
			pm:       clients.Page{},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  10,
					Offset: 0,
					Limit:  0,
				},
				Clients: []clients.Client(nil),
			},
		},
		{
			desc:     "retrieve clients with offset and limit",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 5,
				Limit:  10,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 5,
					Limit:  10,
				},
				Clients: directClients[5:10],
			},
		},
		{
			desc:     "retrieve clients with member id of parent group wth direct group access",
			domainID: domain.ID,
			userID:   groupMemberID,
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Clients: []clients.Client{groupClient},
			},
		},
		{
			desc:     "retrieve clients with member id of domain with domain access",
			domainID: domain.ID,
			userID:   domainMemberID,
			pm: clients.Page{
				Offset: 0,
				Limit:  10,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  10,
					Offset: 0,
					Limit:  10,
				},
				Clients: domainClients,
			},
		},
		{
			desc:     "retrieve clients connected to a channel",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset:  0,
				Limit:   10,
				Channel: channelID,
				Status:  clients.AllStatus,
				Order:   defOrder,
				Dir:     ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Clients: []clients.Client{connectedClient},
			},
		},
		{
			desc:     "retrieve clients with offset out of range and limit",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 1000,
				Limit:  50,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve clients with metadata",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				Metadata: directClients[0].Metadata,
				Status:   clients.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{directClients[0]},
			},
		},
		{
			desc:     "retrieve clients with wrong metadata",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Metadata: clients.Metadata{
					"faculty": namegen.Generate(),
				},
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve clients with invalid metadata",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Metadata: clients.Metadata{
					"faculty": make(chan int),
				},
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve clients with name",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Name:   directClients[0].Name,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{directClients[0]},
			},
		},
		{
			desc:     "retrieve clients with wrong name",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Name:   namegen.Generate(),
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve cliens with identity",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				Identity: directClients[0].Credentials.Identity,
				Status:   clients.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{directClients[0]},
			},
		},
		{
			desc:     "retrieve clients with wrong identity",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				Identity: namegen.Generate(),
				Status:   clients.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
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
			desc:     "retrieve clients with tag",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Tags:   clients.TagsQuery{Elements: []string{directClients[0].Tags[0]}, Operator: clients.OrOp},
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  uint64(nClients),
				},
				Clients: []clients.Client{directClients[0]},
			},
		},
		{
			desc:     "retrieve clients with wrong tags",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Tags:   clients.TagsQuery{Elements: []string{namegen.Generate()}, Operator: clients.OrOp},
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve clients with multiple parameters",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				Metadata: directClients[0].Metadata,
				Name:     directClients[0].Name,
				Tags:     clients.TagsQuery{Elements: []string{directClients[0].Tags[0]}, Operator: clients.OrOp},
				Identity: directClients[0].Credentials.Identity,
				Status:   clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{directClients[0]},
			},
		},
		{
			desc:     "retrieve clients with id",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				ID:     directClients[0].ID,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{directClients[0]},
			},
		},
		{
			desc:     "retrieve clients with wrong id",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				ID:     testsutil.GenerateUUID(t),
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve clients with wrong domain id",
			domainID: testsutil.GenerateUUID(t),
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve clients with wrong user id",
			domainID: domain.ID,
			userID:   testsutil.GenerateUUID(t),
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve clients with parent group",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Group:  &group.ID,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{parentGroupClient},
			},
			err: nil,
		},
		{
			desc:     "retrieve clients with no parent group",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				Group:  &emptyGroupParam,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{},
			},
		},
		{
			desc:     "retrieve clients with access type",
			domainID: domain.ID,
			userID:   domainMemberID,
			pm: clients.Page{
				Offset:     0,
				Limit:      10,
				AccessType: domainAccess,
				Status:     clients.AllStatus,
				Order:      defOrder,
				Dir:        ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  10,
					Offset: 0,
					Limit:  10,
				},
				Clients: domainClients,
			},
		},
		{
			desc:     "retrieve clients with wrong access type",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset:     0,
				Limit:      nClients,
				AccessType: domainAccess,
				Status:     clients.AllStatus,
				Order:      defOrder,
				Dir:        ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{},
			},
		},
		{
			desc:     "retrieve clients with role ID",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				RoleID: directClients[0].RoleID,
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []clients.Client{directClients[0]},
			},
		},
		{
			desc:     "retrieve clients with wrong role ID",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  nClients,
				RoleID: testsutil.GenerateUUID(t),
				Status: clients.AllStatus,
				Order:  defOrder,
				Dir:    ascDir,
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
			desc:     "retrieve clients with role name",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset:   0,
				Limit:    1,
				RoleName: directClients[0].RoleName,
				Status:   clients.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  10,
					Offset: 0,
					Limit:  1,
				},
				Clients: directClients[0:1],
			},
		},
		{
			desc:     "retrieve clients with wrong role name",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset:   0,
				Limit:    nClients,
				RoleName: namegen.Generate(),
				Status:   clients.AllStatus,
				Order:    defOrder,
				Dir:      ascDir,
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
			desc:     "retrieve clients with order by name ascending",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  5,
				Order:  "name",
				Dir:    ascDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  5,
				},
			},
		},
		{
			desc:     "retrieve clients with order by name descending",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  5,
				Order:  "name",
				Dir:    descDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  5,
				},
			},
		},
		{
			desc:     "retrieve clients with order by identity ascending",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  5,
				Order:  "identity",
				Dir:    ascDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  5,
				},
			},
		},
		{
			desc:     "retrieve clients with order by identity descending",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  5,
				Order:  "identity",
				Dir:    descDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  5,
				},
			},
		},
		{
			desc:     "retrieve clients with order by created_at ascending",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  5,
				Order:  defOrder,
				Dir:    ascDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  5,
				},
				Clients: directClients[:5],
			},
		},
		{
			desc:     "retrieve clients with order by created_at descending",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  5,
				Order:  defOrder,
				Dir:    descDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  5,
				},
				Clients: reversedDirectClients[:5],
			},
		},
		{
			desc:     "retrieve clients with order by updated_at ascending",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  5,
				Order:  "updated_at",
				Dir:    ascDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  5,
				},
			},
		},
		{
			desc:     "retrieve clients with order by updated_at descending",
			domainID: domain.ID,
			userID:   userID,
			pm: clients.Page{
				Offset: 0,
				Limit:  5,
				Order:  "updated_at",
				Dir:    descDir,
				Status: clients.AllStatus,
			},
			response: clients.ClientsPage{
				Page: clients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  5,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			page, err := repo.RetrieveUserClients(context.Background(), tc.domainID, tc.userID, tc.pm)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected %s to contain %s\n", err, tc.err))
			if err == nil {
				assert.Equal(t, tc.response.Total, page.Total)
				assert.Equal(t, tc.response.Offset, page.Offset)
				assert.Equal(t, tc.response.Limit, page.Limit)
				if len(tc.response.Clients) > 0 {
					expected := stripClientDetails(tc.response.Clients)
					got := stripClientDetails(page.Clients)
					assert.ElementsMatch(t, expected, got, fmt.Sprintf("expected %+v got %+v\n", expected, got))
				}
				verifyClientsOrdering(t, page.Clients, tc.pm.Order, tc.pm.Dir)
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
	baseTime := time.Now().UTC().Truncate(time.Microsecond)
	for i := 0; i < int(nClients); i++ {
		username := name + strconv.Itoa(i) + emailSuffix
		client := clients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: username,
			Credentials: clients.Credentials{
				Identity: username,
				Secret:   testsutil.GenerateUUID(t),
			},
			Metadata: clients.Metadata{
				"department": namegen.Generate(),
			},
			PrivateMetadata: clients.Metadata{},
			Status:          clients.EnabledStatus,
			CreatedAt:       baseTime.Add(time.Duration(i) * time.Microsecond),
		}
		_, err := repo.Save(context.Background(), client)
		require.Nil(t, err, fmt.Sprintf("save client unexpected error: %s", err))

		expectedClients = append(expectedClients, clients.Client{
			ID:        client.ID,
			Name:      client.Name,
			Metadata:  client.Metadata,
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
				Dir:   ascDir,
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
				Dir:    ascDir,
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
				Dir:    ascDir,
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
				Dir:    ascDir,
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
				Dir:      ascDir,
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
				Dir:    ascDir,
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
				Dir:    descDir,
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
				Dir:      ascDir,
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
				Dir:      descDir,
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
	baseTime := time.Now().UTC().Truncate(time.Microsecond)
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
			Metadata:  map[string]any{"name": name},
			CreatedAt: baseTime.Add(time.Duration(i) * time.Microsecond),
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
	baseTime := time.Now().UTC().Truncate(time.Microsecond)
	for i := 0; i < 10; i++ {
		name := namegen.Generate()
		client := clients.Client{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			ParentGroup: parentID,
			Name:        name,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   baseTime.Add(time.Duration(i) * time.Microsecond),
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
	baseTime := time.Now().UTC().Truncate(time.Microsecond)
	for i := 0; i < 10; i++ {
		name := namegen.Generate()
		client := clients.Client{
			ID:          testsutil.GenerateUUID(t),
			Domain:      testsutil.GenerateUUID(t),
			ParentGroup: parentID,
			Name:        name,
			Metadata:    map[string]any{"name": name},
			CreatedAt:   baseTime.Add(time.Duration(i) * time.Microsecond),
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
		PrivateMetadata: clients.Metadata{
			"name": namegen.Generate(),
		},
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

func generateDomain(t *testing.T, userID, memberID string) domains.Domain {
	domain := domains.Domain{
		ID:        testsutil.GenerateUUID(t),
		Route:     namegen.Generate(),
		Status:    domains.EnabledStatus,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
		CreatedBy: userID,
	}

	drepo := dpostgres.NewRepository(ddatabase)
	_, err := drepo.SaveDomain(context.Background(), domain)
	require.Nil(t, err, fmt.Sprintf("add new domain: expected nil got %s\n", err))
	newRolesProvision := []roles.RoleProvision{
		{
			Role: roles.Role{
				ID:        testsutil.GenerateUUID(t) + "_" + domain.ID,
				Name:      "admin",
				EntityID:  domain.ID,
				CreatedAt: validTimestamp,
				CreatedBy: userID,
			},
			OptionalActions: domainAvailableActions,
			OptionalMembers: []string{userID, memberID},
		},
	}
	_, err = drepo.AddRoles(context.Background(), newRolesProvision)
	require.Nil(t, err, fmt.Sprintf("add new role: expected nil got %s\n", err))
	domain.Roles = []roles.MemberRoleActions{
		{
			RoleID:   newRolesProvision[0].Role.ID,
			RoleName: newRolesProvision[0].Role.Name,
			Actions:  newRolesProvision[0].OptionalActions,
		},
	}

	return domain
}

func generateGroup(t *testing.T, userID, memberID, domainID string) groups.Group {
	group := groups.Group{
		ID:        testsutil.GenerateUUID(t),
		Name:      namegen.Generate(),
		Domain:    domainID,
		Status:    groups.EnabledStatus,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}

	grepo := gpostgres.New(gdatabase)
	_, err := grepo.Save(context.Background(), group)
	require.Nil(t, err, fmt.Sprintf("add new domain: expected nil got %s\n", err))
	newRolesProvision := []roles.RoleProvision{
		{
			Role: roles.Role{
				ID:        testsutil.GenerateUUID(t) + "_" + group.ID,
				Name:      "admin",
				EntityID:  group.ID,
				CreatedAt: validTimestamp,
				CreatedBy: userID,
			},
			OptionalActions: groupAvailableActions,
			OptionalMembers: []string{userID, memberID},
		},
	}
	_, err = grepo.AddRoles(context.Background(), newRolesProvision)
	require.Nil(t, err, fmt.Sprintf("add new role: expected nil got %s\n", err))
	group.Roles = []roles.MemberRoleActions{
		{
			RoleID:   newRolesProvision[0].Role.ID,
			RoleName: newRolesProvision[0].Role.Name,
			Actions:  newRolesProvision[0].OptionalActions,
		},
	}

	return group
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
		clients[i].Actions = []string{}
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

func verifyClientsOrdering(t *testing.T, clients []clients.Client, order, dir string) {
	if order == "" || len(clients) <= 1 {
		return
	}

	switch order {
	case "name":
		for i := 1; i < len(clients); i++ {
			if dir == ascDir {
				assert.LessOrEqual(t, clients[i-1].Name, clients[i].Name)
				continue
			}
			assert.GreaterOrEqual(t, clients[i-1].Name, clients[i].Name)
		}
	case "identity":
		for i := 1; i < len(clients); i++ {
			if dir == ascDir {
				assert.LessOrEqual(t, clients[i-1].Credentials.Identity, clients[i].Credentials.Identity)
				continue
			}
			assert.GreaterOrEqual(t, clients[i-1].Credentials.Identity, clients[i].Credentials.Identity)
		}
	case "created_at":
		for i := 1; i < len(clients); i++ {
			if dir == ascDir {
				assert.True(t, !clients[i-1].CreatedAt.After(clients[i].CreatedAt))
				continue
			}
			assert.True(t, !clients[i-1].CreatedAt.Before(clients[i].CreatedAt))
		}
	case "updated_at":
		for i := 1; i < len(clients); i++ {
			if dir == ascDir {
				assert.True(t, !clients[i-1].UpdatedAt.After(clients[i].UpdatedAt))
				continue
			}
			assert.True(t, !clients[i-1].UpdatedAt.Before(clients[i].UpdatedAt))
		}
	}
}
