// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/users"
	"github.com/absmach/magistrala/users/hasher"
	"github.com/absmach/magistrala/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	idProvider     = uuid.New()
	phasher        = hasher.New()
	secret         = "strongsecret"
	validCMetadata = mgclients.Metadata{"role": "client"}
	client         = mgclients.Client{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mgclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	passRegex    = regexp.MustCompile("^.{8,}$")
	myKey        = "mine"
	validToken   = "token"
	inValidToken = "invalid"
	validID      = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	domainID     = testsutil.GenerateUUID(&testing.T{})
	wrongID      = testsutil.GenerateUUID(&testing.T{})
)

func TestRegisterClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

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
			err:    errors.ErrConflict,
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
			err:   errors.ErrMalformedEntity,
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
			err:   errors.ErrMalformedEntity,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: inValidToken}).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(&magistrala.AddPoliciesRes{Authorized: true}, nil)
		repoCall2 := auth.On("DeletePolicies", mock.Anything, mock.Anything).Return(&magistrala.DeletePoliciesRes{Deleted: true}, nil)
		repoCall3 := cRepo.On("Save", context.Background(), mock.Anything).Return(tc.client, tc.err)
		expected, err := svc.RegisterClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.client.ID = expected.ID
			tc.client.CreatedAt = expected.CreatedAt
			tc.client.UpdatedAt = expected.UpdatedAt
			tc.client.Credentials.Secret = expected.Credentials.Secret
			tc.client.Owner = expected.Owner
			tc.client.UpdatedBy = expected.UpdatedBy
			assert.Equal(t, tc.client, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, expected))
			ok := repoCall3.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall3.Unset()
		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestViewClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

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
			token:    inValidToken,
			clientID: client.ID,
			err:      svcerr.ErrAuthentication,
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
			token:    inValidToken,
			clientID: wrongID,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: inValidToken}).Return(&magistrala.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), tc.clientID).Return(tc.response, tc.err)

		rClient, err := svc.ViewClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.clientID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}

		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestListClients(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

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
				Clients: []mgclients.Client{},
			},
			err: nil,
		},
		{
			desc:  "list clients with an invalid token",
			token: inValidToken,
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
				Offset: 6,
				Limit:  nClients,
				Name:   "notpresentclient",
				Status: mgclients.EnabledStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{},
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
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Name:   "notpresentclient",
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{},
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
				Offset: 6,
				Limit:  nClients,
				Owner:  myKey,
				Name:   "notpresentclient",
				Status: mgclients.AllStatus,
			},
			response: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  0,
				},
				Clients: []mgclients.Client{},
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: inValidToken}).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, tc.err)
		page, err := svc.ListClients(context.Background(), tc.token, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "RetrieveAll", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

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
			token:    inValidToken,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "update client name with invalid ID",
			client: mgclients.Client{
				ID:   wrongID,
				Name: "Updated Client",
			},
			response: mgclients.Client{},
			token:    inValidToken,
			err:      svcerr.ErrAuthentication,
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
			token:    inValidToken,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: inValidToken}).Return(&magistrala.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

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
			token:    inValidToken,
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
			token:    inValidToken,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: inValidToken}).Return(&magistrala.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientTags(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "UpdateTags", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClientIdentity(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

	client2 := client
	client2.Credentials.Identity = "updated@example.com"

	cases := []struct {
		desc     string
		identity string
		response mgclients.Client
		token    string
		id       string
		err      error
	}{
		{
			desc:     "update client identity with valid token",
			identity: "updated@example.com",
			token:    validToken,
			id:       client.ID,
			response: client2,
			err:      nil,
		},
		{
			desc:     "update client identity with invalid id",
			identity: "updated@example.com",
			token:    validToken,
			id:       wrongID,
			response: mgclients.Client{},
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "update client identity with invalid token",
			identity: "updated@example.com",
			token:    inValidToken,
			id:       client2.ID,
			response: mgclients.Client{},
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: inValidToken}).Return(&magistrala.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall2 := cRepo.On("UpdateIdentity", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientIdentity(context.Background(), tc.token, tc.id, tc.identity)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "UpdateIdentity", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClientRole(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

	client.Role = mgclients.AdminRole

	cases := []struct {
		desc     string
		client   mgclients.Client
		response mgclients.Client
		token    string
		err      error
	}{
		{
			desc:     "update client role with valid token",
			client:   client,
			token:    validToken,
			response: client,
			err:      nil,
		},
		{
			desc:     "update client role with invalid token",
			client:   client,
			token:    inValidToken,
			response: mgclients.Client{},
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "update client role with invalid ID",
			client: mgclients.Client{
				ID:   wrongID,
				Role: mgclients.AdminRole,
			},
			response: mgclients.Client{},
			token:    inValidToken,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("DeletePolicy", mock.Anything, mock.Anything).Return(&magistrala.DeletePolicyRes{Deleted: true}, nil)
		repoCall3 := auth.On("AddPolicy", mock.Anything, mock.Anything).Return(&magistrala.AddPolicyRes{Authorized: true}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: inValidToken}).Return(&magistrala.IdentityRes{}, errors.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, errors.ErrAuthorization)
		}
		repoCall4 := cRepo.On("UpdateRole", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientRole(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall4.Parent.AssertCalled(t, "UpdateRole", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateRole was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

	cases := []struct {
		desc      string
		oldSecret string
		newSecret string
		token     string
		response  mgclients.Client
		err       error
	}{
		{
			desc:      "update client secret with valid token",
			oldSecret: client.Credentials.Secret,
			newSecret: "newSecret",
			token:     validToken,
			response:  rClient,
			err:       nil,
		},
		{
			desc:      "update client secret with invalid token",
			oldSecret: client.Credentials.Secret,
			newSecret: "newPassword",
			token:     inValidToken,
			response:  mgclients.Client{},
			err:       svcerr.ErrAuthentication,
		},
		{
			desc:      "update client secret with wrong old secret",
			oldSecret: "oldSecret",
			newSecret: "newSecret",
			token:     validToken,
			response:  mgclients.Client{},
			err:       repoerr.ErrInvalidSecret,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: client.ID}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: inValidToken}).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), client.ID).Return(tc.response, tc.err)
		repoCall2 := cRepo.On("RetrieveByIdentity", context.Background(), client.Credentials.Identity).Return(tc.response, tc.err)
		repoCall3 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.response, tc.err)
		repoCall4 := auth.On("Issue", mock.Anything, mock.Anything).Return(&magistrala.Token{}, nil)
		updatedClient, err := svc.UpdateClientSecret(context.Background(), tc.token, tc.oldSecret, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.response.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.response.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

	enabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.client, tc.err)
		repoCall3 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		_, err := svc.EnableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: client.ID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, nil)
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
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

	enabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus}
	disabledClient1 := mgclients.Client{ID: testsutil.GenerateUUID(t), Credentials: mgclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus}
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.client, tc.err)
		repoCall3 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		_, err := svc.DisableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: client.ID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, nil)
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
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

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
				IDs: clientsToUUIDs(aClients),
			},
			response: mgclients.MembersPage{
				Page: mgclients.Page{
					Total:  0,
					Offset: 0,
					Limit:  10,
				},
				Members: aClients,
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
				IDs:    clientsToUUIDs(aClients[6 : nClients-1]),
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
			token:   inValidToken,
			groupID: testsutil.GenerateUUID(t),
			page:    mgclients.Page{},
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
			desc:    "list clients for an owner",
			token:   validToken,
			groupID: testsutil.GenerateUUID(t),
			page: mgclients.Page{
				IDs: clientsToUUIDs([]mgclients.Client{aClients[0], aClients[3], aClients[6], aClients[9]}),
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
		repoCall := auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, nil)
		if tc.token == inValidToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, errors.ErrAuthentication)
		}
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true, Id: validID}, nil)

		repoCall2 := auth.On("ListAllSubjects", mock.Anything, mock.Anything).Return(&magistrala.ListSubjectsRes{Policies: prefixClientUUIDSWithDomain(tc.response.Members)}, nil)
		repoCall3 := cRepo.On("RetrieveAll", context.Background(), tc.page).Return(mgclients.ClientsPage{Page: tc.response.Page, Clients: tc.response.Members}, tc.err)
		page, err := svc.ListMembers(context.Background(), tc.token, "groups", tc.groupID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveAll", context.Background(), tc.page)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestIssueToken(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

	rClient := client
	rClient2 := client
	rClient3 := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)
	rClient2.Credentials.Secret = "wrongsecret"
	rClient3.Credentials.Secret, _ = phasher.Hash("wrongsecret")

	cases := []struct {
		desc    string
		client  mgclients.Client
		rClient mgclients.Client
		err     error
	}{
		{
			desc:    "issue token for an existing client",
			client:  client,
			rClient: rClient,
			err:     nil,
		},
		{
			desc:    "issue token for a non-existing client",
			client:  client,
			rClient: mgclients.Client{},
			err:     svcerr.ErrAuthentication,
		},
		{
			desc:    "issue token for a client with wrong secret",
			client:  rClient2,
			rClient: rClient3,
			err:     errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Issue", mock.Anything, mock.Anything).Return(&magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"}, tc.err)
		repoCall1 := cRepo.On("RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity).Return(tc.rClient, tc.err)
		token, err := svc.IssueToken(context.Background(), tc.client.Credentials.Identity, tc.client.Credentials.Secret, "")
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestRefreshToken(t *testing.T) {
	cRepo := new(mocks.Repository)
	auth := new(authmocks.Service)
	e := mocks.NewEmailer()
	svc := users.NewService(cRepo, auth, e, phasher, idProvider, passRegex, true)

	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

	cases := []struct {
		desc   string
		token  string
		client mgclients.Client
		err    error
	}{
		{
			desc:   "refresh token with refresh token for an existing client",
			token:  validToken,
			client: client,
			err:    nil,
		},
		{
			desc:   "refresh token with refresh token for a non-existing client",
			token:  validToken,
			client: mgclients.Client{},
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "refresh token with access token for an existing client",
			token:  validToken,
			client: client,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "refresh token with access token for a non-existing client",
			token:  validToken,
			client: mgclients.Client{},
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "refresh token with invalid token for an existing client",
			token:  inValidToken,
			client: client,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Refresh", mock.Anything, mock.Anything).Return(&magistrala.Token{AccessToken: validToken, RefreshToken: &validToken, AccessType: "3"}, tc.err)
		token, err := svc.RefreshToken(context.Background(), tc.token, "")
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.GetAccessToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetAccessToken()))
			assert.NotEmpty(t, token.GetRefreshToken(), fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.GetRefreshToken()))
		}
		repoCall.Unset()
	}
}

func clientsToUUIDs(clients []mgclients.Client) []string {
	ids := []string{}
	for _, c := range clients {
		ids = append(ids, c.ID)
	}
	return ids
}

func prefixClientUUIDSWithDomain(clients []mgclients.Client) []string {
	ids := []string{}
	for _, c := range clients {
		ids = append(ids, fmt.Sprintf("%s_%s", domainID, c.ID))
	}
	return ids
}
