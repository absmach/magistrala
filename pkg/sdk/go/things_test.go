// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	policies "github.com/absmach/magistrala/pkg/policies"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	api "github.com/absmach/magistrala/things/api/http"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupThings() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	tsvc := new(mocks.Service)
	gsvc := new(gmocks.Service)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	authn := new(authnmocks.Authentication)
	api.MakeHandler(tsvc, gsvc, authn, mux, logger, "")

	return httptest.NewServer(mux), tsvc, authn
}

func TestCreateThing(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	createThingReq := sdk.Thing{
		Name:        thing.Name,
		Tags:        thing.Tags,
		Credentials: thing.Credentials,
		Metadata:    thing.Metadata,
		Status:      thing.Status,
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}

	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		createThingReq  sdk.Thing
		svcReq          mgclients.Client
		svcRes          []mgclients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Thing
		err             errors.SDKError
	}{
		{
			desc:           "create new thing successfully",
			domainID:       domainID,
			token:          validToken,
			createThingReq: createThingReq,
			svcReq:         convertThing(createThingReq),
			svcRes:         []mgclients.Client{convertThing(thing)},
			svcErr:         nil,
			response:       thing,
			err:            nil,
		},
		{
			desc:            "create new thing with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			createThingReq:  createThingReq,
			svcReq:          convertThing(createThingReq),
			svcRes:          []mgclients.Client{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Thing{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:           "create new thing with empty token",
			domainID:       domainID,
			token:          "",
			createThingReq: createThingReq,
			svcReq:         convertThing(createThingReq),
			svcRes:         []mgclients.Client{},
			svcErr:         nil,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:           "create an existing thing",
			domainID:       domainID,
			token:          validToken,
			createThingReq: createThingReq,
			svcReq:         convertThing(createThingReq),
			svcRes:         []mgclients.Client{},
			svcErr:         svcerr.ErrCreateEntity,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "create a thing with name too long",
			domainID: domainID,
			token:    validToken,
			createThingReq: sdk.Thing{
				Name:        strings.Repeat("a", 1025),
				Tags:        thing.Tags,
				Credentials: thing.Credentials,
				Metadata:    thing.Metadata,
				Status:      thing.Status,
			},
			svcReq:   mgclients.Client{},
			svcRes:   []mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:     "create a thing with invalid id",
			domainID: domainID,
			token:    validToken,
			createThingReq: sdk.Thing{
				ID:          "123456789",
				Name:        thing.Name,
				Tags:        thing.Tags,
				Credentials: thing.Credentials,
				Metadata:    thing.Metadata,
				Status:      thing.Status,
			},
			svcReq:   mgclients.Client{},
			svcRes:   []mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidIDFormat), http.StatusBadRequest),
		},
		{
			desc:     "create a thing with a request that can't be marshalled",
			domainID: domainID,
			token:    validToken,
			createThingReq: sdk.Thing{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   mgclients.Client{},
			svcRes:   []mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:           "create a thing with a response that can't be unmarshalled",
			domainID:       domainID,
			token:          validToken,
			createThingReq: createThingReq,
			svcReq:         convertThing(createThingReq),
			svcRes: []mgclients.Client{{
				Name:        thing.Name,
				Tags:        thing.Tags,
				Credentials: mgclients.Credentials(thing.Credentials),
				Metadata: mgclients.Metadata{
					"test": make(chan int),
				},
			}},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("CreateThings", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateThing(tc.createThingReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateThings", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestCreateThings(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	things := []sdk.Thing{}
	for i := 0; i < 3; i++ {
		thing := generateTestThing(t)
		things = append(things, thing)
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc                string
		domainID            string
		token               string
		session             mgauthn.Session
		createThingsRequest []sdk.Thing
		svcReq              []mgclients.Client
		svcRes              []mgclients.Client
		svcErr              error
		authenticateErr     error
		response            []sdk.Thing
		err                 errors.SDKError
	}{
		{
			desc:                "create new things successfully",
			domainID:            domainID,
			token:               validToken,
			createThingsRequest: things,
			svcReq:              convertThings(things...),
			svcRes:              convertThings(things...),
			svcErr:              nil,
			response:            things,
			err:                 nil,
		},
		{
			desc:                "create new things with invalid token",
			domainID:            domainID,
			token:               invalidToken,
			createThingsRequest: things,
			svcReq:              convertThings(things...),
			svcRes:              []mgclients.Client{},
			authenticateErr:     svcerr.ErrAuthentication,
			response:            []sdk.Thing{},
			err:                 errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:                "create new things with empty token",
			domainID:            domainID,
			token:               "",
			createThingsRequest: things,
			svcReq:              convertThings(things...),
			svcRes:              []mgclients.Client{},
			svcErr:              nil,
			response:            []sdk.Thing{},
			err:                 errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:                "create new things with a request that can't be marshalled",
			domainID:            domainID,
			token:               validToken,
			createThingsRequest: []sdk.Thing{{Name: "test", Metadata: map[string]interface{}{"test": make(chan int)}}},
			svcReq:              convertThings(things...),
			svcRes:              []mgclients.Client{},
			svcErr:              nil,
			response:            []sdk.Thing{},
			err:                 errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:                "create new things with a response that can't be unmarshalled",
			domainID:            domainID,
			token:               validToken,
			createThingsRequest: things,
			svcReq:              convertThings(things...),
			svcRes: []mgclients.Client{{
				Name:        things[0].Name,
				Tags:        things[0].Tags,
				Credentials: mgclients.Credentials(things[0].Credentials),
				Metadata: mgclients.Metadata{
					"test": make(chan int),
				},
			}},
			svcErr:   nil,
			response: []sdk.Thing{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("CreateThings", mock.Anything, tc.session, tc.svcReq[0], tc.svcReq[1], tc.svcReq[2]).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateThings(tc.createThingsRequest, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateThings", mock.Anything, tc.session, tc.svcReq[0], tc.svcReq[1], tc.svcReq[2])
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListThings(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	var things []sdk.Thing
	for i := 10; i < 100; i++ {
		thing := generateTestThing(t)
		if i == 50 {
			thing.Status = mgclients.DisabledStatus.String()
			thing.Tags = []string{"tag1", "tag2"}
		}
		things = append(things, thing)
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		pageMeta        sdk.PageMetadata
		svcReq          mgclients.Page
		svcRes          mgclients.ClientsPage
		svcErr          error
		authenticateErr error
		response        sdk.ThingsPage
		err             errors.SDKError
	}{
		{
			desc:  "list all things successfully",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  uint64(len(things)),
				},
				Clients: convertThings(things...),
			},
			svcErr: nil,
			response: sdk.ThingsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: uint64(len(things)),
				},
				Things: things,
			},
		},
		{
			desc:  "list all things with an invalid token",
			token: invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes:          mgclients.ClientsPage{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.ThingsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list all things with limit greater than max",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    1000,
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:  "list all things with name size greater than max",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				Name:     strings.Repeat("a", 1025),
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:  "list all things with status",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				Status:   mgclients.DisabledStatus.String(),
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
				Status:     mgclients.DisabledStatus,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: convertThings(things[50]),
			},
			svcErr: nil,
			response: sdk.ThingsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: 1,
				},
				Things: []sdk.Thing{things[50]},
			},
			err: nil,
		},
		{
			desc:  "list all things with tags",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				Tag:      "tag1",
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
				Tag:        "tag1",
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: convertThings(things[50]),
			},
			svcErr: nil,
			response: sdk.ThingsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: 1,
				},
				Things: []sdk.Thing{things[50]},
			},
			err: nil,
		},
		{
			desc:  "list all things with invalid metadata",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "list all things with response that can't be unmarshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: []mgclients.Client{{
					Name:        things[0].Name,
					Tags:        things[0].Tags,
					Credentials: mgclients.Credentials(things[0].Credentials),
					Metadata: mgclients.Metadata{
						"test": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("ListClients", mock.Anything, tc.session, mock.Anything, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Things(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClients", mock.Anything, tc.session, mock.Anything, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListThingsByChannel(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	var things []sdk.Thing
	for i := 10; i < 100; i++ {
		thing := generateTestThing(t)
		if i == 50 {
			thing.Status = mgclients.DisabledStatus.String()
		}
		things = append(things, thing)
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		channelID       string
		pageMeta        sdk.PageMetadata
		svcReq          mgclients.Page
		svcRes          mgclients.MembersPage
		svcErr          error
		authenticateErr error
		response        sdk.ThingsPage
		err             errors.SDKError
	}{
		{
			desc:      "list things successfully",
			token:     validToken,
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes: mgclients.MembersPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  uint64(len(things)),
				},
				Members: convertThings(things...),
			},
			svcErr: nil,
			response: sdk.ThingsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: uint64(len(things)),
				},
				Things: things,
			},
		},
		{
			desc:      "list things with an invalid token",
			token:     invalidToken,
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes:          mgclients.MembersPage{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.ThingsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "list things with empty token",
			token:     "",
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.MembersPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "list things with status",
			token:     validToken,
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				Status:   mgclients.DisabledStatus.String(),
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
				Status:     mgclients.DisabledStatus,
			},
			svcRes: mgclients.MembersPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Members: convertThings(things[50]),
			},
			svcErr: nil,
			response: sdk.ThingsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: 1,
				},
				Things: []sdk.Thing{things[50]},
			},
			err: nil,
		},
		{
			desc:      "list things with empty channel id",
			token:     validToken,
			channelID: "",
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.MembersPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "list things with invalid metadata",
			token:     validToken,
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.MembersPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:      "list things with response that can't be unmarshalled",
			token:     validToken,
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes: mgclients.MembersPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Members: []mgclients.Client{{
					Name:        things[0].Name,
					Tags:        things[0].Tags,
					Credentials: mgclients.Credentials(things[0].Credentials),
					Metadata: mgclients.Metadata{
						"test": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("ListClientsByGroup", mock.Anything, tc.session, tc.channelID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ThingsByChannel(tc.channelID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClientsByGroup", mock.Anything, tc.session, tc.channelID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewThing(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		thingID         string
		svcRes          mgclients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Thing
		err             errors.SDKError
	}{
		{
			desc:     "view thing successfully",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			svcRes:   convertThing(thing),
			svcErr:   nil,
			response: thing,
			err:      nil,
		},
		{
			desc:            "view thing with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			thingID:         thing.ID,
			svcRes:          mgclients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Thing{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view thing with empty token",
			domainID: domainID,
			token:    "",
			thingID:  thing.ID,
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view thing with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  wrongID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view thing with empty thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:     "view thing with response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			svcRes: mgclients.Client{
				Name:        thing.Name,
				Tags:        thing.Tags,
				Credentials: mgclients.Credentials(thing.Credentials),
				Metadata: mgclients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("ViewClient", mock.Anything, tc.session, tc.thingID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Thing(tc.thingID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewClient", mock.Anything, tc.session, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewThingPermissions(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := sdk.Thing{
		Permissions: []string{policies.ViewPermission},
	}
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		thingID         string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        sdk.Thing
		err             errors.SDKError
	}{
		{
			desc:     "view thing permissions successfully",
			domainID: domainID,
			token:    validToken,
			thingID:  validID,
			svcRes:   []string{policies.ViewPermission},
			svcErr:   nil,
			response: thing,
			err:      nil,
		},
		{
			desc:            "view thing permissions with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			thingID:         validID,
			svcRes:          []string{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Thing{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view thing permissions with empty token",
			domainID: domainID,
			token:    "",
			thingID:  thing.ID,
			svcRes:   []string{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view thing permissions with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  wrongID,
			svcRes:   []string{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view thing permissions with empty thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  "",
			svcRes:   []string{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("ViewClientPerms", mock.Anything, tc.session, tc.thingID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ThingPermissions(tc.thingID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewClientPerms", mock.Anything, tc.session, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateThing(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	updatedThing := thing
	updatedThing.Name = "newName"
	updatedThing.Metadata = map[string]interface{}{
		"newKey": "newValue",
	}
	updateThingReq := sdk.Thing{
		ID:       thing.ID,
		Name:     updatedThing.Name,
		Metadata: updatedThing.Metadata,
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		updateThingReq  sdk.Thing
		svcReq          mgclients.Client
		svcRes          mgclients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Thing
		err             errors.SDKError
	}{
		{
			desc:           "update thing successfully",
			domainID:       domainID,
			token:          validToken,
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         convertThing(updatedThing),
			svcErr:         nil,
			response:       updatedThing,
			err:            nil,
		},
		{
			desc:            "update thing with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			updateThingReq:  updateThingReq,
			svcReq:          convertThing(updateThingReq),
			svcRes:          mgclients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Thing{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:           "update thing with empty token",
			domainID:       domainID,
			token:          "",
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         mgclients.Client{},
			svcErr:         nil,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update thing with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			updateThingReq: sdk.Thing{
				ID:   wrongID,
				Name: updatedThing.Name,
			},
			svcReq: convertThing(sdk.Thing{
				ID:   wrongID,
				Name: updatedThing.Name,
			}),
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "update thing with empty thing id",
			domainID: domainID,
			token:    validToken,

			updateThingReq: sdk.Thing{
				ID:   "",
				Name: updatedThing.Name,
			},
			svcReq: convertThing(sdk.Thing{
				ID:   "",
				Name: updatedThing.Name,
			}),
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:     "update thing with a request that can't be marshalled",
			domainID: domainID,
			token:    validToken,

			updateThingReq: sdk.Thing{
				ID: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:           "update thing with a response that can't be unmarshalled",
			domainID:       domainID,
			token:          validToken,
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes: mgclients.Client{
				Name:        updatedThing.Name,
				Tags:        updatedThing.Tags,
				Credentials: mgclients.Credentials(updatedThing.Credentials),
				Metadata: mgclients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("UpdateClient", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateThing(tc.updateThingReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClient", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateThingTags(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	updatedThing := thing
	updatedThing.Tags = []string{"newTag1", "newTag2"}
	updateThingReq := sdk.Thing{
		ID:   thing.ID,
		Tags: updatedThing.Tags,
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		updateThingReq  sdk.Thing
		svcReq          mgclients.Client
		svcRes          mgclients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Thing
		err             errors.SDKError
	}{
		{
			desc:           "update thing tags successfully",
			domainID:       domainID,
			token:          validToken,
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         convertThing(updatedThing),
			svcErr:         nil,
			response:       updatedThing,
			err:            nil,
		},
		{
			desc:            "update thing tags with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			updateThingReq:  updateThingReq,
			svcReq:          convertThing(updateThingReq),
			svcRes:          mgclients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Thing{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:           "update thing tags with empty token",
			domainID:       domainID,
			token:          "",
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         mgclients.Client{},
			svcErr:         nil,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update thing tags with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			updateThingReq: sdk.Thing{
				ID:   wrongID,
				Tags: updatedThing.Tags,
			},
			svcReq: convertThing(sdk.Thing{
				ID:   wrongID,
				Tags: updatedThing.Tags,
			}),
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "update thing tags with empty thing id",
			domainID: domainID,
			token:    validToken,
			updateThingReq: sdk.Thing{
				ID:   "",
				Tags: updatedThing.Tags,
			},
			svcReq: convertThing(sdk.Thing{
				ID:   "",
				Tags: updatedThing.Tags,
			}),
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "update thing tags with a request that can't be marshalled",
			domainID: domainID,
			token:    validToken,
			updateThingReq: sdk.Thing{
				ID: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:           "update thing tags with a response that can't be unmarshalled",
			domainID:       domainID,
			token:          validToken,
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes: mgclients.Client{
				Name:        updatedThing.Name,
				Tags:        updatedThing.Tags,
				Credentials: mgclients.Credentials(updatedThing.Credentials),
				Metadata: mgclients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("UpdateClientTags", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateThingTags(tc.updateThingReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClientTags", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateThingSecret(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	newSecret := generateUUID(t)
	updatedThing := thing
	updatedThing.Credentials.Secret = newSecret

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		thingID         string
		newSecret       string
		svcRes          mgclients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Thing
		err             errors.SDKError
	}{
		{
			desc:      "update thing secret successfully",
			domainID:  domainID,
			token:     validToken,
			thingID:   thing.ID,
			newSecret: newSecret,
			svcRes:    convertThing(updatedThing),
			svcErr:    nil,
			response:  updatedThing,
			err:       nil,
		},
		{
			desc:            "update thing secret with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			thingID:         thing.ID,
			newSecret:       newSecret,
			svcRes:          mgclients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Thing{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "update thing secret with empty token",
			domainID:  domainID,
			token:     "",
			thingID:   thing.ID,
			newSecret: newSecret,
			svcRes:    mgclients.Client{},
			svcErr:    nil,
			response:  sdk.Thing{},
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "update thing secret with an invalid thing id",
			domainID:  domainID,
			token:     validToken,
			thingID:   wrongID,
			newSecret: newSecret,
			svcRes:    mgclients.Client{},
			svcErr:    svcerr.ErrUpdateEntity,
			response:  sdk.Thing{},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:      "update thing secret with empty thing id",
			domainID:  domainID,
			token:     validToken,
			thingID:   "",
			newSecret: newSecret,
			svcRes:    mgclients.Client{},
			svcErr:    nil,
			response:  sdk.Thing{},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "update thing with empty new secret",
			domainID:  domainID,
			token:     validToken,
			thingID:   thing.ID,
			newSecret: "",
			svcRes:    mgclients.Client{},
			svcErr:    nil,
			response:  sdk.Thing{},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingSecret), http.StatusBadRequest),
		},
		{
			desc:      "update thing secret with a response that can't be unmarshalled",
			domainID:  domainID,
			token:     validToken,
			thingID:   thing.ID,
			newSecret: newSecret,
			svcRes: mgclients.Client{
				Name:        updatedThing.Name,
				Tags:        updatedThing.Tags,
				Credentials: mgclients.Credentials(updatedThing.Credentials),
				Metadata: mgclients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("UpdateClientSecret", mock.Anything, tc.session, tc.thingID, tc.newSecret).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateThingSecret(tc.thingID, tc.newSecret, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClientSecret", mock.Anything, tc.session, tc.thingID, tc.newSecret)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableThing(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	enabledThing := thing
	enabledThing.Status = mgclients.EnabledStatus.String()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		thingID         string
		svcRes          mgclients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Thing
		err             errors.SDKError
	}{
		{
			desc:     "enable thing successfully",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			svcRes:   convertThing(enabledThing),
			svcErr:   nil,
			response: enabledThing,
			err:      nil,
		},
		{
			desc:            "enable thing with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			thingID:         thing.ID,
			svcRes:          mgclients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Thing{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "enable thing with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  wrongID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrEnableClient,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrEnableClient, http.StatusUnprocessableEntity),
		},
		{
			desc:     "enable thing with empty thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "enable thing with a response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			svcRes: mgclients.Client{
				Name:        enabledThing.Name,
				Tags:        enabledThing.Tags,
				Credentials: mgclients.Credentials(enabledThing.Credentials),
				Metadata: mgclients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("EnableClient", mock.Anything, tc.session, tc.thingID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.EnableThing(tc.thingID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "EnableClient", mock.Anything, tc.session, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableThing(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	disabledThing := thing
	disabledThing.Status = mgclients.DisabledStatus.String()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		thingID         string
		svcRes          mgclients.Client
		svcErr          error
		authenticateErr error
		response        sdk.Thing
		err             errors.SDKError
	}{
		{
			desc:     "disable thing successfully",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			svcRes:   convertThing(disabledThing),
			svcErr:   nil,
			response: disabledThing,
			err:      nil,
		},
		{
			desc:            "disable thing with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			thingID:         thing.ID,
			svcRes:          mgclients.Client{},
			authenticateErr: svcerr.ErrAuthorization,
			response:        sdk.Thing{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "disable thing with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  wrongID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrDisableClient,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrDisableClient, http.StatusInternalServerError),
		},
		{
			desc:     "disable thing with empty thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "disable thing with a response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			svcRes: mgclients.Client{
				Name:        disabledThing.Name,
				Tags:        disabledThing.Tags,
				Credentials: mgclients.Credentials(disabledThing.Credentials),
				Metadata: mgclients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("DisableClient", mock.Anything, tc.session, tc.thingID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DisableThing(tc.thingID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DisableClient", mock.Anything, tc.session, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestShareThing(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		thingID         string
		shareReq        sdk.UsersRelationRequest
		authenticateErr error
		svcErr          error
		err             errors.SDKError
	}{
		{
			desc:     "share thing successfully",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "share thing with an invalid token",
			domainID: domainID,
			token:    invalidToken,
			thingID:  thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			authenticateErr: svcerr.ErrAuthorization,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "share thing with empty token",
			domainID: domainID,
			token:    "",
			thingID:  thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "share thing with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  wrongID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			svcErr: svcerr.ErrUpdateEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "share thing with empty thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  "",
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "share thing with empty relation",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: "",
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicy), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("Share", mock.Anything, tc.session, tc.thingID, tc.shareReq.Relation, tc.shareReq.UserIDs[0]).Return(tc.svcErr)
			err := mgsdk.ShareThing(tc.thingID, tc.shareReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Share", mock.Anything, tc.session, tc.thingID, tc.shareReq.Relation, tc.shareReq.UserIDs[0])
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUnshareThing(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		thingID         string
		shareReq        sdk.UsersRelationRequest
		authenticateErr error
		svcErr          error
		err             errors.SDKError
	}{
		{
			desc:     "unshare thing successfully",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "unshare thing with an invalid token",
			domainID: domainID,
			token:    invalidToken,
			thingID:  thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			authenticateErr: svcerr.ErrAuthorization,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "unshare thing with empty token",
			domainID: domainID,
			token:    "",
			thingID:  thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			err: errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "unshare thing with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  wrongID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			svcErr: svcerr.ErrUpdateEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "unshare thing with empty thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  "",
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: policies.EditorRelation,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("Unshare", mock.Anything, tc.session, tc.thingID, tc.shareReq.Relation, tc.shareReq.UserIDs[0]).Return(tc.svcErr)
			err := mgsdk.UnshareThing(tc.thingID, tc.shareReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Unshare", mock.Anything, tc.session, tc.thingID, tc.shareReq.Relation, tc.shareReq.UserIDs[0])
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteThing(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		thingID         string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "delete thing successfully",
			domainID: domainID,
			token:    validToken,
			thingID:  thing.ID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:            "delete thing with an invalid token",
			domainID:        domainID,
			token:           invalidToken,
			thingID:         thing.ID,
			authenticateErr: svcerr.ErrAuthorization,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "delete thing with empty token",
			domainID: domainID,
			token:    "",
			thingID:  thing.ID,
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "delete thing with an invalid thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  wrongID,
			svcErr:   svcerr.ErrRemoveEntity,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrRemoveEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "delete thing with empty thing id",
			domainID: domainID,
			token:    validToken,
			thingID:  "",
			svcErr:   nil,
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("DeleteClient", mock.Anything, tc.session, tc.thingID).Return(tc.svcErr)
			err := mgsdk.DeleteThing(tc.thingID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DeleteClient", mock.Anything, tc.session, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListUserThings(t *testing.T) {
	ts, tsvc, auth := setupThings()
	defer ts.Close()

	var things []sdk.Thing
	for i := 10; i < 100; i++ {
		thing := generateTestThing(t)
		if i == 50 {
			thing.Status = mgclients.DisabledStatus.String()
			thing.Tags = []string{"tag1", "tag2"}
		}
		things = append(things, thing)
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		userID          string
		pageMeta        sdk.PageMetadata
		svcReq          mgclients.Page
		svcRes          mgclients.ClientsPage
		svcErr          error
		authenticateErr error
		response        sdk.ThingsPage
		err             errors.SDKError
	}{
		{
			desc:   "list user things successfully",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  uint64(len(things)),
				},
				Clients: convertThings(things...),
			},
			svcErr: nil,
			response: sdk.ThingsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: uint64(len(things)),
				},
				Things: things,
			},
		},
		{
			desc:   "list user things with an invalid token",
			token:  invalidToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes:          mgclients.ClientsPage{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.ThingsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:   "list user things with limit greater than max",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    1000,
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:   "list user things with name size greater than max",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				Name:     strings.Repeat("a", 1025),
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:   "list user things with status",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				Status:   mgclients.DisabledStatus.String(),
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
				Status:     mgclients.DisabledStatus,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: convertThings(things[50]),
			},
			svcErr: nil,
			response: sdk.ThingsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: 1,
				},
				Things: []sdk.Thing{things[50]},
			},
			err: nil,
		},
		{
			desc:   "list user things with tags",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				Tag:      "tag1",
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
				Tag:        "tag1",
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: convertThings(things[50]),
			},
			svcErr: nil,
			response: sdk.ThingsPage{
				PageRes: sdk.PageRes{
					Limit: 100,
					Total: 1,
				},
				Things: []sdk.Thing{things[50]},
			},
			err: nil,
		},
		{
			desc:   "list user things with invalid metadata",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
				DomainID: domainID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "list user things with response that can't be unmarshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    100,
				DomainID: domainID,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: policies.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 0,
					Limit:  100,
					Total:  1,
				},
				Clients: []mgclients.Client{{
					Name:        things[0].Name,
					Tags:        things[0].Tags,
					Credentials: mgclients.Credentials(things[0].Credentials),
					Metadata: mgclients.Metadata{
						"test": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, mock.Anything).Return(tc.session, tc.authenticateErr)
			svcCall := tsvc.On("ListClients", mock.Anything, tc.session, tc.userID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ListUserThings(tc.userID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClients", mock.Anything, tc.session, tc.userID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func generateTestThing(t *testing.T) sdk.Thing {
	createdAt, err := time.Parse(time.RFC3339, "2023-03-03T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	updatedAt := createdAt
	return sdk.Thing{
		ID:   testsutil.GenerateUUID(t),
		Name: "clientname",
		Credentials: sdk.ClientCredentials{
			Identity: "thing@example.com",
			Secret:   generateUUID(t),
		},
		Tags:      []string{"tag1", "tag2"},
		Metadata:  validMetadata,
		Status:    mgclients.EnabledStatus.String(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
