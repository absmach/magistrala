// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package clients_test

import (
	context "context"
	fmt "fmt"
	"regexp"
	"testing"
	"time"

	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/internal/testsutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users/clients"
	"github.com/mainflux/mainflux/users/clients/mocks"
	"github.com/mainflux/mainflux/users/hasher"
	"github.com/mainflux/mainflux/users/jwt"
	pmocks "github.com/mainflux/mainflux/users/policies/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	idProvider     = uuid.New()
	phasher        = hasher.New()
	secret         = "strongsecret"
	validCMetadata = mfclients.Metadata{"role": "client"}
	client         = mfclients.Client{
		ID:          testsutil.GenerateUUID(&testing.T{}, idProvider),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mfclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mfclients.EnabledStatus,
	}
	inValidToken    = "invalidToken"
	withinDuration  = 5 * time.Second
	passRegex       = regexp.MustCompile("^.{8,}$")
	accessDuration  = time.Minute * 1
	refreshDuration = time.Minute * 10
	myKey           = "mine"
)

func TestRegisterClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	cases := []struct {
		desc   string
		client mfclients.Client
		token  string
		err    error
	}{
		{
			desc:   "register new client",
			client: client,
			token:  testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			err:    nil,
		},
		{
			desc:   "register existing client",
			client: client,
			token:  testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("Save", context.Background(), mock.Anything).Return(&mfclients.Client{}, tc.err)
		registerTime := time.Now()
		expected, err := svc.RegisterClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, expected.ID, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, expected.ID))
			assert.WithinDuration(t, expected.CreatedAt, registerTime, withinDuration, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected.CreatedAt, registerTime))
			tc.client.ID = expected.ID
			tc.client.CreatedAt = expected.CreatedAt
			tc.client.UpdatedAt = expected.UpdatedAt
			tc.client.Credentials.Secret = expected.Credentials.Secret
			tc.client.Owner = expected.Owner
			tc.client.UpdatedBy = expected.UpdatedBy
			assert.Equal(t, tc.client, expected, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.client, expected))
			ok := repoCall.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
	}
}

func TestViewClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

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
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			clientID: mocks.WrongID,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "view client with an invalid token and invalid client id",
			response: mfclients.Client{},
			token:    inValidToken,
			clientID: mocks.WrongID,
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("Evaluate", context.Background(), "client", mock.Anything).Return(nil)
		repoCall1 := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), tc.clientID).Return(tc.response, tc.err)
		rClient, err := svc.ViewClient(context.Background(), tc.token, tc.clientID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.clientID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestListClients(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	var nClients = uint64(200)
	var aClients = []mfclients.Client{}
	var OwnerID = testsutil.GenerateUUID(t, idProvider)
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),

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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			page: mfclients.Page{
				Offset:   6,
				Limit:    nClients,
				SharedBy: myKey,
				Status:   mfclients.EnabledStatus,
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			page: mfclients.Page{
				Offset:   6,
				Limit:    nClients,
				SharedBy: myKey,
				Name:     "TestListClients3",
				Status:   mfclients.EnabledStatus,
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			page: mfclients.Page{
				Offset:   6,
				Limit:    nClients,
				SharedBy: myKey,
				Name:     "notpresentclient",
				Status:   mfclients.EnabledStatus,
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			page: mfclients.Page{
				Offset:   6,
				Limit:    nClients,
				Owner:    myKey,
				SharedBy: myKey,
				Status:   mfclients.AllStatus,
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			page: mfclients.Page{
				Offset:   6,
				Limit:    nClients,
				SharedBy: myKey,
				Owner:    myKey,
				Name:     "TestListClients3",
				Status:   mfclients.AllStatus,
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			page: mfclients.Page{
				Offset:   6,
				Limit:    nClients,
				SharedBy: myKey,
				Owner:    myKey,
				Name:     "notpresentclient",
				Status:   mfclients.AllStatus,
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
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),

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
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(tc.err)
		repoCall1 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, tc.err)
		page, err := svc.ListClients(context.Background(), tc.token, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveAll", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

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
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token:    "non-existent",
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "update client metadata with valid token",
			client:   client2,
			response: client2,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := cRepo.On("Update", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

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
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token:    "non-existent",
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := cRepo.On("UpdateTags", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientTags(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "UpdateTags", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientIdentity(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	client2 := client
	client2.Credentials.Identity = "updated@example.com"

	cases := []struct {
		desc     string
		identity string
		response mfclients.Client
		token    string
		id       string
		err      error
	}{
		{
			desc:     "update client identity with valid token",
			identity: "updated@example.com",
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			id:       client.ID,
			response: client2,
			err:      nil,
		},
		{
			desc:     "update client identity with invalid id",
			identity: "updated@example.com",
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			id:       mocks.WrongID,
			response: mfclients.Client{},
			err:      errors.ErrNotFound,
		},
		{
			desc:     "update client identity with invalid token",
			identity: "updated@example.com",
			token:    "non-existent",
			id:       client2.ID,
			response: mfclients.Client{},
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := cRepo.On("UpdateIdentity", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientIdentity(context.Background(), tc.token, tc.id, tc.identity)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "UpdateIdentity", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientOwner(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

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
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
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
			token:    "non-existent",
			err:      errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := cRepo.On("UpdateOwner", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientOwner(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "UpdateOwner", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateOwner was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

	repoCall := cRepo.On("RetrieveByIdentity", context.Background(), client.Credentials.Identity).Return(rClient, nil)
	token, err := svc.IssueToken(context.Background(), client.Credentials.Identity, client.Credentials.Secret)
	assert.Nil(t, err, fmt.Sprintf("Issue token expected nil got %s\n", err))
	repoCall.Unset()

	cases := []struct {
		desc      string
		oldSecret string
		newSecret string
		token     string
		response  mfclients.Client
		err       error
	}{
		{
			desc:      "update client secret with valid token",
			oldSecret: client.Credentials.Secret,
			newSecret: "newSecret",
			token:     token.AccessToken,
			response:  rClient,
			err:       nil,
		},
		{
			desc:      "update client secret with invalid token",
			oldSecret: client.Credentials.Secret,
			newSecret: "newPassword",
			token:     "non-existent",
			response:  mfclients.Client{},
			err:       errors.ErrAuthentication,
		},
		{
			desc:      "update client secret with wrong old secret",
			oldSecret: "oldSecret",
			newSecret: "newSecret",
			token:     token.AccessToken,
			response:  mfclients.Client{},
			err:       apiutil.ErrInvalidSecret,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", context.Background(), client.ID).Return(tc.response, tc.err)
		repoCall1 := cRepo.On("RetrieveByIdentity", context.Background(), client.Credentials.Identity).Return(tc.response, tc.err)
		repoCall2 := cRepo.On("UpdateSecret", context.Background(), mock.Anything).Return(tc.response, tc.err)
		updatedClient, err := svc.UpdateClientSecret(context.Background(), tc.token, tc.oldSecret, tc.newSecret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, updatedClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, updatedClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.response.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.response.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "UpdateSecret", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	enabledClient1 := mfclients.Client{ID: testsutil.GenerateUUID(t, idProvider), Credentials: mfclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mfclients.EnabledStatus}
	disabledClient1 := mfclients.Client{ID: testsutil.GenerateUUID(t, idProvider), Credentials: mfclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mfclients.DisabledStatus}
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
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			client:   disabledClient1,
			response: endisabledClient1,
			err:      nil,
		},
		{
			desc:     "enable enabled client",
			id:       enabledClient1.ID,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			client:   enabledClient1,
			response: enabledClient1,
			err:      mfclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "enable non-existing client",
			id:       mocks.WrongID,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			client:   mfclients.Client{},
			response: mfclients.Client{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.client, tc.err)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		_, err := svc.EnableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
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
			Action: "c_list",
		}
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher), pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	enabledClient1 := mfclients.Client{ID: testsutil.GenerateUUID(t, idProvider), Credentials: mfclients.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mfclients.EnabledStatus}
	disabledClient1 := mfclients.Client{ID: testsutil.GenerateUUID(t, idProvider), Credentials: mfclients.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mfclients.DisabledStatus}
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
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			client:   enabledClient1,
			response: disenabledClient1,
			err:      nil,
		},
		{
			desc:     "disable disabled client",
			id:       disabledClient1.ID,
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			client:   disabledClient1,
			response: mfclients.Client{},
			err:      mfclients.ErrStatusAlreadyAssigned,
		},
		{
			desc:     "disable non-existing client",
			id:       mocks.WrongID,
			client:   mfclients.Client{},
			token:    testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			response: mfclients.Client{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := cRepo.On("RetrieveByID", context.Background(), tc.id).Return(tc.client, tc.err)
		repoCall2 := cRepo.On("ChangeStatus", context.Background(), mock.Anything).Return(tc.response, tc.err)
		_, err := svc.DisableClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
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
			Action: "c_list",
		}
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := cRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(tc.response, nil)
		page, err := svc.ListClients(context.Background(), testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher), pm)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Clients))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListMembers(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	var nClients = uint64(10)
	var aClients = []mfclients.Client{}
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
	validID := testsutil.GenerateUUID(t, idProvider)
	validToken := testsutil.GenerateValidToken(t, validID, svc, cRepo, phasher)

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
				Subject: validID,
				Action:  "g_list",
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
				Offset:  6,
				Limit:   nClients,
				Status:  mfclients.AllStatus,
				Subject: validID,
				Action:  "g_list",
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
				Subject: validID,
				Action:  "g_list",
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
				Subject: validID,
				Action:  "g_list",
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
				Subject: owner,
				Action:  "g_list",
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
		repoCall := pRepo.On("CheckAdmin", context.Background(), validID).Return(nil)
		repoCall1 := cRepo.On("Members", context.Background(), tc.groupID, tc.page).Return(tc.response, tc.err)
		page, err := svc.ListMembers(context.Background(), tc.token, tc.groupID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), validID)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "Members", context.Background(), tc.groupID, tc.page)
			assert.True(t, ok, fmt.Sprintf("Members was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestIssueToken(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	rClient := client
	rClient2 := client
	rClient3 := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)
	rClient2.Credentials.Secret = "wrongsecret"
	rClient3.Credentials.Secret, _ = phasher.Hash("wrongsecret")

	cases := []struct {
		desc    string
		client  mfclients.Client
		rClient mfclients.Client
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
			rClient: mfclients.Client{},
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "issue token for a client with wrong secret",
			client:  rClient2,
			rClient: rClient3,
			err:     errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity).Return(tc.rClient, tc.err)
		token, err := svc.IssueToken(context.Background(), tc.client.Credentials.Identity, tc.client.Credentials.Secret)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.AccessToken, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.AccessToken))
			assert.NotEmpty(t, token.RefreshToken, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.RefreshToken))
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
	}
}

func TestRefreshToken(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	svc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)

	rClient := client
	rClient.Credentials.Secret, _ = phasher.Hash(client.Credentials.Secret)

	repoCall := cRepo.On("RetrieveByIdentity", context.Background(), client.Credentials.Identity).Return(rClient, nil)
	token, err := svc.IssueToken(context.Background(), client.Credentials.Identity, client.Credentials.Secret)
	assert.Nil(t, err, fmt.Sprintf("Issue token expected nil got %s\n", err))
	repoCall.Unset()

	cases := []struct {
		desc   string
		token  string
		client mfclients.Client
		err    error
	}{
		{
			desc:   "refresh token with refresh token for an existing client",
			token:  token.RefreshToken,
			client: client,
			err:    nil,
		},
		{
			desc:   "refresh token with refresh token for a non-existing client",
			token:  token.RefreshToken,
			client: mfclients.Client{},
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "refresh token with access token for an existing client",
			token:  token.AccessToken,
			client: client,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "refresh token with access token for a non-existing client",
			token:  token.AccessToken,
			client: mfclients.Client{},
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "refresh token with invalid token for an existing client",
			token:  testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), svc, cRepo, phasher),
			client: client,
			err:    errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall1 := cRepo.On("RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity).Return(tc.client, nil)
		repoCall2 := cRepo.On("RetrieveByID", context.Background(), mock.Anything).Return(tc.client, tc.err)
		token, err := svc.RefreshToken(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.NotEmpty(t, token.AccessToken, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.AccessToken))
			assert.NotEmpty(t, token.RefreshToken, fmt.Sprintf("%s: expected %s not to be empty\n", tc.desc, token.RefreshToken))
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByIdentity", context.Background(), tc.client.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "RetrieveByID", context.Background(), tc.client.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall1.Unset()
		repoCall2.Unset()
	}
}
