// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/groups"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	oauth2mocks "github.com/absmach/magistrala/pkg/oauth2/mocks"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/users"
	"github.com/absmach/magistrala/users/api"
	umocks "github.com/absmach/magistrala/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	id         = generateUUID(&testing.T{})
	validToken = "token"
	adminToken = "adminToken"
	validID    = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID    = testsutil.GenerateUUID(&testing.T{})
)

func setupUsers() (*httptest.Server, *umocks.Repository, *gmocks.Repository, *authmocks.AuthClient) {
	crepo := new(umocks.Repository)
	gRepo := new(gmocks.Repository)

	auth := new(authmocks.AuthClient)
	csvc := users.NewService(crepo, auth, emailer, phasher, idProvider, constraintsProvider, true)
	gsvc := groups.NewService(gRepo, idProvider, constraintsProvider, auth)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	api.MakeHandler(csvc, gsvc, mux, logger, "", passRegex, provider)

	return httptest.NewServer(mux), crepo, gRepo, auth
}

func TestCreateClient(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	user := sdk.User{
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "admin@example.com", Secret: "12345678"},
		Status:      mgclients.EnabledStatus.String(),
	}
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		total    uint64
		client   sdk.User
		response sdk.User
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "register new user",
			client:   user,
			response: user,
			token:    token,
			err:      nil,
		},
		{
			desc:     "register existing user",
			client:   user,
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "register empty user",
			client:   sdk.User{},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusBadRequest),
		},
		{
			desc: "register a user that can't be marshalled",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
		{
			desc: "register user with invalid identity",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: wrongID,
					Secret:   "password",
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity), http.StatusBadRequest),
		},
		{
			desc: "register user with empty secret",
			client: sdk.User{
				Name: "emptysecret",
				Credentials: sdk.Credentials{
					Secret: "",
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusBadRequest),
		},
		{
			desc: "register user with empty identity",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "",
					Secret:   secret,
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusBadRequest),
		},
		{
			desc: "register user with every field defined",
			client: sdk.User{
				ID:          id,
				Name:        "name",
				Tags:        []string{"tag1", "tag2"},
				Credentials: user.Credentials,
				Metadata:    validMetadata,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      mgclients.EnabledStatus.String(),
			},
			response: sdk.User{
				ID:          id,
				Name:        "name",
				Tags:        []string{"tag1", "tag2"},
				Credentials: user.Credentials,
				Metadata:    validMetadata,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      mgclients.EnabledStatus.String(),
			},
			token: token,
			err:   nil,
		},
	}
	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(&magistrala.AddPoliciesRes{Added: true}, nil)
		repoCall2 := auth.On("DeletePolicies", mock.Anything, mock.Anything).Return(&magistrala.DeletePoliciesRes{Deleted: true}, nil)
		repoCall3 := crepo.On("Save", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		retrieveAllCall := crepo.On("RetrieveAll", mock.Anything, mgclients.Page{}).Return(mgclients.ClientsPage{Page: mgclients.Page{Total: tc.total}}, nil)
		rClient, err := mgsdk.CreateUser(tc.client, tc.token)
		tc.response.ID = rClient.ID
		tc.response.CreatedAt = rClient.CreatedAt
		tc.response.UpdatedAt = rClient.UpdatedAt
		rClient.Credentials.Secret = tc.response.Credentials.Secret
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall3.Unset()
		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
		retrieveAllCall.Unset()
	}
}

func TestListClients(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	var cls []sdk.User
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		cl := sdk.User{
			ID:   generateUUID(t),
			Name: fmt.Sprintf("client_%d", i),
			Credentials: sdk.Credentials{
				Identity: fmt.Sprintf("identity_%d", i),
				Secret:   fmt.Sprintf("password_%d", i),
			},
			Metadata: sdk.Metadata{"name": fmt.Sprintf("client_%d", i)},
			Status:   mgclients.EnabledStatus.String(),
		}
		if i == 50 {
			cl.Status = mgclients.DisabledStatus.String()
			cl.Tags = []string{"tag1", "tag2"}
		}
		cls = append(cls, cl)
	}

	cases := []struct {
		desc       string
		token      string
		status     string
		total      uint64
		offset     uint64
		limit      uint64
		name       string
		identifier string
		tag        string
		metadata   sdk.Metadata
		err        errors.SDKError
		response   []sdk.User
	}{
		{
			desc:     "get a list of users",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			err:      nil,
			response: cls[offset:limit],
		},
		{
			desc:     "get a list of users with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of users with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of users with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			response: nil,
		},
		{
			desc:     "get a list of users with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
			response: []sdk.User(nil),
		},
		{
			desc:       "get a list of users with same identity",
			token:      token,
			offset:     0,
			limit:      1,
			err:        nil,
			identifier: Identity,
			metadata:   sdk.Metadata{},
			response:   []sdk.User{cls[89]},
		},
		{
			desc:       "get a list of users with same identity and metadata",
			token:      token,
			offset:     0,
			limit:      1,
			err:        nil,
			identifier: Identity,
			metadata: sdk.Metadata{
				"name": "client99",
			},
			response: []sdk.User{cls[89]},
		},
		{
			desc:   "list users with given metadata",
			token:  validToken,
			offset: 0,
			limit:  1,
			metadata: sdk.Metadata{
				"name": "client99",
			},
			response: []sdk.User{cls[89]},
			err:      nil,
		},
		{
			desc:     "list users with given name",
			token:    validToken,
			offset:   0,
			limit:    1,
			name:     "client10",
			response: []sdk.User{cls[0]},
			err:      nil,
		},

		{
			desc:     "list users with given status",
			token:    validToken,
			offset:   0,
			limit:    1,
			status:   mgclients.DisabledStatus.String(),
			response: []sdk.User{cls[50]},
			err:      nil,
		},
		{
			desc:     "list users with given tag",
			token:    validToken,
			offset:   0,
			limit:    1,
			tag:      "tag1",
			response: []sdk.User{cls[50]},
			err:      nil,
		},
	}

	for _, tc := range cases {
		pm := sdk.PageMetadata{
			Status:   tc.status,
			Offset:   tc.offset,
			Limit:    tc.limit,
			Name:     tc.name,
			Metadata: tc.metadata,
			Tag:      tc.tag,
		}

		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := crepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(mgclients.ClientsPage{Page: convertClientPage(pm), Clients: convertClients(tc.response)}, tc.err)
		page, err := mgsdk.Users(pm, validToken)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Users, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestClient(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	user = sdk.User{
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mgclients.EnabledStatus.String(),
	}

	basicUser := sdk.User{
		Name:   "clientname",
		Status: mgclients.EnabledStatus.String(),
	}
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc                 string
		token                string
		clientID             string
		response             sdk.User
		retrieveByIDResponse sdk.User
		err                  errors.SDKError
		authorizeErr         error
		retrieveByIDErr      error
		checkSuperAdminErr   errors.Error
		identifyErr          errors.Error
	}{
		{
			desc:               "view client successfully",
			response:           basicUser,
			token:              validToken,
			clientID:           generateUUID(t),
			authorizeErr:       svcerr.ErrAuthentication,
			checkSuperAdminErr: svcerr.ErrAuthentication,
			err:                nil,
		},
		{
			desc:     "view client successfully as admin",
			response: user,
			token:    adminToken,
			clientID: generateUUID(t),
			err:      nil,
		},
		{
			desc:               "view client with an invalid token",
			response:           sdk.User{},
			token:              invalidToken,
			clientID:           generateUUID(t),
			identifyErr:        svcerr.ErrAuthentication,
			authorizeErr:       svcerr.ErrAuthentication,
			checkSuperAdminErr: svcerr.ErrAuthentication,
			err:                errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			retrieveByIDErr:    svcerr.ErrAuthentication,
		},
		{
			desc:               "view client with valid token and invalid client id",
			response:           sdk.User{},
			token:              validToken,
			clientID:           wrongID,
			authorizeErr:       svcerr.ErrAuthentication,
			checkSuperAdminErr: svcerr.ErrAuthentication,
			err:                errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
			retrieveByIDErr:    svcerr.ErrViewEntity,
		},
		{
			desc:               "view client with an invalid token and invalid client id",
			response:           sdk.User{},
			token:              invalidToken,
			identifyErr:        svcerr.ErrAuthentication,
			authorizeErr:       svcerr.ErrAuthentication,
			clientID:           wrongID,
			checkSuperAdminErr: svcerr.ErrAuthentication,
			err:                errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			retrieveByIDErr:    svcerr.ErrAuthentication,
		},
		{
			desc:               "view client as normal user with failed check on admin",
			response:           basicUser,
			token:              validToken,
			authorizeErr:       svcerr.ErrAuthentication,
			checkSuperAdminErr: svcerr.ErrAuthentication,
			clientID:           generateUUID(t),
			err:                nil,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{UserId: validID}, tc.identifyErr)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, tc.authorizeErr)
		repoCall2 := crepo.On("RetrieveByID", mock.Anything, tc.clientID).Return(convertClient(tc.response), tc.err)
		superAdminCall := crepo.On("CheckSuperAdmin", mock.Anything, mock.Anything).Return(tc.checkSuperAdminErr)
		rClient, err := mgsdk.User(tc.clientID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
		superAdminCall.Unset()
	}
}

func TestProfile(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	user = sdk.User{
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mgclients.EnabledStatus.String(),
	}
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		response sdk.User
		err      errors.SDKError
	}{
		{
			desc:     "view client successfully",
			response: user,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "view client with an invalid token",
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCal1 := crepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		rClient, err := mgsdk.UserProfile(tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCal1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCal1.Unset()
		repoCall.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	user = sdk.User{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mgclients.EnabledStatus.String(),
	}

	client1 := user
	client1.Name = "Updated client"

	client2 := user
	client2.Metadata = sdk.Metadata{"role": "test"}
	client2.ID = invalidIdentity

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      errors.SDKError
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
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update client name with invalid id",
			client:   client2,
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "update a user that can't be marshalled",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := crepo.On("Update", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		uClient, err := mgsdk.UpdateUser(tc.client, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Identify was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	user = sdk.User{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mgclients.EnabledStatus.String(),
	}

	client1 := user
	client1.Tags = []string{"updatedTag1", "updatedTag2"}

	client2 := user
	client2.ID = invalidIdentity

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      error
	}{
		{
			desc:     "update client name with valid token",
			client:   user,
			response: client1,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client name with invalid token",
			client:   client1,
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update client name with invalid id",
			client:   client2,
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "update a user that can't be marshalled",
			client: sdk.User{
				ID: generateUUID(t),
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := crepo.On("UpdateTags", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		uClient, err := mgsdk.UpdateUserTags(tc.client, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Identify was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "UpdateTags", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClientIdentity(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	user = sdk.User{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "updatedclientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mgclients.EnabledStatus.String(),
	}

	client2 := user
	client2.Metadata = sdk.Metadata{"role": "test"}
	client2.ID = invalidIdentity

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "update client name with valid token",
			client:   user,
			response: user,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client name with invalid token",
			client:   user,
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update client name with invalid id",
			client:   client2,
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "update a user that can't be marshalled",
			client: sdk.User{
				ID: generateUUID(t),
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := crepo.On("UpdateIdentity", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		uClient, err := mgsdk.UpdateUserIdentity(tc.client, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Identify was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "UpdateIdentity", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	user.ID = generateUUID(t)
	rclient := user
	rclient.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)

	cases := []struct {
		desc      string
		oldSecret string
		newSecret string
		token     string
		response  sdk.User
		err       error
		repoErr   error
	}{
		{
			desc:      "update client secret with valid token",
			oldSecret: user.Credentials.Secret,
			newSecret: "newSecret",
			token:     validToken,
			response:  rclient,
			repoErr:   nil,
			err:       nil,
		},
		{
			desc:      "update client secret with invalid token",
			oldSecret: user.Credentials.Secret,
			newSecret: "newPassword",
			token:     "non-existent",
			response:  sdk.User{},
			repoErr:   svcerr.ErrAuthentication,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "update client secret with wrong old secret",
			oldSecret: "oldSecret",
			newSecret: "newSecret",
			token:     validToken,
			response:  sdk.User{},
			repoErr:   apiutil.ErrMissingSecret,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: user.ID}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := auth.On("Issue", mock.Anything, mock.Anything).Return(&magistrala.Token{AccessToken: validToken}, nil)
		repoCall2 := crepo.On("RetrieveByID", mock.Anything, user.ID).Return(convertClient(tc.response), tc.repoErr)
		repoCall3 := crepo.On("RetrieveByIdentity", mock.Anything, user.Credentials.Identity).Return(convertClient(tc.response), tc.repoErr)
		repoCall4 := crepo.On("UpdateSecret", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.repoErr)
		uClient, err := mgsdk.UpdatePassword(tc.oldSecret, tc.newSecret, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, user.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "RetrieveByIdentity", mock.Anything, user.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = repoCall4.Parent.AssertCalled(t, "UpdateSecret", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestUpdateClientRole(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	user = sdk.User{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mgclients.EnabledStatus.String(),
	}

	client2 := user
	client2.ID = invalidIdentity

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "update client name with valid token",
			client:   user,
			response: user,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client name with invalid token",
			client:   client2,
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update client name with invalid id",
			client:   client2,
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "update a user that can't be marshalled",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
		}
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("DeletePolicy", mock.Anything, mock.Anything).Return(&magistrala.DeletePolicyRes{Deleted: true}, nil)
		repoCall3 := auth.On("AddPolicy", mock.Anything, mock.Anything).Return(&magistrala.AddPolicyRes{Added: true}, nil)
		repoCall4 := crepo.On("UpdateRole", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		uClient, err := mgsdk.UpdateUserRole(tc.client, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Identify was not called on %s", tc.desc))
			ok = repoCall4.Parent.AssertCalled(t, "UpdateRole", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateRole was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enabledClient1 := sdk.User{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus.String()}
	disabledClient1 := sdk.User{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus.String()}
	endisabledClient1 := disabledClient1
	endisabledClient1.Status = mgclients.EnabledStatus.String()
	endisabledClient1.ID = testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		id       string
		token    string
		client   sdk.User
		response sdk.User
		repoErr  error
		err      errors.SDKError
	}{
		{
			desc:     "enable disabled client",
			id:       disabledClient1.ID,
			token:    validToken,
			client:   disabledClient1,
			response: endisabledClient1,
			repoErr:  nil,
			err:      nil,
		},
		{
			desc:     "enable enabled client",
			id:       enabledClient1.ID,
			token:    validToken,
			client:   enabledClient1,
			response: sdk.User{},
			repoErr:  sdk.ErrFailedEnable,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrEnableClient, http.StatusBadRequest),
		},
		{
			desc:     "enable non-existing client",
			id:       wrongID,
			token:    validToken,
			client:   sdk.User{},
			response: sdk.User{},
			repoErr:  sdk.ErrFailedEnable,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrEnableClient, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := crepo.On("RetrieveByID", mock.Anything, tc.id).Return(convertClient(tc.client), tc.repoErr)
		repoCall3 := crepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.repoErr)
		eClient, err := mgsdk.EnableUser(tc.id, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, eClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, eClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Identify was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}

	cases2 := []struct {
		desc     string
		token    string
		status   string
		metadata sdk.Metadata
		response sdk.UsersPage
		size     uint64
	}{
		{
			desc:   "list enabled clients",
			status: mgclients.EnabledStatus.String(),
			size:   2,
			response: sdk.UsersPage{
				Users: []sdk.User{enabledClient1, endisabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mgclients.DisabledStatus.String(),
			size:   1,
			response: sdk.UsersPage{
				Users: []sdk.User{disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mgclients.AllStatus.String(),
			size:   3,
			response: sdk.UsersPage{
				Users: []sdk.User{enabledClient1, disabledClient1, endisabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := sdk.PageMetadata{
			Total:  100,
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := crepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertClientsPage(tc.response), nil)
		clientsPage, err := mgsdk.Users(pm, validToken)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(clientsPage.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	ts, crepo, _, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enabledClient1 := sdk.User{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mgclients.EnabledStatus.String()}
	disabledClient1 := sdk.User{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mgclients.DisabledStatus.String()}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = mgclients.DisabledStatus.String()
	disenabledClient1.ID = testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		id       string
		token    string
		client   sdk.User
		response sdk.User
		repoErr  error
		err      errors.SDKError
	}{
		{
			desc:     "disable enabled client",
			id:       enabledClient1.ID,
			token:    validToken,
			client:   enabledClient1,
			response: disenabledClient1,
			err:      nil,
			repoErr:  nil,
		},
		{
			desc:     "disable disabled client",
			id:       disabledClient1.ID,
			token:    validToken,
			client:   disabledClient1,
			response: sdk.User{},
			repoErr:  sdk.ErrFailedDisable,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "disable non-existing client",
			id:       wrongID,
			client:   sdk.User{},
			token:    validToken,
			response: sdk.User{},
			repoErr:  sdk.ErrFailedDisable,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := crepo.On("RetrieveByID", mock.Anything, tc.id).Return(convertClient(tc.client), tc.repoErr)
		repoCall3 := crepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.repoErr)
		dClient, err := mgsdk.DisableUser(tc.id, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, dClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, dClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "Identify", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Identify was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}

	cases2 := []struct {
		desc     string
		token    string
		status   string
		metadata sdk.Metadata
		response sdk.UsersPage
		size     uint64
	}{
		{
			desc:   "list enabled clients",
			status: mgclients.EnabledStatus.String(),
			size:   2,
			response: sdk.UsersPage{
				Users: []sdk.User{enabledClient1, disenabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mgclients.DisabledStatus.String(),
			size:   1,
			response: sdk.UsersPage{
				Users: []sdk.User{disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mgclients.AllStatus.String(),
			size:   3,
			response: sdk.UsersPage{
				Users: []sdk.User{enabledClient1, disabledClient1, disenabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := sdk.PageMetadata{
			Total:  100,
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{UserId: validID}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := crepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertClientsPage(tc.response), nil)
		page, err := mgsdk.Users(pm, validToken)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}
