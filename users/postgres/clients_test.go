// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mainflux/mainflux/internal/testsutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	cpostgres "github.com/mainflux/mainflux/users/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxNameSize = 254
)

var (
	idProvider     = uuid.New()
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

	uid := testsutil.GenerateUUID(t, idProvider)

	cases := []struct {
		desc   string
		client mfclients.Client
		err    error
	}{
		{
			desc: "add new client successfully",
			client: mfclients.Client{
				ID:   uid,
				Name: clientName,
				Credentials: mfclients.Credentials{
					Identity: clientIdentity,
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add new client with an owner",
			client: mfclients.Client{
				ID:    testsutil.GenerateUUID(t, idProvider),
				Owner: uid,
				Name:  clientName,
				Credentials: mfclients.Credentials{
					Identity: "withowner-client@example.com",
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add client with duplicate client identity",
			client: mfclients.Client{
				ID:   testsutil.GenerateUUID(t, idProvider),
				Name: clientName,
				Credentials: mfclients.Credentials{
					Identity: clientIdentity,
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrConflict,
		},
		{
			desc: "add client with invalid client id",
			client: mfclients.Client{
				ID:   invalidName,
				Name: clientName,
				Credentials: mfclients.Credentials{
					Identity: "invalidid-client@example.com",
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with invalid client name",
			client: mfclients.Client{
				ID:   testsutil.GenerateUUID(t, idProvider),
				Name: invalidName,
				Credentials: mfclients.Credentials{
					Identity: "invalidname-client@example.com",
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with invalid client owner",
			client: mfclients.Client{
				ID:    testsutil.GenerateUUID(t, idProvider),
				Owner: invalidName,
				Credentials: mfclients.Credentials{
					Identity: "invalidowner-client@example.com",
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with invalid client identity",
			client: mfclients.Client{
				ID:   testsutil.GenerateUUID(t, idProvider),
				Name: clientName,
				Credentials: mfclients.Credentials{
					Identity: invalidName,
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "add client with a missing client identity",
			client: mfclients.Client{
				ID: testsutil.GenerateUUID(t, idProvider),
				Credentials: mfclients.Credentials{
					Identity: "",
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
			},
			err: nil,
		},
		{
			desc: "add client with a missing client secret",
			client: mfclients.Client{
				ID: testsutil.GenerateUUID(t, idProvider),
				Credentials: mfclients.Credentials{
					Identity: "missing-client-secret@example.com",
					Secret:   "",
				},
				Metadata: mfclients.Metadata{},
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

	owner := mfclients.Client{
		ID:   testsutil.GenerateUUID(t, idProvider),
		Name: "owner",
		Credentials: mfclients.Credentials{
			Identity: "owner@example.com",
			Secret:   password,
		},
		Metadata: mfclients.Metadata{},
		Status:   mfclients.EnabledStatus,
	}
	owner, err := repo.Save(context.Background(), owner)
	require.Nil(t, err, fmt.Sprintf("save owner unexpected error: %s", err))

	cases := []struct {
		desc    string
		ownerID string
		client  mfclients.Client
		err     error
	}{
		{
			desc:    "add new client successfully with an owner",
			ownerID: owner.ID,
			client: mfclients.Client{
				ID:   testsutil.GenerateUUID(t, idProvider),
				Name: clientName,
				Credentials: mfclients.Credentials{
					Identity: "withowner@example.com",
					Secret:   password,
				},
				Owner:    owner.ID,
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc:    "add new client successfully without an owner",
			ownerID: owner.ID,
			client: mfclients.Client{
				ID:   testsutil.GenerateUUID(t, idProvider),
				Name: clientName,
				Credentials: mfclients.Credentials{
					Identity: "withoutowner@example.com",
					Secret:   password,
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
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
