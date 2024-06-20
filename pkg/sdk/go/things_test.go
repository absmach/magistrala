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
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/things"
	api "github.com/absmach/magistrala/things/api/http"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupThings() (*httptest.Server, *mocks.Repository, *gmocks.Repository, *authmocks.AuthClient, *mocks.Cache) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	thingCache := new(mocks.Cache)

	auth := new(authmocks.AuthClient)
	csvc := things.NewService(auth, cRepo, gRepo, thingCache, idProvider)
	gsvc := groups.NewService(gRepo, idProvider, auth)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	api.MakeHandler(csvc, gsvc, mux, logger, "")

	return httptest.NewServer(mux), cRepo, gRepo, auth, thingCache
}

func setupThingsMinimal() (*httptest.Server, *authmocks.AuthClient) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	thingCache := new(mocks.Cache)

	auth := new(authmocks.AuthClient)
	csvc := things.NewService(auth, cRepo, gRepo, thingCache, idProvider)
	gsvc := groups.NewService(gRepo, idProvider, auth)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	api.MakeHandler(csvc, gsvc, mux, logger, "")

	return httptest.NewServer(mux), auth
}

func TestCreateThing(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	thing := sdk.Thing{
		Name:   "test",
		Status: mgclients.EnabledStatus.String(),
	}
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		client   sdk.Thing
		response sdk.Thing
		token    string
		repoErr  error
		err      errors.SDKError
	}{
		{
			desc:     "register new thing",
			client:   thing,
			response: thing,
			token:    token,
			repoErr:  nil,
			err:      nil,
		},
		{
			desc:     "register existing thing",
			client:   thing,
			response: sdk.Thing{},
			token:    token,
			repoErr:  sdk.ErrFailedCreation,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "register empty thing",
			client:   sdk.Thing{},
			response: sdk.Thing{},
			token:    token,
			repoErr:  errors.ErrMalformedEntity,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusBadRequest),
		},
		{
			desc: "register a thing that can't be marshalled",
			client: sdk.Thing{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.Thing{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
			repoErr:  errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
		{
			desc: "register thing with empty secret",
			client: sdk.Thing{
				Name: "emptysecret",
				Credentials: sdk.Credentials{
					Secret: "",
				},
			},
			response: sdk.Thing{
				Name: "emptysecret",
				Credentials: sdk.Credentials{
					Secret: "",
				},
			},
			token:   token,
			err:     nil,
			repoErr: nil,
		},
		{
			desc: "register thing with empty identity",
			client: sdk.Thing{
				Credentials: sdk.Credentials{
					Identity: "",
					Secret:   secret,
				},
			},
			response: sdk.Thing{
				Credentials: sdk.Credentials{
					Identity: "",
					Secret:   secret,
				},
			},
			token:   token,
			repoErr: nil,
			err:     nil,
		},
		{
			desc: "register thing with every field defined",
			client: sdk.Thing{
				ID:          id,
				Name:        "name",
				Tags:        []string{"tag1", "tag2"},
				Credentials: user.Credentials,
				Metadata:    validMetadata,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      mgclients.EnabledStatus.String(),
			},
			response: sdk.Thing{
				ID:          id,
				Name:        "name",
				Tags:        []string{"tag1", "tag2"},
				Credentials: user.Credentials,
				Metadata:    validMetadata,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      mgclients.EnabledStatus.String(),
			},
			token:   token,
			repoErr: nil,
			err:     nil,
		},
	}
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		authCall1 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(&magistrala.AddPoliciesRes{Added: true}, nil)
		authCall2 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		authCall3 := auth.On("DeletePolicies", mock.Anything, mock.Anything).Return(&magistrala.DeletePoliciesRes{Deleted: false}, nil)
		repoCall3 := cRepo.On("Save", mock.Anything, mock.Anything).Return(convertThings(tc.response), tc.repoErr)
		rThing, err := mgsdk.CreateThing(tc.client, tc.token)

		tc.response.ID = rThing.ID
		tc.response.CreatedAt = rThing.CreatedAt
		tc.response.UpdatedAt = rThing.UpdatedAt
		rThing.Credentials.Secret = tc.response.Credentials.Secret
		rThing.Status = tc.response.Status
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rThing))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		authCall2.Unset()
		authCall3.Unset()
		repoCall3.Unset()
	}
}

func TestCreateThings(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	thingsList := []sdk.Thing{
		{
			Name:   "test",
			Status: mgclients.EnabledStatus.String(),
		},
		{
			Name:   "test2",
			Status: mgclients.EnabledStatus.String(),
		},
	}
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		things   []sdk.Thing
		response []sdk.Thing
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "register new things",
			things:   thingsList,
			response: thingsList,
			token:    token,
			err:      nil,
		},
		{
			desc:     "register existing things",
			things:   thingsList,
			response: []sdk.Thing{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "register empty things",
			things:   []sdk.Thing{},
			response: []sdk.Thing{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
		},
		{
			desc: "register things that can't be marshalled",
			things: []sdk.Thing{
				{
					Name: "test",
					Metadata: map[string]interface{}{
						"test": make(chan int),
					},
				},
			},
			response: []sdk.Thing{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		authCall1 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(&magistrala.AddPoliciesRes{Added: true}, nil)
		authCall2 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		authCall3 := auth.On("DeletePolicies", mock.Anything, mock.Anything).Return(&magistrala.DeletePoliciesRes{Deleted: false}, nil)
		repoCall1 := cRepo.On("Save", mock.Anything, mock.Anything).Return(convertThings(tc.response...), tc.err)
		if len(tc.things) > 0 {
			repoCall1 = cRepo.On("Save", mock.Anything, mock.Anything, mock.Anything).Return(convertThings(tc.response...), tc.err)
		}
		rThing, err := mgsdk.CreateThings(tc.things, tc.token)
		for i, t := range rThing {
			tc.response[i].ID = t.ID
			tc.response[i].CreatedAt = t.CreatedAt
			tc.response[i].UpdatedAt = t.UpdatedAt
			tc.response[i].Credentials.Secret = t.Credentials.Secret
			t.Status = tc.response[i].Status
		}
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rThing))
		if tc.err == nil {
			switch len(tc.things) {
			case 1:
				ok := repoCall1.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
				assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
			case 2:
				ok := repoCall1.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything, mock.Anything)
				assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
			}
		}
		authCall.Unset()
		authCall1.Unset()
		authCall2.Unset()
		authCall3.Unset()
		repoCall1.Unset()
	}
}

func TestListThings(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	var ths []sdk.Thing
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		th := sdk.Thing{
			ID:   generateUUID(t),
			Name: fmt.Sprintf("thing_%d", i),
			Credentials: sdk.Credentials{
				Identity: fmt.Sprintf("identity_%d", i),
				Secret:   generateUUID(t),
			},
			Metadata: sdk.Metadata{"name": fmt.Sprintf("thing_%d", i)},
			Status:   mgclients.EnabledStatus.String(),
		}
		if i == 50 {
			th.Status = mgclients.DisabledStatus.String()
			th.Tags = []string{"tag1", "tag2"}
		}
		ths = append(ths, th)
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
		response   []sdk.Thing
	}{
		{
			desc:     "get a list of things",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			err:      nil,
			response: ths[offset:limit],
		},
		{
			desc:     "get a list of things with invalid token",
			token:    authmocks.InvalidValue,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of things with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      nil,
			response: []sdk.Thing{},
		},
		{
			desc:     "get a list of things with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
			response: []sdk.Thing(nil),
		},
		{
			desc:       "get a list of things with same identity",
			token:      token,
			offset:     0,
			limit:      1,
			err:        nil,
			identifier: Identity,
			metadata:   sdk.Metadata{},
			response:   []sdk.Thing{ths[89]},
		},
		{
			desc:       "get a list of things with same identity and metadata",
			token:      token,
			offset:     0,
			limit:      1,
			err:        nil,
			identifier: Identity,
			metadata: sdk.Metadata{
				"name": "client99",
			},
			response: []sdk.Thing{ths[89]},
		},
		{
			desc:   "list things with given metadata",
			token:  validToken,
			offset: 0,
			limit:  1,
			metadata: sdk.Metadata{
				"name": "client99",
			},
			response: []sdk.Thing{ths[89]},
			err:      nil,
		},
		{
			desc:     "list things with given name",
			token:    validToken,
			offset:   0,
			limit:    1,
			name:     "client10",
			response: []sdk.Thing{ths[0]},
			err:      nil,
		},
		{
			desc:     "list things with given status",
			token:    validToken,
			offset:   0,
			limit:    1,
			status:   mgclients.DisabledStatus.String(),
			response: []sdk.Thing{ths[50]},
			err:      nil,
		},
		{
			desc:     "list things with given tag",
			token:    validToken,
			offset:   0,
			limit:    1,
			tag:      "tag1",
			response: []sdk.Thing{ths[50]},
			err:      nil,
		},
	}

	for _, tc := range cases {
		pm := sdk.PageMetadata{
			Status:   tc.status,
			Total:    total,
			Offset:   tc.offset,
			Limit:    tc.limit,
			Name:     tc.name,
			Metadata: tc.metadata,
			Tag:      tc.tag,
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
			repoCall2 = auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{}, svcerr.ErrAuthorization)
		}
		repoCall3 := cRepo.On("RetrieveAllByIDs", mock.Anything, mock.Anything).Return(mgclients.ClientsPage{Page: convertClientPage(pm), Clients: convertThings(tc.response...)}, tc.err)
		page, err := mgsdk.Things(pm, validToken)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall2.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		repoCall3.Unset()
	}
}

func TestListUserThings(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	var ths []sdk.Thing
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		th := sdk.Thing{
			ID:   generateUUID(t),
			Name: fmt.Sprintf("thing_%d", i),
			Credentials: sdk.Credentials{
				Identity: fmt.Sprintf("identity_%d", i),
				Secret:   generateUUID(t),
			},
			Metadata: sdk.Metadata{"name": fmt.Sprintf("thing_%d", i)},
			Status:   mgclients.EnabledStatus.String(),
		}
		if i == 50 {
			th.Status = mgclients.DisabledStatus.String()
			th.Tags = []string{"tag1", "tag2"}
		}
		ths = append(ths, th)
	}

	cases := []struct {
		desc       string
		token      string
		status     string
		total      uint64
		offset     uint64
		limit      uint64
		name       string
		userID     string
		identifier string
		tag        string
		metadata   sdk.Metadata
		err        errors.SDKError
		response   []sdk.Thing
	}{
		{
			desc:     "get a list of user things",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			userID:   validID,
			err:      nil,
			response: ths[offset:limit],
		},
		{
			desc:     "get a list of user things with invalid token",
			token:    authmocks.InvalidValue,
			offset:   offset,
			limit:    limit,
			userID:   validID,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of user things with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			userID:   validID,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of user things with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			userID:   validID,
			err:      nil,
			response: []sdk.Thing{},
		},
		{
			desc:     "get a list of user things with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			userID:   validID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
			response: []sdk.Thing(nil),
		},
		{
			desc:       "get a list of user things with same identity",
			token:      token,
			offset:     0,
			limit:      1,
			err:        nil,
			identifier: Identity,
			userID:     validID,
			metadata:   sdk.Metadata{},
			response:   []sdk.Thing{ths[89]},
		},
		{
			desc:       "get a list of user things with same identity and metadata",
			token:      token,
			offset:     0,
			limit:      1,
			err:        nil,
			userID:     validID,
			identifier: Identity,
			metadata: sdk.Metadata{
				"name": "client99",
			},
			response: []sdk.Thing{ths[89]},
		},
		{
			desc:   "list user things with given metadata",
			token:  validToken,
			offset: 0,
			limit:  1,
			metadata: sdk.Metadata{
				"name": "client99",
			},
			userID:   validID,
			response: []sdk.Thing{ths[89]},
			err:      nil,
		},
		{
			desc:     "list user things with given name",
			token:    validToken,
			offset:   0,
			limit:    1,
			name:     "client10",
			userID:   validID,
			response: []sdk.Thing{ths[0]},
			err:      nil,
		},
		{
			desc:     "list user things with given status",
			token:    validToken,
			offset:   0,
			limit:    1,
			userID:   validID,
			status:   mgclients.DisabledStatus.String(),
			response: []sdk.Thing{ths[50]},
			err:      nil,
		},
		{
			desc:     "list user things with given tag",
			token:    validToken,
			offset:   0,
			limit:    1,
			tag:      "tag1",
			response: []sdk.Thing{ths[50]},
			userID:   validID,
			err:      nil,
		},
	}

	for _, tc := range cases {
		pm := sdk.PageMetadata{
			Status:   tc.status,
			Total:    total,
			Offset:   tc.offset,
			Limit:    tc.limit,
			Name:     tc.name,
			Metadata: tc.metadata,
			Tag:      tc.tag,
			User:     tc.userID,
		}
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Identify", mock.Anything, mock.Anything).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
			repoCall2 = auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{}, svcerr.ErrAuthorization)
		}
		repoCall3 := cRepo.On("RetrieveAllByIDs", mock.Anything, mock.Anything).Return(mgclients.ClientsPage{Page: convertClientPage(pm), Clients: convertThings(tc.response...)}, tc.err)
		page, err := mgsdk.ListUserThings(pm, validToken)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Things, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall2.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		repoCall3.Unset()
	}
}

func TestListThingsByChannel(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	nThing := uint64(10)
	aThings := []sdk.Thing{}

	for i := uint64(1); i < nThing; i++ {
		thing := sdk.Thing{
			Name: fmt.Sprintf("member_%d@example.com", i),
			Credentials: sdk.Credentials{
				Secret: generateUUID(t),
			},
			Tags:     []string{"tag1", "tag2"},
			Metadata: sdk.Metadata{"role": "client"},
			Status:   mgclients.EnabledStatus.String(),
		}
		aThings = append(aThings, thing)
	}

	cases := []struct {
		desc     string
		token    string
		page     sdk.PageMetadata
		response []sdk.Thing
		err      errors.SDKError
	}{
		{
			desc:  "list things with authorized token",
			token: validToken,
			page: sdk.PageMetadata{
				Channel: testsutil.GenerateUUID(t),
			},
			response: aThings,
			err:      nil,
		},
		{
			desc:  "list things with offset and limit",
			token: validToken,
			page: sdk.PageMetadata{
				Offset:  4,
				Limit:   nThing,
				Channel: testsutil.GenerateUUID(t),
			},
			response: aThings[4:],
			err:      nil,
		},
		{
			desc:  "list things with given name",
			token: validToken,
			page: sdk.PageMetadata{
				Name:    Identity,
				Offset:  6,
				Limit:   nThing,
				Channel: testsutil.GenerateUUID(t),
			},
			response: aThings[6:],
			err:      nil,
		},
		{
			desc:  "list things with given subject",
			token: validToken,
			page: sdk.PageMetadata{
				Subject: subject,
				Offset:  6,
				Limit:   nThing,
				Channel: testsutil.GenerateUUID(t),
			},
			response: aThings[6:],
			err:      nil,
		},
		{
			desc:  "list things with given object",
			token: validToken,
			page: sdk.PageMetadata{
				Object:  object,
				Offset:  6,
				Limit:   nThing,
				Channel: testsutil.GenerateUUID(t),
			},
			response: aThings[6:],
			err:      nil,
		},
		{
			desc:  "list things with an invalid token",
			token: invalidToken,
			page: sdk.PageMetadata{
				Channel: testsutil.GenerateUUID(t),
			},
			response: []sdk.Thing(nil),
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list things with an invalid id",
			token: validToken,
			page: sdk.PageMetadata{
				Channel: wrongID,
			},
			response: []sdk.Thing(nil),
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", mock.Anything, mock.Anything).Return(mgclients.ClientsPage{Page: convertClientPage(tc.page), Clients: convertThings(tc.response...)}, tc.err)
		membersPage, err := mgsdk.ThingsByChannel(tc.page, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, membersPage.Things, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, membersPage.Things))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveAllByIDs", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Members was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestThing(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	thing := sdk.Thing{
		Name:        "thingname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: generateUUID(t)},
		Metadata:    validMetadata,
		Status:      mgclients.EnabledStatus.String(),
	}
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		thingID  string
		response sdk.Thing
		err      errors.SDKError
	}{
		{
			desc:     "view thing successfully",
			response: thing,
			token:    validToken,
			thingID:  generateUUID(t),
			err:      nil,
		},
		{
			desc:     "view thing with an invalid token",
			response: sdk.Thing{},
			token:    invalidToken,
			thingID:  generateUUID(t),
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view thing with valid token and invalid thing id",
			response: sdk.Thing{},
			token:    validToken,
			thingID:  wrongID,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view thing with an invalid token and invalid thing id",
			response: sdk.Thing{},
			token:    invalidToken,
			thingID:  wrongID,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token != validToken {
			repoCall = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall1 := cRepo.On("RetrieveByID", mock.Anything, tc.thingID).Return(convertThing(tc.response), tc.err)
		rClient, err := mgsdk.Thing(tc.thingID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.thingID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestUpdateThing(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	thing := sdk.Thing{
		ID:          generateUUID(t),
		Name:        "clientname",
		Credentials: sdk.Credentials{Secret: generateUUID(t)},
		Metadata:    validMetadata,
		Status:      mgclients.EnabledStatus.String(),
	}

	thing1 := thing
	thing1.Name = "Updated client"

	thing2 := thing
	thing2.Metadata = sdk.Metadata{"role": "test"}
	thing2.ID = invalidIdentity

	cases := []struct {
		desc     string
		thing    sdk.Thing
		response sdk.Thing
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "update thing name with valid token",
			thing:    thing1,
			response: thing1,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update thing name with invalid token",
			thing:    thing1,
			response: sdk.Thing{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "update thing name with invalid id",
			thing:    thing2,
			response: sdk.Thing{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "update thing that can't be marshalled",
			thing: sdk.Thing{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.Thing{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token != validToken {
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := cRepo.On("Update", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.err)
		uClient, err := mgsdk.UpdateThing(tc.thing, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall2.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateThingTags(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	thing := sdk.Thing{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Secret: generateUUID(t)},
		Status:      mgclients.EnabledStatus.String(),
	}

	thing1 := thing
	thing1.Tags = []string{"updatedTag1", "updatedTag2"}

	thing2 := thing
	thing2.ID = invalidIdentity

	cases := []struct {
		desc     string
		thing    sdk.Thing
		response sdk.Thing
		token    string
		err      error
	}{
		{
			desc:     "update thing name with valid token",
			thing:    thing,
			response: thing1,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update thing name with invalid token",
			thing:    thing1,
			response: sdk.Thing{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "update thing name with invalid id",
			thing:    thing2,
			response: sdk.Thing{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "update thing that can't be marshalled",
			thing: sdk.Thing{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.Thing{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token != validToken {
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := cRepo.On("UpdateTags", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.err)
		uClient, err := mgsdk.UpdateThingTags(tc.thing, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "UpdateTags", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
		}
		repoCall2.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateThingSecret(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	user.ID = generateUUID(t)
	rthing := thing
	rthing.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)

	cases := []struct {
		desc      string
		oldSecret string
		newSecret string
		token     string
		response  sdk.Thing
		repoErr   error
		err       error
	}{
		{
			desc:      "update thing secret with valid token",
			oldSecret: thing.Credentials.Secret,
			newSecret: "newSecret",
			token:     validToken,
			response:  rthing,
			repoErr:   nil,
			err:       nil,
		},
		{
			desc:      "update thing secret with invalid token",
			oldSecret: thing.Credentials.Secret,
			newSecret: "newPassword",
			token:     "non-existent",
			response:  sdk.Thing{},
			repoErr:   svcerr.ErrAuthorization,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusForbidden),
		},
		{
			desc:      "update thing secret with wrong old secret",
			oldSecret: "oldSecret",
			newSecret: "newSecret",
			token:     validToken,
			response:  sdk.Thing{},
			repoErr:   apiutil.ErrMissingSecret,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token != validToken {
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := cRepo.On("UpdateSecret", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.repoErr)
		uClient, err := mgsdk.UpdateThingSecret(tc.oldSecret, tc.newSecret, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "UpdateSecret", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		repoCall2.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestEnableThing(t *testing.T) {
	ts, cRepo, _, auth, _ := setupThings()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enabledThing1 := sdk.Thing{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client1@example.com", Secret: generateUUID(t)}, Status: mgclients.EnabledStatus.String()}
	disabledThing1 := sdk.Thing{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client3@example.com", Secret: generateUUID(t)}, Status: mgclients.DisabledStatus.String()}
	endisabledThing1 := disabledThing1
	endisabledThing1.Status = mgclients.EnabledStatus.String()
	endisabledThing1.ID = testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		id       string
		token    string
		thing    sdk.Thing
		response sdk.Thing
		repoErr  error
		err      errors.SDKError
	}{
		{
			desc:     "enable disabled thing",
			id:       disabledThing1.ID,
			token:    validToken,
			thing:    disabledThing1,
			response: endisabledThing1,
			repoErr:  nil,
			err:      nil,
		},
		{
			desc:     "enable enabled thing",
			id:       enabledThing1.ID,
			token:    validToken,
			thing:    enabledThing1,
			response: sdk.Thing{},
			repoErr:  sdk.ErrFailedEnable,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrEnableClient, http.StatusBadRequest),
		},
		{
			desc:     "enable non-existing thing",
			id:       wrongID,
			token:    validToken,
			thing:    sdk.Thing{},
			response: sdk.Thing{},
			repoErr:  sdk.ErrFailedEnable,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrEnableClient, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token != validToken {
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := cRepo.On("RetrieveByID", mock.Anything, tc.id).Return(convertThing(tc.thing), tc.repoErr)
		repoCall3 := cRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.repoErr)
		eClient, err := mgsdk.EnableThing(tc.id, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, eClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, eClient))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.id)
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
		response sdk.ThingsPage
		size     uint64
	}{
		{
			desc:   "list enabled clients",
			status: mgclients.EnabledStatus.String(),
			size:   2,
			response: sdk.ThingsPage{
				Things: []sdk.Thing{enabledThing1, endisabledThing1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mgclients.DisabledStatus.String(),
			size:   1,
			response: sdk.ThingsPage{
				Things: []sdk.Thing{disabledThing1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mgclients.AllStatus.String(),
			size:   3,
			response: sdk.ThingsPage{
				Things: []sdk.Thing{enabledThing1, disabledThing1, endisabledThing1},
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token != validToken {
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response.Things)}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", mock.Anything, mock.Anything).Return(convertThingsPage(tc.response), nil)
		clientsPage, err := mgsdk.Things(pm, validToken)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(clientsPage.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestDisableThing(t *testing.T) {
	ts, cRepo, _, auth, cache := setupThings()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enabledThing1 := sdk.Thing{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client1@example.com", Secret: generateUUID(t)}, Status: mgclients.EnabledStatus.String()}
	disabledThing1 := sdk.Thing{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client3@example.com", Secret: generateUUID(t)}, Status: mgclients.DisabledStatus.String()}
	disenabledThing1 := enabledThing1
	disenabledThing1.Status = mgclients.DisabledStatus.String()
	disenabledThing1.ID = testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		id       string
		token    string
		thing    sdk.Thing
		response sdk.Thing
		repoErr  error
		err      errors.SDKError
	}{
		{
			desc:     "disable enabled thing",
			id:       enabledThing1.ID,
			token:    validToken,
			thing:    enabledThing1,
			response: disenabledThing1,
			repoErr:  nil,
			err:      nil,
		},
		{
			desc:     "disable disabled thing",
			id:       disabledThing1.ID,
			token:    validToken,
			thing:    disabledThing1,
			response: sdk.Thing{},
			repoErr:  sdk.ErrFailedDisable,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrDisableClient, http.StatusBadRequest),
		},
		{
			desc:     "disable non-existing thing",
			id:       wrongID,
			thing:    sdk.Thing{},
			token:    validToken,
			response: sdk.Thing{},
			repoErr:  sdk.ErrFailedDisable,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrDisableClient, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token != validToken {
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := cRepo.On("RetrieveByID", mock.Anything, tc.id).Return(convertThing(tc.thing), tc.repoErr)
		repoCall3 := cRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(convertThing(tc.response), tc.repoErr)
		repoCall4 := cache.On("Remove", mock.Anything, mock.Anything).Return(nil)
		dThing, err := mgsdk.DisableThing(tc.id, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, dThing, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, dThing))
		if tc.err == nil {
			ok := repoCall2.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}

	cases2 := []struct {
		desc     string
		token    string
		status   string
		metadata sdk.Metadata
		response sdk.ThingsPage
		size     uint64
	}{
		{
			desc:   "list enabled things",
			status: mgclients.EnabledStatus.String(),
			size:   2,
			response: sdk.ThingsPage{
				Things: []sdk.Thing{enabledThing1, disenabledThing1},
			},
		},
		{
			desc:   "list disabled things",
			status: mgclients.DisabledStatus.String(),
			size:   1,
			response: sdk.ThingsPage{
				Things: []sdk.Thing{disabledThing1},
			},
		},
		{
			desc:   "list enabled and disabled things",
			status: mgclients.AllStatus.String(),
			size:   3,
			response: sdk.ThingsPage{
				Things: []sdk.Thing{enabledThing1, disabledThing1, disenabledThing1},
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token != validToken {
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response.Things)}, nil)
		repoCall3 := cRepo.On("RetrieveAllByIDs", mock.Anything, mock.Anything).Return(convertThingsPage(tc.response), nil)
		page, err := mgsdk.Things(pm, validToken)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestShareThing(t *testing.T) {
	ts, auth := setupThingsMinimal()
	auth.Test(t)
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc      string
		channelID string
		thingID   string
		token     string
		err       errors.SDKError
		repoErr   error
	}{
		{
			desc:      "share thing with valid token",
			channelID: generateUUID(t),
			thingID:   "thingID",
			token:     validToken,
			err:       nil,
		},
		{
			desc:      "share thing with invalid token",
			channelID: generateUUID(t),
			thingID:   "thingID",
			token:     invalidToken,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "share thing with valid token for unauthorized user",
			channelID: generateUUID(t),
			thingID:   "thingID",
			token:     validToken,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
			repoErr:   svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, tc.repoErr)
		repoCall2 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(&magistrala.AddPoliciesRes{Added: true}, nil)
		if tc.token != validToken {
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall3 := auth.On("AddPolicy", mock.Anything, mock.Anything).Return(&magistrala.AddPolicyRes{Added: true}, nil)
		req := sdk.UsersRelationRequest{
			Relation: "contributor",
			UserIDs:  []string{tc.channelID},
		}
		err := mgsdk.ShareThing(tc.thingID, req, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestDeleteThing(t *testing.T) {
	ts, cRepo, _, auth, cache := setupThings()

	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	thing := sdk.Thing{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Secret: generateUUID(t)},
		Status:      mgclients.EnabledStatus.String(),
	}

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
	repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, nil)
	repoCall2 := cRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
	repoCall4 := cache.On("Remove", mock.Anything, thing.ID).Return(nil)
	err := mgsdk.DeleteThing("wrongID", validToken)
	assert.Equal(t, err, errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden), fmt.Sprintf("Delete thing with wrong id: expected %v got %v", svcerr.ErrNotFound, err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	repoCall = auth.On("DeletePolicy", mock.Anything, mock.Anything, mock.Anything).Return(&magistrala.DeletePolicyRes{Deleted: true}, nil)
	repoCall1 = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
	repoCall2 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall3 := cRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
	err = mgsdk.DeleteThing(thing.ID, validToken)
	assert.Nil(t, err, fmt.Sprintf("Delete thing with correct id: expected %v got %v", nil, err))
	ok := repoCall3.Parent.AssertCalled(t, "Delete", mock.Anything, thing.ID)
	assert.True(t, ok, "Delete was not called on deleting thing with correct id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
	repoCall3.Unset()
	repoCall4.Unset()
}
