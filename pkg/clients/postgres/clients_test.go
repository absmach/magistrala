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
	"github.com/absmach/magistrala/internal/testsutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	pgclients "github.com/absmach/magistrala/pkg/clients/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	ipostgres "github.com/absmach/magistrala/pkg/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	password    = "$tr0ngPassw0rd"
	emailSuffix = "@example.com"
	namegen     = namegenerator.NewGenerator()
)

func TestRetrieveByID(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)

	cases := []struct {
		desc     string
		id       string
		response mgclients.Client
		err      error
	}{
		{
			desc:     "successfully",
			id:       client.ID,
			response: client,
			err:      nil,
		},
		{
			desc:     "with invalid user id",
			id:       testsutil.GenerateUUID(t),
			response: mgclients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "with empty user id",
			id:       "",
			response: mgclients.Client{},
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

func TestRetrieveByIdentity(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)

	cases := []struct {
		desc     string
		identity string
		response mgclients.Client
		err      error
	}{
		{
			desc:     "successfully",
			identity: client.Credentials.Identity,
			response: client,
			err:      nil,
		},
		{
			desc:     "with invalid user id",
			identity: testsutil.GenerateUUID(t),
			response: mgclients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "with empty user id",
			identity: "",
			response: mgclients.Client{},
			err:      repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			cli, err := repo.RetrieveByIdentity(context.Background(), c.identity)
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

func TestRetrieveAll(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	nClients := uint64(200)

	expectedClients := []mgclients.Client{}
	disabledClients := []mgclients.Client{}
	for i := uint64(0); i < nClients; i++ {
		client := mgclients.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: testsutil.GenerateUUID(t),
			Name:   namegen.Generate(),
			Credentials: mgclients.Credentials{
				Identity: namegen.Generate() + emailSuffix,
				Secret:   password,
			},
			Tags: namegen.GenerateMultiple(5),
			Metadata: mgclients.Metadata{
				"department": namegen.Generate(),
			},
			Status:    mgclients.EnabledStatus,
			CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
			Role:      mgclients.UserRole,
		}
		if i%50 == 0 {
			client.Status = mgclients.DisabledStatus
			client.Role = mgclients.AdminRole
		}
		client, err := save(context.Background(), repo, client)
		require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))
		expectedClients = append(expectedClients, client)
		if client.Status == mgclients.DisabledStatus {
			disabledClients = append(disabledClients, client)
		}
	}

	cases := []struct {
		desc     string
		pm       mgclients.Page
		response mgclients.ClientsPage
		err      error
	}{
		{
			desc: "with empty page",
			pm:   mgclients.Page{},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  196,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with offset only",
			pm: mgclients.Page{
				Offset: 50,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 50,
					Limit:  0,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with limit only",
			pm: mgclients.Page{
				Limit:  50,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  50,
				},
				Clients: expectedClients[0:50],
			},
		},
		{
			desc: "retrieve all clients",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with offset and limit",
			pm: mgclients.Page{
				Offset: 50,
				Limit:  50,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 50,
					Limit:  50,
				},
				Clients: expectedClients[50:100],
			},
		},
		{
			desc: "with offset out of range and limit",
			pm: mgclients.Page{
				Offset: 1000,
				Limit:  50,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 1000,
					Limit:  50,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with offset and limit out of range",
			pm: mgclients.Page{
				Offset: 170,
				Limit:  50,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 170,
					Limit:  50,
				},
				Clients: expectedClients[170:200],
			},
		},
		{
			desc: "with metadata",
			pm: mgclients.Page{
				Offset:   0,
				Limit:    nClients,
				Metadata: expectedClients[0].Metadata,
				Status:   mgclients.AllStatus,
				Role:     mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong metadata",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Metadata: mgclients.Metadata{
					"faculty": namegen.Generate(),
				},
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with invalid metadata",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Metadata: mgclients.Metadata{
					"faculty": make(chan int),
				},
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  uint64(nClients),
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
			err: repoerr.ErrViewEntity,
		},
		{
			desc: "with name",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Name:   expectedClients[0].Name,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong name",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Name:   namegen.Generate(),
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with identity",
			pm: mgclients.Page{
				Offset:   0,
				Limit:    nClients,
				Identity: expectedClients[0].Credentials.Identity,
				Status:   mgclients.AllStatus,
				Role:     mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong identity",
			pm: mgclients.Page{
				Offset:   0,
				Limit:    nClients,
				Identity: namegen.Generate(),
				Status:   mgclients.AllStatus,
				Role:     mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with domain",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Domain: expectedClients[0].Domain,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong domain",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Domain: testsutil.GenerateUUID(t),
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with enabled status",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  10,
				Status: mgclients.EnabledStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  196,
					Offset: 0,
					Limit:  10,
				},
				Clients: expectedClients[1:11],
			},
		},
		{
			desc: "with disabled status",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: mgclients.DisabledStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: disabledClients,
			},
		},
		{
			desc: "with combined status",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with the wrong status",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: 10,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with user role",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  10,
				Role:   mgclients.UserRole,
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  196,
					Offset: 0,
					Limit:  10,
				},
				Clients: expectedClients[1:11],
			},
		},
		{
			desc: "with admin role",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Role:   mgclients.AdminRole,
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: disabledClients,
			},
		},
		{
			desc: "with combined role",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with the wrong role",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Role:   10,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with tag",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Tag:    expectedClients[0].Tags[0],
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  uint64(nClients),
				},
				Clients: []mgclients.Client{expectedClients[0]},
			},
		},
		{
			desc: "with wrong tags",
			pm: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
				Tag:    namegen.Generate(),
				Status: mgclients.AllStatus,
				Role:   mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with multiple parameters",
			pm: mgclients.Page{
				Offset:   0,
				Limit:    nClients,
				Metadata: expectedClients[0].Metadata,
				Name:     expectedClients[0].Name,
				Tag:      expectedClients[0].Tags[0],
				Identity: expectedClients[0].Credentials.Identity,
				Domain:   expectedClients[0].Domain,
				Status:   mgclients.AllStatus,
				Role:     mgclients.AllRole,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client{expectedClients[0]},
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
				assert.ElementsMatch(t, page.Clients, c.response.Clients)
			}
		})
	}
}

func TestRetrieveByIDs(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	num := 200

	var items []mgclients.Client
	for i := 0; i < num; i++ {
		name := namegen.Generate()
		client := mgclients.Client{
			ID:     testsutil.GenerateUUID(t),
			Domain: testsutil.GenerateUUID(t),
			Name:   name,
			Credentials: mgclients.Credentials{
				Identity: name + emailSuffix,
				Secret:   password,
			},
			Tags:      namegen.GenerateMultiple(5),
			Metadata:  map[string]interface{}{"name": name},
			CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
			Status:    mgclients.EnabledStatus,
		}
		client, err := save(context.Background(), repo, client)
		require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))
		items = append(items, client)
	}

	page, err := repo.RetrieveAll(context.Background(), mgclients.Page{Offset: 0, Limit: uint64(num)})
	require.Nil(t, err, fmt.Sprintf("retrieve all clients unexpected error: %s", err))
	assert.Equal(t, uint64(num), page.Total)

	cases := []struct {
		desc     string
		page     mgclients.Page
		response mgclients.ClientsPage
		err      error
	}{
		{
			desc: "successfully",
			page: mgclients.Page{
				Offset: 0,
				Limit:  10,
				IDs:    getIDs(items[0:3]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  3,
					Offset: 0,
					Limit:  10,
				},
				Clients: items[0:3],
			},
			err: nil,
		},
		{
			desc: "with empty ids",
			page: mgclients.Page{
				Offset: 0,
				Limit:  10,
				IDs:    []string{},
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  10,
				},
				Clients: []mgclients.Client(nil),
			},
			err: nil,
		},
		{
			desc: "with empty ids but with domain id",
			page: mgclients.Page{
				Offset: 0,
				Limit:  10,
				Domain: items[0].Domain,
				IDs:    []string{},
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Clients: []mgclients.Client{items[0]},
			},
			err: nil,
		},
		{
			desc: "with offset only",
			page: mgclients.Page{
				Offset: 10,
				IDs:    getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  20,
					Offset: 10,
					Limit:  0,
				},
				Clients: []mgclients.Client(nil),
			},
			err: nil,
		},
		{
			desc: "with limit only",
			page: mgclients.Page{
				Limit: 10,
				IDs:   getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  20,
					Offset: 0,
					Limit:  10,
				},
				Clients: items[0:10],
			},
			err: nil,
		},
		{
			desc: "with offset out of range",
			page: mgclients.Page{
				Offset: 1000,
				Limit:  50,
				IDs:    getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  20,
					Offset: 1000,
					Limit:  50,
				},
				Clients: []mgclients.Client(nil),
			},
			err: nil,
		},
		{
			desc: "with offset and limit out of range",
			page: mgclients.Page{
				Offset: 15,
				Limit:  10,
				IDs:    getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  20,
					Offset: 15,
					Limit:  10,
				},
				Clients: items[15:20],
			},
			err: nil,
		},
		{
			desc: "with limit out of range",
			page: mgclients.Page{
				Offset: 0,
				Limit:  1000,
				IDs:    getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  20,
					Offset: 0,
					Limit:  1000,
				},
				Clients: items[:20],
			},
			err: nil,
		},
		{
			desc: "with name",
			page: mgclients.Page{
				Offset: 0,
				Limit:  10,
				Name:   items[0].Name,
				IDs:    getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Clients: []mgclients.Client{items[0]},
			},
			err: nil,
		},
		{
			desc: "with domain id",
			page: mgclients.Page{
				Offset: 0,
				Limit:  10,
				Domain: items[0].Domain,
				IDs:    getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Clients: []mgclients.Client{items[0]},
			},
			err: nil,
		},
		{
			desc: "with metadata",
			page: mgclients.Page{
				Offset:   0,
				Limit:    10,
				Metadata: items[0].Metadata,
				IDs:      getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
				Clients: []mgclients.Client{items[0]},
			},
			err: nil,
		},
		{
			desc: "with invalid metadata",
			page: mgclients.Page{
				Offset: 0,
				Limit:  10,
				Metadata: map[string]interface{}{
					"key": make(chan int),
				},
				IDs: getIDs(items[0:20]),
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Clients: []mgclients.Client(nil),
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, c := range cases {
		switch response, err := repo.RetrieveAllByIDs(context.Background(), c.page); {
		case err == nil:
			assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s\n", c.desc, c.err, err))
			assert.Equal(t, c.response.Total, response.Total)
			assert.Equal(t, c.response.Limit, response.Limit)
			assert.Equal(t, c.response.Offset, response.Offset)
			assert.ElementsMatch(t, response.Clients, c.response.Clients)
		default:
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
		}
	}
}

func TestSearchClients(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	name := namegen.Generate()

	nClients := uint64(200)
	expectedClients := []mgclients.Client{}
	for i := 0; i < int(nClients); i++ {
		username := name + strconv.Itoa(i) + emailSuffix
		client := mgclients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: username,
			Credentials: mgclients.Credentials{
				Identity: username,
				Secret:   password,
			},
			Metadata:  mgclients.Metadata{},
			Status:    mgclients.EnabledStatus,
			CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
		}
		client, err := save(context.Background(), repo, client)
		require.Nil(t, err, fmt.Sprintf("save client unexpected error: %s", err))

		expectedClients = append(expectedClients, mgclients.Client{
			ID:        client.ID,
			Name:      client.Name,
			CreatedAt: client.CreatedAt,
		})
	}

	page, err := repo.RetrieveAll(context.Background(), mgclients.Page{Offset: 0, Limit: nClients})
	require.Nil(t, err, fmt.Sprintf("retrieve all clients unexpected error: %s", err))
	assert.Equal(t, nClients, page.Total)

	cases := []struct {
		desc     string
		page     mgclients.Page
		response mgclients.ClientsPage
		err      error
	}{
		{
			desc: "with empty page",
			page: mgclients.Page{},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client(nil),
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  0,
				},
			},
			err: nil,
		},
		{
			desc: "with offset only",
			page: mgclients.Page{
				Offset: 50,
			},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client(nil),
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 50,
					Limit:  0,
				},
			},
			err: nil,
		},
		{
			desc: "with limit only",
			page: mgclients.Page{
				Limit: 10,
				Order: "name",
				Dir:   "asc",
			},
			response: mgclients.ClientsPage{
				Clients: expectedClients[0:10],
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve all clients",
			page: mgclients.Page{
				Offset: 0,
				Limit:  nClients,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: expectedClients,
			},
		},
		{
			desc: "with offset and limit",
			page: mgclients.Page{
				Offset: 10,
				Limit:  10,
				Order:  "name",
				Dir:    "asc",
			},
			response: mgclients.ClientsPage{
				Clients: expectedClients[10:20],
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 10,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with offset out of range and limit",
			page: mgclients.Page{
				Offset: 1000,
				Limit:  50,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 1000,
					Limit:  50,
				},
				Clients: []mgclients.Client(nil),
			},
		},
		{
			desc: "with offset and limit out of range",
			page: mgclients.Page{
				Offset: 190,
				Limit:  50,
				Order:  "name",
				Dir:    "asc",
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 190,
					Limit:  50,
				},
				Clients: expectedClients[190:200],
			},
		},
		{
			desc: "with shorter name",
			page: mgclients.Page{
				Name:   expectedClients[0].Name[:4],
				Offset: 0,
				Limit:  10,
				Order:  "name",
				Dir:    "asc",
			},
			response: mgclients.ClientsPage{
				Clients: findClients(expectedClients, expectedClients[0].Name[:4], 0, 10),
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with longer name",
			page: mgclients.Page{
				Name:   expectedClients[0].Name,
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client{expectedClients[0]},
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with name SQL injected",
			page: mgclients.Page{
				Name:   fmt.Sprintf("%s' OR '1'='1", expectedClients[0].Name[:1]),
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
			desc: "with shorter Identity",
			page: mgclients.Page{
				Identity: expectedClients[0].Name[:4],
				Offset:   0,
				Limit:    10,
				Order:    "name",
				Dir:      "asc",
			},
			response: mgclients.ClientsPage{
				Clients: findClients(expectedClients, expectedClients[0].Name[:4], 0, 10),
				Page: mgclients.Page{
					Total:  nClients,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with longer Identity",
			page: mgclients.Page{
				Identity: expectedClients[0].Name,
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{
				Clients: []mgclients.Client{expectedClients[0]},
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  10,
				},
			},
			err: nil,
		},
		{
			desc: "with Identity SQL injected",
			page: mgclients.Page{
				Identity: fmt.Sprintf("%s' OR '1'='1", expectedClients[0].Name[:1]),
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
			desc: "with unknown name",
			page: mgclients.Page{
				Name:   namegen.Generate(),
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
			desc: "with unknown name SQL injected",
			page: mgclients.Page{
				Name:   fmt.Sprintf("%s' OR '1'='1", namegen.Generate()),
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
			desc: "with unknown identity",
			page: mgclients.Page{
				Identity: namegen.Generate(),
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
			desc: "with name in asc order",
			page: mgclients.Page{
				Order:  "name",
				Dir:    "asc",
				Name:   expectedClients[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "with name in desc order",
			page: mgclients.Page{
				Order:  "name",
				Dir:    "desc",
				Name:   expectedClients[0].Name[:1],
				Offset: 0,
				Limit:  10,
			},
			response: mgclients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "with identity in asc order",
			page: mgclients.Page{
				Order:    "identity",
				Dir:      "asc",
				Identity: expectedClients[0].Name[:1],
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{},
			err:      nil,
		},
		{
			desc: "with identity in desc order",
			page: mgclients.Page{
				Order:    "identity",
				Dir:      "desc",
				Identity: expectedClients[0].Name[:1],
				Offset:   0,
				Limit:    10,
			},
			response: mgclients.ClientsPage{},
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

func TestUpdate(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client1 := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)
	client2 := generateClient(t, mgclients.DisabledStatus, mgclients.UserRole, repo)

	cases := []struct {
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
					"update": namegen.Generate(),
				},
			},
			err: nil,
		},
		{
			desc:   "update malformed metadata for enabled client",
			update: "metadata",
			client: mgclients.Client{
				ID: client1.ID,
				Metadata: mgclients.Metadata{
					"update": make(chan int),
				},
			},
			err: repoerr.ErrUpdateEntity,
		},
		{
			desc:   "update metadata for disabled client",
			update: "metadata",
			client: mgclients.Client{
				ID: client2.ID,
				Metadata: mgclients.Metadata{
					"update": namegen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update name for enabled client",
			update: "name",
			client: mgclients.Client{
				ID:   client1.ID,
				Name: namegen.Generate(),
			},
			err: nil,
		},
		{
			desc:   "update name for disabled client",
			update: "name",
			client: mgclients.Client{
				ID:   client2.ID,
				Name: namegen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update name and metadata for enabled client",
			update: "both",
			client: mgclients.Client{
				ID:   client1.ID,
				Name: namegen.Generate(),
				Metadata: mgclients.Metadata{
					"update": namegen.Generate(),
				},
			},
			err: nil,
		},
		{
			desc:   "update name and metadata for a disabled client",
			update: "both",
			client: mgclients.Client{
				ID:   client2.ID,
				Name: namegen.Generate(),
				Metadata: mgclients.Metadata{
					"update": namegen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update metadata for invalid client",
			update: "metadata",
			client: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Metadata: mgclients.Metadata{
					"update": namegen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update name for invalid client",
			update: "name",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update name and metadata for invalid client",
			update: "both",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Name: namegen.Generate(),
				Metadata: mgclients.Metadata{
					"update": namegen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update metadata for empty client",
			update: "metadata",
			client: mgclients.Client{
				Metadata: mgclients.Metadata{
					"update": namegen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update name for empty client",
			update: "name",
			client: mgclients.Client{
				Name: namegen.Generate(),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "update name and metadata for empty client",
			update: "both",
			client: mgclients.Client{
				Name: namegen.Generate(),
				Metadata: mgclients.Metadata{
					"update": namegen.Generate(),
				},
			},
			err: repoerr.ErrNotFound,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.client.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.client.UpdatedBy = testsutil.GenerateUUID(t)
			expected, err := repo.Update(context.Background(), c.client)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				switch c.update {
				case "metadata":
					assert.Equal(t, c.client.Metadata, expected.Metadata)
				case "name":
					assert.Equal(t, c.client.Name, expected.Name)
				case "both":
					assert.Equal(t, c.client.Metadata, expected.Metadata)
					assert.Equal(t, c.client.Name, expected.Name)
				}
				assert.Equal(t, c.client.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.client.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestUpdateTags(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client1 := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)
	client2 := generateClient(t, mgclients.DisabledStatus, mgclients.UserRole, repo)

	cases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "for enabled client",
			client: mgclients.Client{
				ID:   client1.ID,
				Tags: namegen.GenerateMultiple(5),
			},
			err: nil,
		},
		{
			desc: "for disabled client",
			client: mgclients.Client{
				ID:   client2.ID,
				Tags: namegen.GenerateMultiple(5),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid client",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Tags: namegen.GenerateMultiple(5),
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: mgclients.Client{},
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

func TestUpdateSecret(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client1 := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)
	client2 := generateClient(t, mgclients.DisabledStatus, mgclients.UserRole, repo)

	cases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "for enabled client",
			client: mgclients.Client{
				ID: client1.ID,
				Credentials: mgclients.Credentials{
					Secret: "newpassword",
				},
			},
			err: nil,
		},
		{
			desc: "for disabled client",
			client: mgclients.Client{
				ID: client2.ID,
				Credentials: mgclients.Credentials{
					Secret: "newpassword",
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid client",
			client: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Secret: "newpassword",
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: mgclients.Client{},
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

func TestUpdateIdentity(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client1 := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)
	client2 := generateClient(t, mgclients.DisabledStatus, mgclients.UserRole, repo)

	cases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "for enabled client",
			client: mgclients.Client{
				ID: client1.ID,
				Credentials: mgclients.Credentials{
					Identity: namegen.Generate() + emailSuffix,
				},
			},
			err: nil,
		},
		{
			desc: "for disabled client",
			client: mgclients.Client{
				ID: client2.ID,
				Credentials: mgclients.Credentials{
					Identity: namegen.Generate() + emailSuffix,
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid client",
			client: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: namegen.Generate() + emailSuffix,
				},
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: mgclients.Client{},
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

func TestChangeStatus(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client1 := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)
	client2 := generateClient(t, mgclients.DisabledStatus, mgclients.UserRole, repo)

	cases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "for an enabled client",
			client: mgclients.Client{
				ID:     client1.ID,
				Status: mgclients.DisabledStatus,
			},
			err: nil,
		},
		{
			desc: "for a disabled client",
			client: mgclients.Client{
				ID:     client2.ID,
				Status: mgclients.EnabledStatus,
			},
			err: nil,
		},
		{
			desc: "for invalid client",
			client: mgclients.Client{
				ID:     testsutil.GenerateUUID(t),
				Status: mgclients.DisabledStatus,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: mgclients.Client{},
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

func TestUpdateRole(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client1 := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)
	client2 := generateClient(t, mgclients.DisabledStatus, mgclients.UserRole, repo)

	cases := []struct {
		desc   string
		client mgclients.Client
		err    error
	}{
		{
			desc: "for an enabled client",
			client: mgclients.Client{
				ID:   client1.ID,
				Role: mgclients.AdminRole,
			},
			err: nil,
		},
		{
			desc: "for a disabled client",
			client: mgclients.Client{
				ID:   client2.ID,
				Role: mgclients.AdminRole,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "for invalid client",
			client: mgclients.Client{
				ID:   testsutil.GenerateUUID(t),
				Role: mgclients.AdminRole,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc:   "for empty client",
			client: mgclients.Client{},
			err:    repoerr.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			c.client.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
			c.client.UpdatedBy = testsutil.GenerateUUID(t)
			expected, err := repo.UpdateRole(context.Background(), c.client)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected %s to contain %s\n", err, c.err))
			if err == nil {
				assert.Equal(t, c.client.Role, expected.Role)
				assert.Equal(t, c.client.UpdatedAt, expected.UpdatedAt)
				assert.Equal(t, c.client.UpdatedBy, expected.UpdatedBy)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM clients")
		require.Nil(t, err, fmt.Sprintf("clean clients unexpected error: %s", err))
	})
	repo := &pgclients.Repository{database}

	client := generateClient(t, mgclients.EnabledStatus, mgclients.UserRole, repo)

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

func findClients(clis []mgclients.Client, query string, offset, limit uint64) []mgclients.Client {
	rclients := []mgclients.Client{}
	for _, client := range clis {
		if strings.Contains(client.Name, query) {
			rclients = append(rclients, client)
		}
	}

	if offset > uint64(len(rclients)) {
		return []mgclients.Client{}
	}

	if limit > uint64(len(rclients)) {
		return rclients[offset:]
	}

	return rclients[offset:limit]
}

func generateClient(t *testing.T, status mgclients.Status, role mgclients.Role, repo *pgclients.Repository) mgclients.Client {
	client := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: namegen.Generate(),
		Credentials: mgclients.Credentials{
			Identity: namegen.Generate() + emailSuffix,
			Secret:   password,
		},
		Tags: namegen.GenerateMultiple(5),
		Metadata: mgclients.Metadata{
			"name": namegen.Generate(),
		},
		Status:    status,
		Role:      role,
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	_, err := save(context.Background(), repo, client)
	require.Nil(t, err, fmt.Sprintf("add new client: expected nil got %s\n", err))

	return client
}

func save(ctx context.Context, repo *pgclients.Repository, c mgclients.Client) (mgclients.Client, error) {
	q := `INSERT INTO clients (id, name, tags, domain_id, identity, secret, metadata, created_at, status, role)
        VALUES (:id, :name, :tags, :domain_id, :identity, :secret, :metadata, :created_at, :status, :role)
        RETURNING id, name, tags, identity, metadata, COALESCE(domain_id, '') AS domain_id, status, created_at`
	dbc, err := pgclients.ToDBClient(c)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	row, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return mgclients.Client{}, ipostgres.HandleError(repoerr.ErrCreateEntity, err)
	}
	defer row.Close()

	dbc = pgclients.DBClient{}
	if row.Next() {
		if err := row.StructScan(&dbc); err != nil {
			return mgclients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return pgclients.ToClient(dbc)
	}

	return mgclients.Client{}, repoerr.ErrCreateEntity
}

func getIDs(clis []mgclients.Client) []string {
	var ids []string
	for _, client := range clis {
		ids = append(ids, client.ID)
	}

	return ids
}
