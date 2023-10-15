// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	authmocks "github.com/mainflux/mainflux/auth/mocks"
	"github.com/mainflux/mainflux/internal/apiutil"
	gmocks "github.com/mainflux/mainflux/internal/groups/mocks"
	"github.com/mainflux/mainflux/internal/testsutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	idProvider     = uuid.New()
	secret         = "strongsecret"
	validCMetadata = mfclients.Metadata{"role": "client"}
	ID             = testsutil.GenerateUUID(&testing.T{}, idProvider)
	client         = mfclients.Client{
		ID:          ID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mfclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mfclients.EnabledStatus,
	}
	withinDuration = 5 * time.Second
	adminEmail     = "admin@example.com"
	myKey          = "mine"
	validToken     = "token"
	inValidToken   = "invalidToken"
	validID        = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

func newService() (things.Service, *mocks.Repository, *authmocks.Service) {
	auth := new(authmocks.Service)
	thingCache := mocks.NewCache()
	idProvider := uuid.NewMock()
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)

	return things.NewService(auth, cRepo, gRepo, thingCache, idProvider), cRepo, auth
}

func TestRegisterClient(t *testing.T) {
	svc, cRepo, auth := newService()

	cases := []struct {
		desc   string
		client mfclients.Client
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
			err:    errors.ErrConflict,
		},
		{
			desc: "register a new enabled client with name",
			client: mfclients.Client{
				Name: "clientWithName",
				Credentials: mfclients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
				Status: mfclients.EnabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new disabled client with name",
			client: mfclients.Client{
				Name: "clientWithName",
				Credentials: mfclients.Credentials{
					Identity: "newclientwithname@example.com",
					Secret:   secret,
				},
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new enabled client with tags",
			client: mfclients.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: mfclients.Credentials{
					Identity: "newclientwithtags@example.com",
					Secret:   secret,
				},
				Status: mfclients.EnabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new disabled client with tags",
			client: mfclients.Client{
				Tags: []string{"tag1", "tag2"},
				Credentials: mfclients.Credentials{
					Identity: "newclientwithtags@example.com",
					Secret:   secret,
				},
				Status: mfclients.DisabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new enabled client with metadata",
			client: mfclients.Client{
				Credentials: mfclients.Credentials{
					Identity: "newclientwithmetadata@example.com",
					Secret:   secret,
				},
				Metadata: validCMetadata,
				Status:   mfclients.EnabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new disabled client with metadata",
			client: mfclients.Client{
				Credentials: mfclients.Credentials{
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
			client: mfclients.Client{
				Credentials: mfclients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new client with valid disabled status",
			client: mfclients.Client{
				Credentials: mfclients.Credentials{
					Identity: "newclientwithvalidstatus@example.com",
					Secret:   secret,
				},
				Status: mfclients.DisabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new client with all fields",
			client: mfclients.Client{
				Name: "newclientwithallfields",
				Tags: []string{"tag1", "tag2"},
				Credentials: mfclients.Credentials{
					Identity: "newclientwithallfields@example.com",
					Secret:   secret,
				},
				Metadata: mfclients.Metadata{
					"name": "newclientwithallfields",
				},
				Status: mfclients.EnabledStatus,
			},
			err:   nil,
			token: validToken,
		},
		{
			desc: "register a new client with missing identity",
			client: mfclients.Client{
				Name: "clientWithMissingIdentity",
				Credentials: mfclients.Credentials{
					Secret: secret,
				},
			},
			err:   errors.ErrMalformedEntity,
			token: validToken,
		},
		{
			desc: "register a new client with invalid owner",
			client: mfclients.Client{
				Owner: mocks.WrongID,
				Credentials: mfclients.Credentials{
					Identity: "newclientwithinvalidowner@example.com",
					Secret:   secret,
				},
			},
			err:   errors.ErrMalformedEntity,
			token: validToken,
		},
		{
			desc: "register a new client with empty secret",
			client: mfclients.Client{
				Owner: testsutil.GenerateUUID(t, idProvider),
				Credentials: mfclients.Credentials{
					Identity: "newclientwithemptysecret@example.com",
				},
			},
			err:   apiutil.ErrMissingSecret,
			token: validToken,
		},
		{
			desc: "register a new client with invalid status",
			client: mfclients.Client{
				Credentials: mfclients.Credentials{
					Identity: "newclientwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: mfclients.AllStatus,
			},
			err:   apiutil.ErrInvalidStatus,
			token: validToken,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("AddPolicy", mock.Anything, mock.Anything).Return(&mainflux.AddPolicyRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("Save", context.Background(), mock.Anything).Return(&mfclients.Client{}, tc.err)
		registerTime := time.Now()
		expected, err := svc.CreateThings(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, expected[0].ID, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, expected[0].ID))
			assert.WithinDuration(t, expected[0].CreatedAt, registerTime, withinDuration, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected[0].CreatedAt, registerTime))
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
	}
}

func TestViewClient(t *testing.T) {
	svc, cRepo, auth := newService()

	cases := []struct {
		desc     string
		token    string
		clientID string
		response mfclients.Client
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
			response: mfclients.Client{},
			token:    inValidToken,
			clientID: "",
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "view client with valid token and invalid client id",
			response: mfclients.Client{},
			token:    validToken,
			clientID: mocks.WrongID,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "view client with an invalid token and invalid client id",
			response: mfclients.Client{},
			token:    inValidToken,
			clientID: mocks.WrongID,
			err:      errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true}, tc.err)
		if tc.token == inValidToken {
			repoCall = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
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
	svc, cRepo, auth := newService()

	nClients := uint64(200)
	aClients := []mfclients.Client{}
	OwnerID := testsutil.GenerateUUID(t, idProvider)
	for i := uint64(1); i < nClients; i++ {
		identity := fmt.Sprintf("TestListClients_%d@example.com", i)
		client := mfclients.Client{
			Name: identity,
			Credentials: mfclients.Credentials{
				Identity: identity,
				Secret:   "password",
			},
			Tags:     []string{"tag1", "tag2"},
			Metadata: mfclients.Metadata{"role": "client"},
		}
		if i%50 == 0 {
			client.Owner = OwnerID
			client.Owner = testsutil.GenerateUUID(t, idProvider)
		}
		aClients = append(aClients, client)
	}

	cases := []struct {
		desc     string
		token    string
		page     mfclients.Page
		response mfclients.ClientsPage
		size     uint64
		err      error
	}{
		{
			desc:  "list clients with authorized token",
			token: validToken,

			page: mfclients.Page{
				Status: mfclients.AllStatus,
			},
			size: 0,
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{},
			},
			err: nil,
		},
		{
			desc:  "list clients with an invalid token",
			token: inValidToken,
			page: mfclients.Page{
				Status: mfclients.AllStatus,
			},
			size: 0,
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			err: errors.ErrAuthentication,
		},
		{
			desc:  "list clients that are shared with me",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Status: mfclients.EnabledStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that are shared with me with a specific name",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Name:   "TestListClients3",
				Status: mfclients.EnabledStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that are shared with me with an invalid name",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Name:   "notpresentclient",
				Status: mfclients.EnabledStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{},
			},
			size: 0,
		},
		{
			desc:  "list clients that I own",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Status: mfclients.EnabledStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that I own with a specific name",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Name:   "TestListClients3",
				Status: mfclients.AllStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that I own with an invalid name",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Name:   "notpresentclient",
				Status: mfclients.AllStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{},
			},
			size: 0,
		},
		{
			desc:  "list clients that I own and are shared with me",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Status: mfclients.AllStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that I own and are shared with me with a specific name",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Name:   "TestListClients3",
				Status: mfclients.AllStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  4,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{aClients[0], aClients[50], aClients[100], aClients[150]},
			},
			size: 4,
		},
		{
			desc:  "list clients that I own and are shared with me with an invalid name",
			token: validToken,
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Name:   "notpresentclient",
				Status: mfclients.AllStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mfclients.Client{},
			},
			size: 0,
		},
		{
			desc:  "list clients with offset and limit",
			token: validToken,

			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Status: mfclients.AllStatus,
			},
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
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
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&mainflux.ListObjectsRes{}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&mainflux.ListObjectsRes{}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.response, tc.err)
		page, err := svc.ListClients(context.Background(), tc.token, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	svc, cRepo, auth := newService()

	client1 := client
	client2 := client
	client1.Name = "Updated client"
	client2.Metadata = mfclients.Metadata{"role": "test"}

	cases := []struct {
		desc     string
		client   mfclients.Client
		response mfclients.Client
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
			response: mfclients.Client{},
			token:    "non-existent",
			err:      errors.ErrAuthentication,
		},
		{
			desc: "update client name with invalid ID",
			client: mfclients.Client{
				ID:   mocks.WrongID,
				Name: "Updated Client",
			},
			response: mfclients.Client{},
			token:    validToken,
			err:      errors.ErrNotFound,
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
			response: mfclients.Client{},
			token:    "non-existent",
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(mfclients.Client{}, tc.err)
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
	svc, cRepo, auth := newService()

	client.Tags = []string{"updated"}

	cases := []struct {
		desc     string
		client   mfclients.Client
		response mfclients.Client
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
			response: mfclients.Client{},
			err:      errors.ErrAuthentication,
		},
		{
			desc: "update client name with invalid ID",
			client: mfclients.Client{
				ID:   mocks.WrongID,
				Name: "Updated name",
			},
			response: mfclients.Client{},
			token:    validToken,
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(mfclients.Client{}, tc.err)
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

func TestUpdateClientOwner(t *testing.T) {
	svc, cRepo, auth := newService()

	client.Owner = "newowner@mail.com"

	cases := []struct {
		desc     string
		client   mfclients.Client
		response mfclients.Client
		token    string
		err      error
	}{
		{
			desc:     "update client owner with valid token",
			client:   client,
			token:    validToken,
			response: client,
			err:      nil,
		},
		{
			desc:     "update client owner with invalid token",
			client:   client,
			token:    "non-existent",
			response: mfclients.Client{},
			err:      errors.ErrAuthentication,
		},
		{
			desc: "update client owner with invalid ID",
			client: mfclients.Client{
				ID:    mocks.WrongID,
				Owner: "updatedowner@mail.com",
			},
			response: mfclients.Client{},
			token:    validToken,
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(mfclients.Client{}, tc.err)
		repoCall3 := cRepo.On("UpdateOwner", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientOwner(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	svc, cRepo, auth := newService()

	cases := []struct {
		desc      string
		id        string
		newSecret string
		token     string
		response  mfclients.Client
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
			response:  mfclients.Client{},
			err:       errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
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
	svc, cRepo, auth := newService()

	enabledClient1 := mfclients.Client{ID: ID, Credentials: mfclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mfclients.EnabledStatus}
	disabledClient1 := mfclients.Client{ID: ID, Credentials: mfclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mfclients.DisabledStatus}
	endisabledClient1 := disabledClient1
	endisabledClient1.Status = mfclients.EnabledStatus

	cases := []struct {
		desc     string
		id       string
		token    string
		client   mfclients.Client
		response mfclients.Client
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
			err:      mfclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "enable non-existing client",
			id:       mocks.WrongID,
			token:    validToken,
			client:   mfclients.Client{},
			response: mfclients.Client{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
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
		status   mfclients.Status
		size     uint64
		response mfclients.ClientsPage
	}{
		{
			desc:   "list enabled clients",
			status: mfclients.EnabledStatus,
			size:   2,
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mfclients.Client{enabledClient1, endisabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mfclients.DisabledStatus,
			size:   1,
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mfclients.Client{disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mfclients.AllStatus,
			size:   3,
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  3,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mfclients.Client{enabledClient1, disabledClient1, endisabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := mfclients.Page{
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&mainflux.ListObjectsRes{}, nil)
		repoCall2 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	svc, cRepo, auth := newService()

	enabledClient1 := mfclients.Client{ID: ID, Credentials: mfclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mfclients.EnabledStatus}
	disabledClient1 := mfclients.Client{ID: ID, Credentials: mfclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mfclients.DisabledStatus}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = mfclients.DisabledStatus

	cases := []struct {
		desc     string
		id       string
		token    string
		client   mfclients.Client
		response mfclients.Client
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
			response: mfclients.Client{},
			err:      mfclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "disable non-existing client",
			id:       mocks.WrongID,
			client:   mfclients.Client{},
			token:    validToken,
			response: mfclients.Client{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.client, tc.err)
		repoCall3 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		_, err := svc.DisableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}

	cases2 := []struct {
		desc     string
		status   mfclients.Status
		size     uint64
		response mfclients.ClientsPage
	}{
		{
			desc:   "list enabled clients",
			status: mfclients.EnabledStatus,
			size:   1,
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  1,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mfclients.Client{enabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mfclients.DisabledStatus,
			size:   2,
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  2,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mfclients.Client{disenabledClient1, disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mfclients.AllStatus,
			size:   3,
			response: mfclients.ClientsPage{
				Page: mfclients.Page{
					Total:  3,
					Offset: 0,
					Limit:  100,
				},
				Clients: []mfclients.Client{enabledClient1, disabledClient1, disenabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := mfclients.Page{
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&mainflux.ListObjectsRes{}, nil)
		repoCall2 := cRepo.On("RetrieveAllByIDs", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), validToken, pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestListMembers(t *testing.T) {
	svc, cRepo, auth := newService()

	nClients := uint64(10)
	aClients := []mfclients.Client{}
	owner := testsutil.GenerateUUID(t, idProvider)
	for i := uint64(0); i < nClients; i++ {
		identity := fmt.Sprintf("member_%d@example.com", i)
		client := mfclients.Client{
			ID:   testsutil.GenerateUUID(t, idProvider),
			Name: identity,
			Credentials: mfclients.Credentials{
				Identity: identity,
				Secret:   "password",
			},
			Tags:     []string{"tag1", "tag2"},
			Metadata: mfclients.Metadata{"role": "client"},
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
		page     mfclients.Page
		response mfclients.MembersPage
		err      error
	}{
		{
			desc:    "list clients with authorized token",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t, idProvider),
			page: mfclients.Page{
				Owner: adminEmail,
			},
			response: mfclients.MembersPage{
				Page: mfclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Members: []mfclients.Client{},
			},
			err: nil,
		},
		{
			desc:    "list clients with offset and limit",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t, idProvider),
			page: mfclients.Page{
				Offset: 6,
				Limit:  nClients,
				Status: mfclients.AllStatus,
				Owner:  adminEmail,
			},
			response: mfclients.MembersPage{
				Page: mfclients.Page{
					Total: nClients - 6 - 1,
				},
				Members: aClients[6 : nClients-1],
			},
		},
		{
			desc:    "list clients with an invalid token",
			token:   inValidToken,
			groupID: testsutil.GenerateUUID(t, idProvider),
			page: mfclients.Page{
				Owner: adminEmail,
			},
			response: mfclients.MembersPage{
				Page: mfclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			err: errors.ErrAuthentication,
		},
		{
			desc:    "list clients with an invalid id",
			token:   validToken,
			groupID: mocks.WrongID,
			page: mfclients.Page{
				Owner: adminEmail,
			},
			response: mfclients.MembersPage{
				Page: mfclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
			},
			err: errors.ErrNotFound,
		},
		{
			desc:    "list clients for an owner",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t, idProvider),
			page: mfclients.Page{
				Owner: adminEmail,
			},
			response: mfclients.MembersPage{
				Page: mfclients.Page{
					Total: 4,
				},
				Members: []mfclients.Client{aClients[0], aClients[3], aClients[6], aClients[9]},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: validToken}).Return(&mainflux.IdentityRes{Id: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &mainflux.IdentityReq{Token: inValidToken}).Return(&mainflux.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&mainflux.ListObjectsRes{}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", context.Background(), tc.page).Return(mfclients.ClientsPage{Page: tc.response.Page, Clients: tc.response.Members}, tc.err)
		page, err := svc.ListClientsByGroup(context.Background(), tc.token, tc.groupID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}
