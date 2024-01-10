// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/things"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	secret         = "strongsecret"
	validCMetadata = mgclients.Metadata{"role": "client"}
	ID             = testsutil.GenerateUUID(&testing.T{})
	client         = mgclients.Client{
		ID:          ID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mgclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	adminEmail = "admin@example.com"
	myKey      = "mine"
	validToken = "token"
	validID    = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID    = testsutil.GenerateUUID(&testing.T{})
)

func newService() (things.Service, *mocks.Repository, *authmocks.Service, *mocks.Cache) {
	auth := new(authmocks.Service)
	thingCache := new(mocks.Cache)
	idProvider := uuid.NewMock()
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)

	return things.NewService(auth, cRepo, gRepo, thingCache, idProvider), cRepo, auth, thingCache
}

func TestRegisterClient(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		err    error
	}{
		{
			desc:   "register new client",
			client: client,
			token:  validToken,
			err:    nil,
		},
		{
			desc:   "register existing client",
			client: client,
			token:  validToken,
			err:    svcerr.ErrConflict,
		},
		{
			desc: "register a new enabled client with name",
			client: mgclients.Client{
				Name: "clientWithName",
				Credentials: mgclients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new disabled client with name",
			client: mgclients.Client{
				Name: "clientWithName",
				Credentials: mgclients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new enabled client with tags",
			client: mgclients.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: mgclients.Credentials{
					Identity: "newclientwithtags@example.com",
					Secret:   secret,
				},
				Status: mgclients.EnabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new disabled client with tags",
			client: mgclients.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: mgclients.Credentials{
					Identity: "newclientwithtags@example.com",
					Secret:   secret,
				},
				Status: mgclients.DisabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new enabled client with metadata",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validCMetadata,
				Status:   mgclients.EnabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new disabled client with metadata",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validCMetadata,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new disabled client",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new client with valid disabled status",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
				Status: mgclients.DisabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new client with all fields",
			client: mgclients.Client{
				Name: "newclientwithallfields",
				Tags: []string{"tag1", "tag2"},
				Credentials: mgclients.Credentials{
					Identity: "newclientwithallfields@example.com",
					Secret:   secret,
				},
				Metadata: mgclients.Metadata{
					"name": "newclientwithallfields",
				},
				Status: mgclients.EnabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new client with missing identity",
			client: mgclients.Client{
				Name: "clientWithMissingIdentity",
				Credentials: mgclients.Credentials{
					Secret: secret,
				},
			},
			err:   repoerr.ErrMalformedEntity,
			token: validToken,
		},
		{
			desc: "register a new client with invalid owner",
			client: mgclients.Client{
				Owner: wrongID,
				Credentials: mgclients.Credentials{
					Identity: "newclientwithinvalidowner@example.com",
					Secret:   secret,
				},
			},
			err:   repoerr.ErrMalformedEntity,
			token: validToken,
		},
		{
			desc: "register a new client with empty secret",
			client: mgclients.Client{
				Owner: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: "newclientwithemptysecret@example.com",
				},
			},
			err:   repoerr.ErrMissingSecret,
			token: validToken,
		},
		{
			desc: "register a new client with invalid status",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: mgclients.AllStatus,
			},
			err:   svcerr.ErrInvalidStatus,
			token: validToken,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(&magistrala.AddPoliciesRes{Authorized: true}, nil)
		repoCall2 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall3 := cRepo.On("Save", context.Background(), mock.Anything).Return([]mgclients.Client{tc.client}, tc.err)
		expected, err := svc.CreateThings(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.client.ID = expected[0].ID
			tc.client.CreatedAt = expected[0].CreatedAt
			tc.client.UpdatedAt = expected[0].UpdatedAt
			tc.client.Credentials.Secret = expected[0].Credentials.Secret
			tc.client.Owner = expected[0].Owner
			tc.client.UpdatedBy = expected[0].UpdatedBy
			assert.Equal(t, tc.client, expected[0], fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, expected[0]))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestViewClient(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	cases := []struct {
		desc     string
		token    string
		clientID string
		response mgclients.Client
		err      error
	}{
		{
			desc:     "view client successfully",
			response: client,
			token:    validToken,
			clientID: client.ID,
			err:      nil,
		},
		{
			desc:     "view client with an invalid token",
			response: mgclients.Client{},
			token:    authmocks.InvalidValue,
			clientID: "",
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "view client with valid token and invalid client id",
			response: mgclients.Client{},
			token:    validToken,
			clientID: wrongID,
			err:      svcerr.ErrNotFound,
		},
		{
			desc:     "view client with an invalid token and invalid client id",
			response: mgclients.Client{},
			token:    authmocks.InvalidValue,
			clientID: wrongID,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, tc.err)
		if tc.token == authmocks.InvalidValue {
			repoCall = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.response, tc.err)
		rClient, err := svc.ViewClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListClients(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	nClients := uint64(200)
	aClients := []mgclients.Client{}
	OwnerID := testsutil.GenerateUUID(t)
	for i := uint64(1); i < nClients; i++ {
		identity := fmt.Sprintf("TestListClients_%d@example.com", i)
		client := mgclients.Client{
			Name: identity,
			Credentials: mgclients.Credentials{
				Identity: identity,
				Secret:   "password",
			},
			Tags:     []string{"tag1", "tag2"},
			Metadata: mgclients.Metadata{"role": "client"},
		}
		if i%50 == 0 {
			client.Owner = OwnerID
			client.Owner = testsutil.GenerateUUID(t)
		}
		aClients = append(aClients, client)
	}

	cases := []struct {
		desc     string
		token    string
		page     mgclients.Page
		response mgclients.ClientsPage
		size     uint64
		err      error
	}{
		{
			desc:  "list clients with authorized token",
			token: validToken,

			page: mgclients.Page{
				Status: mgclients.AllStatus,
			},
			size: 0,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client(nil),
			},
			err: nil,
		},
		{
			desc:  "list clients with an invalid token",
			token: authmocks.InvalidValue,
			page: mgclients.Page{
				Status: mgclients.AllStatus,
			},
			size: 0,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:  "list clients that are shared with me",
			token: validToken,
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Status: mgclients.EnabledStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that are shared with me with a specific name",
			token: validToken,
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Name:   "TestListClients3",
				Status: mgclients.EnabledStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that are shared with me with an invalid name",
			token: validToken,
			page: mgclients.Page{
				Limit:  nClients,
				Name:   "notpresentclient",
				Status: mgclients.EnabledStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Limit: nClients,
				},
				Clients: []mgclients.Client(nil),
			},
			size: 0,
		},
		{
			desc:  "list clients that I own",
			token: validToken,
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Status: mgclients.EnabledStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that I own with a specific name",
			token: validToken,
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Name:   "TestListClients3",
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that I own with an invalid name",
			token: validToken,
			page: mgclients.Page{
				Limit:  nClients,
				Owner:  myKey,
				Name:   "notpresentclient",
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
			size: 0,
		},
		{
			desc:  "list clients that I own and are shared with me",
			token: validToken,
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that I own and are shared with me with a specific name",
			token: validToken,
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Name:   "TestListClients3",
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that I own and are shared with me with an invalid name",
			token: validToken,
			page: mgclients.Page{
				Limit:  nClients,
				Owner:  myKey,
				Name:   "notpresentclient",
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  nClients,
				},
				Clients: []mgclients.Client(nil),
			},
			size: 0,
		},
		{
			desc:  "list clients with offset and limit",
			token: validToken,

			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  nClients - 6,
					Offset: 0,
					Limit:  0,
				},
				Clients: aClients[6:nClients],
			},
			size: nClients - 6,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: getIDs(tc.response.Clients)}, nil)
		if tc.token == authmocks.InvalidValue {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: authmocks.InvalidValue}).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
			repoCall2 = auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{}, svcerr.ErrAuthorization)
		}
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.response, tc.err)
		page, err := svc.ListClients(context.Background(), tc.token, "", tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	client1 := client
	client2 := client
	client1.Name = "Updated client"
	client2.Metadata = mgclients.Metadata{"role": "test"}

	cases := []struct {
		desc     string
		client   mgclients.Client
		response mgclients.Client
		token    string
		err      error
	}{
		{
			desc:     "update client name with valid token",
			client:   client1,
			response: client1,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client name with invalid token",
			client:   client1,
			response: mgclients.Client{},
			token:    "non-existent",
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "update client name with invalid ID",
			client: mgclients.Client{
				ID:   wrongID,
				Name: "Updated Client",
			},
			response: mgclients.Client{},
			token:    validToken,
			err:      svcerr.ErrNotFound,
		},
		{
			desc:     "update client metadata with valid token",
			client:   client2,
			response: client2,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client metadata with invalid token",
			client:   client2,
			response: mgclients.Client{},
			token:    "non-existent",
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(mgclients.Client{}, tc.err)
		repoCall3 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	client.Tags = []string{"updated"}

	cases := []struct {
		desc     string
		client   mgclients.Client
		response mgclients.Client
		token    string
		err      error
	}{
		{
			desc:     "update client tags with valid token",
			client:   client,
			token:    validToken,
			response: client,
			err:      nil,
		},
		{
			desc:     "update client tags with invalid token",
			client:   client,
			token:    "non-existent",
			response: mgclients.Client{},
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "update client name with invalid ID",
			client: mgclients.Client{
				ID:   wrongID,
				Name: "Updated name",
			},
			response: mgclients.Client{},
			token:    validToken,
			err:      svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(mgclients.Client{}, tc.err)
		repoCall3 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientTags(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	cases := []struct {
		desc      string
		id        string
		newSecret string
		token     string
		response  mgclients.Client
		err       error
	}{
		{
			desc:      "update client secret with valid token",
			id:        client.ID,
			newSecret: "newSecret",
			token:     validToken,
			response:  client,
			err:       nil,
		},
		{
			desc:      "update client secret with invalid token",
			id:        client.ID,
			newSecret: "newPassword",
			token:     "non-existent",
			response:  mgclients.Client{},
			err:       svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientSecret(context.Background(), tc.token, tc.id, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	enabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	endisabledClient1 := disabledClient1
	endisabledClient1.Status = mgclients.EnabledStatus

	cases := []struct {
		desc     string
		id       string
		token    string
		client   mgclients.Client
		response mgclients.Client
		err      error
	}{
		{
			desc:     "enable disabled client",
			id:       disabledClient1.ID,
			token:    validToken,
			client:   disabledClient1,
			response: endisabledClient1,
			err:      nil,
		},
		{
			desc:     "enable enabled client",
			id:       enabledClient1.ID,
			token:    validToken,
			client:   enabledClient1,
			response: enabledClient1,
			err:      mgclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "enable non-existing client",
			id:       wrongID,
			token:    validToken,
			client:   mgclients.Client{},
			response: mgclients.Client{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.client, tc.err)
		repoCall3 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		_, err := svc.EnableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}

	cases2 := []struct {
		desc     string
		status   mgclients.Status
		size     uint64
		response mgclients.ClientsPage
	}{
		{
			desc:   "list enabled clients",
			status: mgclients.EnabledStatus,
			size:   2,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{enabledClient1, endisabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mgclients.DisabledStatus,
			size:   1,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mgclients.AllStatus,
			size:   3,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  3,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{enabledClient1, disabledClient1, endisabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := mgclients.Page{
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: getIDs(tc.response.Clients)}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, "", pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	svc, cRepo, auth, cache := newService()

	enabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: ID, Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = mgclients.DisabledStatus

	cases := []struct {
		desc     string
		id       string
		token    string
		client   mgclients.Client
		response mgclients.Client
		err      error
	}{
		{
			desc:     "disable enabled client",
			id:       enabledClient1.ID,
			token:    validToken,
			client:   enabledClient1,
			response: disenabledClient1,
			err:      nil,
		},
		{
			desc:     "disable disabled client",
			id:       disabledClient1.ID,
			token:    validToken,
			client:   disabledClient1,
			response: mgclients.Client{},
			err:      mgclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "disable non-existing client",
			id:       wrongID,
			client:   mgclients.Client{},
			token:    validToken,
			response: mgclients.Client{},
			err:      repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.client, tc.err)
		repoCall3 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		repoCall4 := cache.On("Remove", mock.Anything, mock.Anything).Return(nil)
		_, err := svc.DisableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}

	cases2 := []struct {
		desc     string
		status   mgclients.Status
		size     uint64
		response mgclients.ClientsPage
	}{
		{
			desc:   "list enabled clients",
			status: mgclients.EnabledStatus,
			size:   1,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{enabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mgclients.DisabledStatus,
			size:   2,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{disenabledClient1, disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mgclients.AllStatus,
			size:   3,
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  3,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mgclients.Client{enabledClient1, disabledClient1, disenabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := mgclients.Page{
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: getIDs(tc.response.Clients)}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, "", pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestListMembers(t *testing.T) {
	svc, cRepo, auth, _ := newService()

	nClients := uint64(10)
	aClients := []mgclients.Client{}
	owner := testsutil.GenerateUUID(t)
	for i := uint64(0); i < nClients; i++ {
		identity := fmt.Sprintf("member_%d@example.com", i)
		client := mgclients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: identity,
			Credentials: mgclients.Credentials{
				Identity: identity,
				Secret:   "password",
			},
			Tags:     []string{"tag1", "tag2"},
			Metadata: mgclients.Metadata{"role": "client"},
		}
		if i%3 == 0 {
			client.Owner = owner
		}
		aClients = append(aClients, client)
	}

	cases := []struct {
		desc     string
		token    string
		groupID  string
		page     mgclients.Page
		response mgclients.MembersPage
		err      error
	}{
		{
			desc:    "list clients with authorized token",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Owner: adminEmail,
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Members: []mgclients.Client{},
			},
			err: nil,
		},
		{
			desc:    "list clients with offset and limit",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Offset: 6,
				Limit:  nClients,
				Status: mgclients.AllStatus,
				Owner:  adminEmail,
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: nClients - 6 - 1,
				},
				Members: aClients[6 : nClients-1],
			},
		},
		{
			desc:    "list clients with an invalid token",
			token:   authmocks.InvalidValue,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Owner: adminEmail,
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:    "list clients with an invalid id",
			token:   validToken,
			groupID: wrongID,
			page: mgclients.Page{
				Owner: adminEmail,
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			err: svcerr.ErrNotFound,
		},
		{
			desc:    "list clients for an owner",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				Owner: adminEmail,
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 4,
				},
				Members: []mgclients.Client{aClients[0], aClients[3], aClients[6], aClients[9]},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), tc.page).Return(mgclients.ClientsPage{Page: tc.response.Page, Clients: tc.response.Members}, tc.err)
		page, err := svc.ListClientsByGroup(context.Background(), tc.token, tc.groupID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestDeleteClient(t *testing.T) {
	svc, cRepo, auth, cache := newService()

	client := mgclients.Client{
		ID:   testsutil.GenerateUUID(t),
		Name: "TestClient",
		Credentials: mgclients.Credentials{
			Identity: "TestClient@example.com",
			Secret:   "password",
		},
		Tags:     []string{"tag1", "tag2"},
		Metadata: mgclients.Metadata{"role": "client"},
	}
	invalidClientID := "invalidClientID"
	_ = invalidClientID
	cases := []struct {
		desc     string
		token    string
		clientID string
		err      error
	}{
		{
			desc:     "Delete client with authorized token",
			token:    validToken,
			clientID: client.ID,
			err:      nil,
		},
		{
			desc:     "Delete client with unauthorized token",
			token:    authmocks.InvalidValue,
			clientID: client.ID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "Delete invalid client",
			token:    validToken,
			clientID: authmocks.InvalidValue,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "Delete repo error ",
			token:    validToken,
			clientID: client.ID,
			err:      errors.ErrRemoveEntity,
		},
		{
			desc:     "Delete policy error ",
			token:    validToken,
			clientID: client.ID,
			err:      errors.ErrUnidentified,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("DeletePolicy", mock.Anything, mock.Anything).Return(&magistrala.DeletePolicyRes{Deleted: true}, nil)
		repoCall3 := cache.On("Remove", mock.Anything, tc.clientID).Return(nil)
		repoCall4 := cRepo.On("Delete", context.Background(), tc.clientID).Return(nil)
		if tc.err == errors.ErrRemoveEntity {
			repoCall4.Unset()
			repoCall4 = cRepo.On("Delete", context.Background(), tc.clientID).Return(errors.ErrRemoveEntity)
		}
		if tc.err == errors.ErrUnidentified {
			repoCall2.Unset()
			repoCall2 = auth.On("DeletePolicy", mock.Anything, mock.Anything).Return(&magistrala.DeletePolicyRes{Deleted: false}, errors.ErrUnidentified)
		}
		err := svc.DeleteClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func getIDs(clients []mgclients.Client) []string {
	ids := []string{}
	for _, client := range clients {
		ids = append(ids, client.ID)
	}
	return ids
}
