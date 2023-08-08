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
	cpostgres "github.com/mainflux/mainflux/things/clients/postgres"
	"github.com/stretchr/testify/assert"
)

const maxNameSize = 1024

var (
	idProvider     = uuid.New()
	invalidName    = strings.Repeat("m", maxNameSize+10)
	clientIdentity = "client-identity@example.com"
	clientName     = "client name"
)

func TestClientsSave(t *testing.T) {
	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
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
					Secret:   testsutil.GenerateUUID(t, idProvider),
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
					Secret:   testsutil.GenerateUUID(t, idProvider),
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "add client with invalid client id",
			client: mfclients.Client{
				ID:   invalidName,
				Name: clientName,
				Credentials: mfclients.Credentials{
					Identity: "invalidid-client@example.com",
					Secret:   testsutil.GenerateUUID(t, idProvider),
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client name",
			client: mfclients.Client{
				ID:   testsutil.GenerateUUID(t, idProvider),
				Name: invalidName,
				Credentials: mfclients.Credentials{
					Identity: "invalidname-client@example.com",
					Secret:   testsutil.GenerateUUID(t, idProvider),
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client owner",
			client: mfclients.Client{
				ID:    testsutil.GenerateUUID(t, idProvider),
				Owner: invalidName,
				Credentials: mfclients.Credentials{
					Identity: "invalidowner-client@example.com",
					Secret:   testsutil.GenerateUUID(t, idProvider),
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "add client with invalid client identity",
			client: mfclients.Client{
				ID:   testsutil.GenerateUUID(t, idProvider),
				Name: clientName,
				Credentials: mfclients.Credentials{
					Identity: invalidName,
					Secret:   testsutil.GenerateUUID(t, idProvider),
				},
				Metadata: mfclients.Metadata{},
				Status:   mfclients.EnabledStatus,
			},
			err: errors.ErrCreateEntity,
		},
		{
			desc: "add client with a missing client identity",
			client: mfclients.Client{
				ID: testsutil.GenerateUUID(t, idProvider),
				Credentials: mfclients.Credentials{
					Identity: "",
					Secret:   testsutil.GenerateUUID(t, idProvider),
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
			rClient[0].Credentials.Secret = tc.client.Credentials.Secret
			assert.Equal(t, tc.client, rClient[0], fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, rClient[0]))
		}
	}
}
