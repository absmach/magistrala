// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/absmach/magistrala/internal/testsutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	cpostgres "github.com/absmach/magistrala/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxNameSize = 254
)

var (
	invalidName    = strings.Repeat("m", maxNameSize+10)
	password       = "$tr0ngPassw0rd"
	clientIdentity = "client-identity@example.com"
	clientName     = "client name"
)

func TestClientsSave(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	uid := testsutil.GenerateUUID(t)

	cases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "add new client successfully",
			client: mgclients.Client{
				ID:   uid,
				Name: clientName,
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
				Name:  clientName,
				Credentials: mgclients.Credentials{
					Identity: "withowner-client@example.com",
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
				Name: clientName,
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
				Name: clientName,
				Credentials: mgclients.Credentials{
					Identity: "invalidid-client@example.com",
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
					Identity: "invalidname-client@example.com",
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
					Identity: "invalidowner-client@example.com",
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
				Name: clientName,
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
			desc: "add client with a missing client identity",
			client: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: "",
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add client with a missing client secret",
			client: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: "missing-client-secret@example.com",
					Secret:   "",
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

func TestIsOwner(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := cpostgres.NewRepository(database)

	owner := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "owner",
		Credentials: mgclients.Credentials{
			Identity: "owner@example.com",
			Secret:   password,
		},
		Metadata: mgclients.Metadata{},
		Status:   mgclients.EnabledStatus,
	}
	owner, err := repo.Save(context.Background(), owner)
	require.Nil(t, err, fmt.Sprintf("save owner unexpected error: %s", err))

	cases := []struct {
		desc    string
		ownerID string
		client  mgclients.Client
		err     error
	}{
		{
			desc:    "add new client successfully with an owner",
			ownerID: owner.ID,
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: clientName,
				Credentials: mgclients.Credentials{
					Identity: "withowner@example.com",
					Secret:   password,
				},
				Owner:    owner.ID,
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc:    "add new client successfully without an owner",
			ownerID: owner.ID,
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: clientName,
				Credentials: mgclients.Credentials{
					Identity: "withoutowner@example.com",
					Secret:   password,
				},
				Metadata: mgclients.Metadata{},
				Status:   mgclients.EnabledStatus,
			},
			err: errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.client)
		require.Nil(t, err, fmt.Sprintf("%s: save client unexpected error: %s", tc.desc, err))
		err = repo.IsOwner(context.Background(), tc.client.ID, tc.ownerID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
