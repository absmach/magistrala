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
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/things"
	"github.com/absmach/magistrala/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxNameSize = 1024

var (
	invalidName     = strings.Repeat("m", maxNameSize+10)
	thingIdentity   = "thing-identity@example.com"
	thingName       = "thing name"
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
		desc   string
		things []things.Client
		err    error
	}{
		{
			desc: "add new thing successfully",
			things: []things.Client{
				{
					ID:     uid,
					Domain: domainID,
					Name:   thingName,
					Credentials: things.Credentials{
						Identity: thingIdentity,
						Secret:   secret,
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: nil,
		},
		{
			desc: "add multiple things successfully",
			things: []things.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: things.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: things.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: things.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: nil,
		},
		{
			desc: "add new thing with duplicate secret",
			things: []things.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: domainID,
					Name:   namegen.Generate(),
					Credentials: things.Credentials{
						Identity: thingIdentity,
						Secret:   secret,
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add multiple things with one thing having duplicate secret",
			things: []things.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: things.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: domainID,
					Name:   namegen.Generate(),
					Credentials: things.Credentials{
						Identity: thingIdentity,
						Secret:   secret,
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add new thing without domain id",
			things: []things.Client{
				{
					ID:   testsutil.GenerateUUID(t),
					Name: thingName,
					Credentials: things.Credentials{
						Identity: "withoutdomain-thing@example.com",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: nil,
		},
		{
			desc: "add thing with invalid thing id",
			things: []things.Client{
				{
					ID:     invalidName,
					Domain: domainID,
					Name:   thingName,
					Credentials: things.Credentials{
						Identity: "invalidid-thing@example.com",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add multiple things with one thing having invalid thing id",
			things: []things.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: things.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
				{
					ID:     invalidName,
					Domain: testsutil.GenerateUUID(t),
					Name:   namegen.Generate(),
					Credentials: things.Credentials{
						Secret: testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add thing with invalid thing name",
			things: []things.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Name:   invalidName,
					Domain: domainID,
					Credentials: things.Credentials{
						Identity: "invalidname-thing@example.com",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add thing with invalid thing domain id",
			things: []things.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: invalidDomainID,
					Credentials: things.Credentials{
						Identity: "invaliddomainid-thing@example.com",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add thing with invalid thing identity",
			things: []things.Client{
				{
					ID:   testsutil.GenerateUUID(t),
					Name: thingName,
					Credentials: things.Credentials{
						Identity: invalidName,
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
					Status:   things.EnabledStatus,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "add thing with a missing thing identity",
			things: []things.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Name:   "missing-thing-identity",
					Credentials: things.Credentials{
						Identity: "",
						Secret:   testsutil.GenerateUUID(t),
					},
					Metadata: things.Metadata{},
				},
			},
			err: nil,
		},
		{
			desc: "add thing with a missing thing secret",
			things: []things.Client{
				{
					ID:     testsutil.GenerateUUID(t),
					Domain: testsutil.GenerateUUID(t),
					Credentials: things.Credentials{
						Identity: "missing-thing-secret@example.com",
						Secret:   "",
					},
					Metadata: things.Metadata{},
				},
			},
			err: nil,
		},
		{
			desc: "add a thing with invalid metadata",
			things: []things.Client{
				{
					ID:   testsutil.GenerateUUID(t),
					Name: namegen.Generate(),
					Credentials: things.Credentials{
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
		rThings, err := repo.Save(context.Background(), tc.things...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			for i := range rThings {
				tc.things[i].Credentials.Secret = rThings[i].Credentials.Secret
			}
			assert.Equal(t, tc.things, rThings, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.things, rThings))
		}
	}
}

func TestThingsRetrieveBySecret(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := postgres.NewRepository(database)

	thing := things.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: thingName,
		Credentials: things.Credentials{
			Identity: thingIdentity,
			Secret:   testsutil.GenerateUUID(t),
		},
		Metadata: things.Metadata{},
		Status:   things.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		secret   string
		response things.Client
		err      error
	}{
		{
			desc:     "retrieve thing by secret successfully",
			secret:   thing.Credentials.Secret,
			response: thing,
			err:      nil,
		},
		{
			desc:     "retrieve thing by invalid secret",
			secret:   "non-existent-secret",
			response: things.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve thing by empty secret",
			secret:   "",
			response: things.Client{},
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

	thing := things.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: thingName,
		Credentials: things.Credentials{
			Identity: thingIdentity,
			Secret:   testsutil.GenerateUUID(t),
		},
		Metadata: things.Metadata{},
		Status:   things.EnabledStatus,
	}

	_, err := repo.Save(context.Background(), thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		id       string
		response things.Client
		err      error
	}{
		{
			desc:     "successfully",
			id:       thing.ID,
			response: thing,
			err:      nil,
		},
		{
			desc:     "with invalid id",
			id:       testsutil.GenerateUUID(t),
			response: things.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "with empty id",
			id:       "",
			response: things.Client{},
			err:      repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cli, err := repo.RetrieveByID(context.Background(), c.id)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s got %s\n", c.err, err))
			if err == nil {
				assert.Equal(t, thing.ID, cli.ID)
				assert.Equal(t, thing.Name, cli.Name)
				assert.Equal(t, thing.Metadata, cli.Metadata)
				assert.Equal(t, thing.Credentials.Identity, cli.Credentials.Identity)
				assert.Equal(t, thing.Credentials.Secret, cli.Credentials.Secret)
				assert.Equal(t, thing.Status, cli.Status)
			}
		})
	}
}
