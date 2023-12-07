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
