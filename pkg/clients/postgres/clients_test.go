// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

// import (
// 	"context"
// 	"fmt"
// 	"testing"

// 	gpostgres "github.com/mainflux/mainflux/internal/groups/postgres"
// 	"github.com/mainflux/mainflux/internal/testsutil"
// 	mfclients "github.com/mainflux/mainflux/pkg/clients"
// 	"github.com/mainflux/mainflux/pkg/errors"
// 	mfgroups "github.com/mainflux/mainflux/pkg/groups"
// 	"github.com/mainflux/mainflux/pkg/uuid"
// 	ppostgres "github.com/mainflux/mainflux/users/policies/postgres"
// 	cpostgres "github.com/mainflux/mainflux/users/postgres"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// var (
// 	idProvider     = uuid.New()
// 	password       = "$tr0ngPassw0rd"
// 	clientIdentity = "client-identity@example.com"
// 	clientName     = "client name"
// 	wrongName      = "wrong-name"
// 	wrongID        = "wrong-id"
// )

// func TestClientsRetrieveByID(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)

// 	client := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: clientName,
// 		Credentials: mfclients.Credentials{
// 			Identity: clientIdentity,
// 			Secret:   password,
// 		},
// 		Status: mfclients.EnabledStatus,
// 	}

// 	clients, err := repo.Save(context.Background(), client)
// 	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
// 	client = clients

// 	cases := map[string]struct {
// 		ID  string
// 		err error
// 	}{
// 		"retrieve existing client":     {client.ID, nil},
// 		"retrieve non-existing client": {wrongID, errors.ErrNotFound},
// 	}

// 	for desc, tc := range cases {
// 		cli, err := repo.RetrieveByID(context.Background(), tc.ID)
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
// 		if err == nil {
// 			assert.Equal(t, client.ID, cli.ID, fmt.Sprintf("retrieve client by ID : client ID : expected %s got %s\n", client.ID, cli.ID))
// 			assert.Equal(t, client.Name, cli.Name, fmt.Sprintf("retrieve client by ID : client Name : expected %s got %s\n", client.Name, cli.Name))
// 			assert.Equal(t, client.Credentials.Identity, cli.Credentials.Identity, fmt.Sprintf("retrieve client by ID : client Identity : expected %s got %s\n", client.Credentials.Identity, cli.Credentials.Identity))
// 			assert.Equal(t, client.Status, cli.Status, fmt.Sprintf("retrieve client by ID : client Status : expected %d got %d\n", client.Status, cli.Status))
// 		}
// 	}
// }

// func TestClientsRetrieveByIdentity(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)

// 	client := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: clientName,
// 		Credentials: mfclients.Credentials{
// 			Identity: clientIdentity,
// 			Secret:   password,
// 		},
// 		Status: mfclients.EnabledStatus,
// 	}

// 	_, err := repo.Save(context.Background(), client)
// 	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

// 	cases := map[string]struct {
// 		identity string
// 		err      error
// 	}{
// 		"retrieve existing client":     {clientIdentity, nil},
// 		"retrieve non-existing client": {wrongID, errors.ErrNotFound},
// 	}

// 	for desc, tc := range cases {
// 		_, err := repo.RetrieveByIdentity(context.Background(), tc.identity)
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
// 	}
// }

// func TestClientsRetrieveAll(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)
// 	grepo := gpostgres.New(database)
// 	prepo := ppostgres.NewRepository(database)

// 	nClients := uint64(200)
// 	ownerID := testsutil.GenerateUUID(t, idProvider)

// 	meta := mfclients.Metadata{
// 		"admin": "true",
// 	}
// 	wrongMeta := mfclients.Metadata{
// 		"admin": "false",
// 	}
// 	expectedClients := []mfclients.Client{}

// 	sharedGroup := mfgroups.Group{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "shared-group",
// 	}
// 	_, err := grepo.Save(context.Background(), sharedGroup)
// 	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

// 	for i := uint64(0); i < nClients; i++ {
// 		identity := fmt.Sprintf("TestRetrieveAll%d@example.com", i)
// 		client := mfclients.Client{
// 			ID:   testsutil.GenerateUUID(t, idProvider),
// 			Name: identity,
// 			Credentials: mfclients.Credentials{
// 				Identity: identity,
// 				Secret:   password,
// 			},
// 			Metadata: mfclients.Metadata{},
// 			Status:   mfclients.EnabledStatus,
// 		}
// 		if i%10 == 0 {
// 			client.Owner = ownerID
// 			client.Metadata = meta
// 			client.Tags = []string{"Test"}
// 		}
// 		if i%50 == 0 {
// 			client.Status = mfclients.DisabledStatus
// 		}
// 		client, err = repo.Save(context.Background(), client)
// 		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
// 		expectedClients = append(expectedClients, client)
// 		policy := policies.Policy{
// 			Subject: client.ID,
// 			Object:  sharedGroup.ID,
// 			Actions: []string{"c_list"},
// 		}
// 		err = prepo.Save(context.Background(), policy)
// 		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
// 	}

// 	cases := map[string]struct {
// 		size     uint64
// 		pm       mfclients.Page
// 		response []mfclients.Client
// 	}{
// 		"retrieve all clients empty page": {
// 			pm:       mfclients.Page{},
// 			response: []mfclients.Client{},
// 			size:     0,
// 		},
// 		"retrieve all clients": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: expectedClients,
// 			size:     200,
// 		},
// 		"retrieve all clients with limit": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  50,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: expectedClients[0:50],
// 			size:     50,
// 		},
// 		"retrieve all clients with offset": {
// 			pm: mfclients.Page{
// 				Offset: 50,
// 				Limit:  nClients,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: expectedClients[50:200],
// 			size:     150,
// 		},
// 		"retrieve all clients with limit and offset": {
// 			pm: mfclients.Page{
// 				Offset: 50,
// 				Limit:  50,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: expectedClients[50:100],
// 			size:     50,
// 		},
// 		"retrieve all clients with limit and offset not full": {
// 			pm: mfclients.Page{
// 				Offset: 170,
// 				Limit:  50,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: expectedClients[170:200],
// 			size:     30,
// 		},
// 		"retrieve all clients by metadata": {
// 			pm: mfclients.Page{
// 				Offset:   0,
// 				Limit:    nClients,
// 				Total:    nClients,
// 				Metadata: meta,
// 				Status:   mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{
// 				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
// 				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
// 				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
// 			},
// 			size: 20,
// 		},
// 		"retrieve clients by wrong metadata": {
// 			pm: mfclients.Page{
// 				Offset:   0,
// 				Limit:    nClients,
// 				Total:    nClients,
// 				Metadata: wrongMeta,
// 				Status:   mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{},
// 			size:     0,
// 		},
// 		"retrieve all clients by name": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Name:   "TestRetrieveAll3@example.com",
// 				Status: mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{expectedClients[3]},
// 			size:     1,
// 		},
// 		"retrieve clients by wrong name": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Name:   wrongName,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{},
// 			size:     0,
// 		},
// 		"retrieve all clients by owner": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Owner:  ownerID,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{
// 				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
// 				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
// 				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
// 			},
// 			size: 20,
// 		},
// 		"retrieve clients by wrong owner": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Owner:  wrongID,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{},
// 			size:     0,
// 		},
// 		"retrieve all clients shared by": {
// 			pm: mfclients.Page{
// 				Offset:   0,
// 				Limit:    nClients,
// 				Total:    nClients,
// 				SharedBy: expectedClients[0].ID,
// 				Action:   "c_list",
// 				Status:   mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{
// 				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
// 				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
// 				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
// 			},
// 			size: 20,
// 		},
// 		"retrieve all clients shared by and owned by": {
// 			pm: mfclients.Page{
// 				Offset:   0,
// 				Limit:    nClients,
// 				Total:    nClients,
// 				SharedBy: ownerID,
// 				Owner:    ownerID,
// 				Status:   mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{
// 				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
// 				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
// 				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
// 			},
// 			size: 20,
// 		},
// 		"retrieve all clients by disabled status": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Status: mfclients.DisabledStatus,
// 			},
// 			response: []mfclients.Client{expectedClients[0], expectedClients[50], expectedClients[100], expectedClients[150]},
// 			size:     4,
// 		},
// 		"retrieve all clients by combined status": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Status: mfclients.AllStatus,
// 			},
// 			response: expectedClients,
// 			size:     200,
// 		},
// 		"retrieve clients by the wrong status": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Status: 10,
// 			},
// 			response: []mfclients.Client{},
// 			size:     0,
// 		},
// 		"retrieve all clients by tags": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Tag:    "Test",
// 				Status: mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{
// 				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
// 				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
// 				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
// 			},
// 			size: 20,
// 		},
// 		"retrieve clients by wrong tags": {
// 			pm: mfclients.Page{
// 				Offset: 0,
// 				Limit:  nClients,
// 				Total:  nClients,
// 				Tag:    "wrongTags",
// 				Status: mfclients.AllStatus,
// 			},
// 			response: []mfclients.Client{},
// 			size:     0,
// 		},
// 		"retrieve all clients by sharedby": {
// 			pm: mfclients.Page{
// 				Offset:   0,
// 				Limit:    nClients,
// 				Total:    nClients,
// 				SharedBy: expectedClients[0].ID,
// 				Status:   mfclients.AllStatus,
// 				Action:   "c_list",
// 			},
// 			response: []mfclients.Client{
// 				expectedClients[0], expectedClients[10], expectedClients[20], expectedClients[30], expectedClients[40], expectedClients[50], expectedClients[60],
// 				expectedClients[70], expectedClients[80], expectedClients[90], expectedClients[100], expectedClients[110], expectedClients[120], expectedClients[130],
// 				expectedClients[140], expectedClients[150], expectedClients[160], expectedClients[170], expectedClients[180], expectedClients[190],
// 			},
// 			size: 20,
// 		},
// 	}
// 	for desc, tc := range cases {
// 		page, err := repo.RetrieveAll(context.Background(), tc.pm)
// 		size := uint64(len(page.Clients))
// 		assert.ElementsMatch(t, page.Clients, tc.response, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.response, page.Clients))
// 		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
// 		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
// 	}
// }

// func TestGroupsMembers(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	crepo := cpostgres.NewRepository(database)
// 	grepo := gpostgres.New(database)
// 	prepo := ppostgres.NewRepository(database)

// 	clientA := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "client-memberships",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client-memberships1@example.com",
// 			Secret:   password,
// 		},
// 		Metadata: mfclients.Metadata{},
// 		Status:   mfclients.EnabledStatus,
// 	}
// 	clientB := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "client-memberships",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client-memberships2@example.com",
// 			Secret:   password,
// 		},
// 		Metadata: mfclients.Metadata{},
// 		Status:   mfclients.EnabledStatus,
// 	}
// 	group := mfgroups.Group{
// 		ID:       testsutil.GenerateUUID(t, idProvider),
// 		Name:     "group-membership",
// 		Metadata: mfclients.Metadata{},
// 		Status:   mfclients.EnabledStatus,
// 	}

// 	policyA := policies.Policy{
// 		Subject: clientA.ID,
// 		Object:  group.ID,
// 		Actions: []string{"g_list"},
// 	}
// 	policyB := policies.Policy{
// 		Subject: clientB.ID,
// 		Object:  group.ID,
// 		Actions: []string{"g_list"},
// 	}

// 	_, err := crepo.Save(context.Background(), clientA)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("save client: expected %v got %s\n", nil, err))
// 	clientA.Credentials.Secret = ""
// 	_, err = crepo.Save(context.Background(), clientB)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("save client: expected %v got %s\n", nil, err))
// 	clientB.Credentials.Secret = ""
// 	_, err = grepo.Save(context.Background(), group)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("save group: expected %v got %s\n", nil, err))
// 	err = prepo.Save(context.Background(), policyA)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("save policy: expected %v got %s\n", nil, err))
// 	err = prepo.Save(context.Background(), policyB)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("save policy: expected %v got %s\n", nil, err))

// 	cases := map[string]struct {
// 		ID  string
// 		err error
// 	}{
// 		"retrieve members for existing group":     {group.ID, nil},
// 		"retrieve members for non-existing group": {wrongID, nil},
// 	}

// 	for desc, tc := range cases {
// 		mp, err := crepo.Members(context.Background(), tc.ID, mfclients.Page{Total: 10, Offset: 0, Limit: 10, Status: mfclients.AllStatus, Subject: clientB.ID, Action: "g_list"})
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
// 		if tc.ID == group.ID {
// 			assert.ElementsMatch(t, mp.Members, []mfclients.Client{clientA}, fmt.Sprintf("%s: expected %v got %v\n", desc, []mfclients.Client{clientA, clientB}, mp.Members))
// 		}
// 	}
// }

// func TestClientsUpdateMetadata(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)

// 	client1 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "enabled-client",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client1-update@example.com",
// 			Secret:   password,
// 		},
// 		Metadata: mfclients.Metadata{
// 			"name": "enabled-client",
// 		},
// 		Tags:   []string{"enabled", "tag1"},
// 		Status: mfclients.EnabledStatus,
// 	}

// 	client2 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "disabled-client",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client2-update@example.com",
// 			Secret:   password,
// 		},
// 		Metadata: mfclients.Metadata{
// 			"name": "disabled-client",
// 		},
// 		Tags:   []string{"disabled", "tag1"},
// 		Status: mfclients.DisabledStatus,
// 	}

// 	clients1, err := repo.Save(context.Background(), client1)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client with metadata: expected %v got %s\n", nil, err))
// 	clients2, err := repo.Save(context.Background(), client2)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
// 	client1 = clients1
// 	client2 = clients2

// 	ucases := []struct {
// 		desc   string
// 		update string
// 		client mfclients.Client
// 		err    error
// 	}{
// 		{
// 			desc:   "update metadata for enabled client",
// 			update: "metadata",
// 			client: mfclients.Client{
// 				ID: client1.ID,
// 				Metadata: mfclients.Metadata{
// 					"update": "metadata",
// 				},
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc:   "update metadata for disabled client",
// 			update: "metadata",
// 			client: mfclients.Client{
// 				ID: client2.ID,
// 				Metadata: mfclients.Metadata{
// 					"update": "metadata",
// 				},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc:   "update name for enabled client",
// 			update: "name",
// 			client: mfclients.Client{
// 				ID:   client1.ID,
// 				Name: "updated name",
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc:   "update name for disabled client",
// 			update: "name",
// 			client: mfclients.Client{
// 				ID:   client2.ID,
// 				Name: "updated name",
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc:   "update name and metadata for enabled client",
// 			update: "both",
// 			client: mfclients.Client{
// 				ID:   client1.ID,
// 				Name: "updated name and metadata",
// 				Metadata: mfclients.Metadata{
// 					"update": "name and metadata",
// 				},
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc:   "update name and metadata for a disabled client",
// 			update: "both",
// 			client: mfclients.Client{
// 				ID:   client2.ID,
// 				Name: "updated name and metadata",
// 				Metadata: mfclients.Metadata{
// 					"update": "name and metadata",
// 				},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc:   "update metadata for invalid client",
// 			update: "metadata",
// 			client: mfclients.Client{
// 				ID: wrongID,
// 				Metadata: mfclients.Metadata{
// 					"update": "metadata",
// 				},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc:   "update name for invalid client",
// 			update: "name",
// 			client: mfclients.Client{
// 				ID:   wrongID,
// 				Name: "updated name",
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc:   "update name and metadata for invalid client",
// 			update: "both",
// 			client: mfclients.Client{
// 				ID:   client2.ID,
// 				Name: "updated name and metadata",
// 				Metadata: mfclients.Metadata{
// 					"update": "name and metadata",
// 				},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 	}
// 	for _, tc := range ucases {
// 		expected, err := repo.Update(context.Background(), tc.client)
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
// 		if err == nil {
// 			if tc.client.Name != "" {
// 				assert.Equal(t, expected.Name, tc.client.Name, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected.Name, tc.client.Name))
// 			}
// 			if tc.client.Metadata != nil {
// 				assert.Equal(t, expected.Metadata, tc.client.Metadata, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected.Metadata, tc.client.Metadata))
// 			}

// 		}
// 	}
// }

// func TestClientsUpdateTags(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)

// 	client1 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "enabled-client-with-tags",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client1-update-tags@example.com",
// 			Secret:   password,
// 		},
// 		Tags:   []string{"test", "enabled"},
// 		Status: mfclients.EnabledStatus,
// 	}
// 	client2 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "disabled-client-with-tags",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client2-update-tags@example.com",
// 			Secret:   password,
// 		},
// 		Tags:   []string{"test", "disabled"},
// 		Status: mfclients.DisabledStatus,
// 	}

// 	clients1, err := repo.Save(context.Background(), client1)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client with tags: expected %v got %s\n", nil, err))
// 	if err == nil {
// 		assert.Equal(t, client1.ID, client1.ID, fmt.Sprintf("add new client with tags: expected %v got %s\n", nil, err))
// 	}
// 	client1 = clients1
// 	clients2, err := repo.Save(context.Background(), client2)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client with tags: expected %v got %s\n", nil, err))
// 	if err == nil {
// 		assert.Equal(t, client2.ID, client2.ID, fmt.Sprintf("add new disabled client with tags: expected %v got %s\n", nil, err))
// 	}
// 	client2 = clients2
// 	ucases := []struct {
// 		desc   string
// 		client mfclients.Client
// 		err    error
// 	}{
// 		{
// 			desc: "update tags for enabled client",
// 			client: mfclients.Client{
// 				ID:   client1.ID,
// 				Tags: []string{"updated"},
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc: "update tags for disabled client",
// 			client: mfclients.Client{
// 				ID:   client2.ID,
// 				Tags: []string{"updated"},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc: "update tags for invalid client",
// 			client: mfclients.Client{
// 				ID:   wrongID,
// 				Tags: []string{"updated"},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 	}
// 	for _, tc := range ucases {
// 		expected, err := repo.UpdateTags(context.Background(), tc.client)
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
// 		if err == nil {
// 			assert.Equal(t, tc.client.Tags, expected.Tags, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Tags, expected.Tags))
// 		}
// 	}
// }

// func TestClientsUpdateSecret(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)

// 	client1 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "enabled-client",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client1-update@example.com",
// 			Secret:   password,
// 		},
// 		Status: mfclients.EnabledStatus,
// 	}
// 	client2 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "disabled-client",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client2-update@example.com",
// 			Secret:   password,
// 		},
// 		Status: mfclients.DisabledStatus,
// 	}

// 	rClients1, err := repo.Save(context.Background(), client1)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
// 	if err == nil {
// 		assert.Equal(t, client1.ID, rClients1.ID, fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
// 	}
// 	rClients2, err := repo.Save(context.Background(), client2)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
// 	if err == nil {
// 		assert.Equal(t, client2.ID, rClients2.ID, fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
// 	}

// 	ucases := []struct {
// 		desc   string
// 		client mfclients.Client
// 		err    error
// 	}{
// 		{
// 			desc: "update secret for enabled client",
// 			client: mfclients.Client{
// 				ID: client1.ID,
// 				Credentials: mfclients.Credentials{
// 					Identity: "client1-update@example.com",
// 					Secret:   "newpassword",
// 				},
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc: "update secret for disabled client",
// 			client: mfclients.Client{
// 				ID: client2.ID,
// 				Credentials: mfclients.Credentials{
// 					Identity: "client2-update@example.com",
// 					Secret:   "newpassword",
// 				},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc: "update secret for invalid client",
// 			client: mfclients.Client{
// 				ID: wrongID,
// 				Credentials: mfclients.Credentials{
// 					Identity: "client3-update@example.com",
// 					Secret:   "newpassword",
// 				},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 	}
// 	for _, tc := range ucases {
// 		_, err := repo.UpdateSecret(context.Background(), tc.client)
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
// 		if err == nil {
// 			c, err := repo.RetrieveByIdentity(context.Background(), tc.client.Credentials.Identity)
// 			require.Nil(t, err, fmt.Sprintf("retrieve client by id during update of secret unexpected error: %s", err))
// 			assert.Equal(t, tc.client.Credentials.Secret, c.Credentials.Secret, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Credentials.Secret, c.Credentials.Secret))
// 		}
// 	}
// }

// func TestClientsUpdateIdentity(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)

// 	client1 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "enabled-client",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client1-update@example.com",
// 			Secret:   password,
// 		},
// 		Status: mfclients.EnabledStatus,
// 	}
// 	client2 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "disabled-client",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client2-update@example.com",
// 			Secret:   password,
// 		},
// 		Status: mfclients.DisabledStatus,
// 	}

// 	rClients1, err := repo.Save(context.Background(), client1)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
// 	if err == nil {
// 		assert.Equal(t, client1.ID, rClients1.ID, fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
// 	}
// 	rClients2, err := repo.Save(context.Background(), client2)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
// 	if err == nil {
// 		assert.Equal(t, client2.ID, rClients2.ID, fmt.Sprintf("add new disabled client: expected %v got %s\n", nil, err))
// 	}

// 	ucases := []struct {
// 		desc   string
// 		client mfclients.Client
// 		err    error
// 	}{
// 		{
// 			desc: "update identity for enabled client",
// 			client: mfclients.Client{
// 				ID: client1.ID,
// 				Credentials: mfclients.Credentials{
// 					Identity: "client1-updated@example.com",
// 				},
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc: "update identity for disabled client",
// 			client: mfclients.Client{
// 				ID: client2.ID,
// 				Credentials: mfclients.Credentials{
// 					Identity: "client2-updated@example.com",
// 				},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc: "update identity for invalid client",
// 			client: mfclients.Client{
// 				ID: wrongID,
// 				Credentials: mfclients.Credentials{
// 					Identity: "client3-updated@example.com",
// 				},
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 	}
// 	for _, tc := range ucases {
// 		expected, err := repo.UpdateIdentity(context.Background(), tc.client)
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
// 		if err == nil {
// 			assert.Equal(t, tc.client.Credentials.Identity, expected.Credentials.Identity, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Credentials.Identity, expected.Credentials.Identity))
// 		}
// 	}
// }

// func TestClientsUpdateOwner(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)

// 	client1 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "enabled-client-with-owner",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client1-update-owner@example.com",
// 			Secret:   password,
// 		},
// 		Owner:  testsutil.GenerateUUID(t, idProvider),
// 		Status: mfclients.EnabledStatus,
// 	}
// 	client2 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "disabled-client-with-owner",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client2-update-owner@example.com",
// 			Secret:   password,
// 		},
// 		Owner:  testsutil.GenerateUUID(t, idProvider),
// 		Status: mfclients.DisabledStatus,
// 	}

// 	clients1, err := repo.Save(context.Background(), client1)
// 	client1 = clients1
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client with owner: expected %v got %s\n", nil, err))
// 	if err == nil {
// 		assert.Equal(t, client1.ID, client1.ID, fmt.Sprintf("add new client with owner: expected %v got %s\n", nil, err))
// 	}
// 	clients2, err := repo.Save(context.Background(), client2)
// 	client2 = clients2
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new disabled client with owner: expected %v got %s\n", nil, err))
// 	if err == nil {
// 		assert.Equal(t, client2.ID, client2.ID, fmt.Sprintf("add new disabled client with owner: expected %v got %s\n", nil, err))
// 	}
// 	ucases := []struct {
// 		desc   string
// 		client mfclients.Client
// 		err    error
// 	}{
// 		{
// 			desc: "update owner for enabled client",
// 			client: mfclients.Client{
// 				ID:    client1.ID,
// 				Owner: testsutil.GenerateUUID(t, idProvider),
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc: "update owner for disabled client",
// 			client: mfclients.Client{
// 				ID:    client2.ID,
// 				Owner: testsutil.GenerateUUID(t, idProvider),
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 		{
// 			desc: "update owner for invalid client",
// 			client: mfclients.Client{
// 				ID:    wrongID,
// 				Owner: testsutil.GenerateUUID(t, idProvider),
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 	}
// 	for _, tc := range ucases {
// 		expected, err := repo.UpdateOwner(context.Background(), tc.client)
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
// 		if err == nil {
// 			assert.Equal(t, tc.client.Owner, expected.Owner, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Owner, expected.Owner))
// 		}
// 	}
// }

// func TestClientsChangeStatus(t *testing.T) {
// 	t.Cleanup(func() { testsutil.CleanUpDB(t, db) })
// 	repo := cpostgres.NewRepository(database)

// 	client1 := mfclients.Client{
// 		ID:   testsutil.GenerateUUID(t, idProvider),
// 		Name: "enabled-client",
// 		Credentials: mfclients.Credentials{
// 			Identity: "client1-update@example.com",
// 			Secret:   password,
// 		},
// 		Status: mfclients.EnabledStatus,
// 	}

// 	clients1, err := repo.Save(context.Background(), client1)
// 	assert.True(t, errors.Contains(err, nil), fmt.Sprintf("add new client: expected %v got %s\n", nil, err))
// 	client1 = clients1

// 	ucases := []struct {
// 		desc   string
// 		client mfclients.Client
// 		err    error
// 	}{
// 		{
// 			desc: "change client status for an enabled client",
// 			client: mfclients.Client{
// 				ID:     client1.ID,
// 				Status: 0,
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc: "change client status for a disabled client",
// 			client: mfclients.Client{
// 				ID:     client1.ID,
// 				Status: 1,
// 			},
// 			err: nil,
// 		},
// 		{
// 			desc: "change client status for non-existing client",
// 			client: mfclients.Client{
// 				ID:     "invalid",
// 				Status: 2,
// 			},
// 			err: errors.ErrNotFound,
// 		},
// 	}

// 	for _, tc := range ucases {
// 		expected, err := repo.ChangeStatus(context.Background(), tc.client)
// 		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
// 		if err == nil {
// 			assert.Equal(t, tc.client.Status, expected.Status, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.client.Status, expected.Status))
// 		}
// 	}
// }
