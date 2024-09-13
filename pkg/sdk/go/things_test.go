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

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	authmocks "github.com/absmach/magistrala/pkg/auth/mocks"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	api "github.com/absmach/magistrala/things/api/http"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupThings() (*httptest.Server, *mocks.Service) {
	tsvc := new(mocks.Service)
	gsvc := new(gmocks.Service)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	authClient := new(authmocks.AuthClient)
	api.MakeHandler(tsvc, gsvc, authClient, mux, logger, "")

	return httptest.NewServer(mux), tsvc
}

func TestCreateThing(t *testing.T) {
	ts, tsvc := setupThings()
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
		desc           string
		token          string
		createThingReq sdk.Thing
		svcReq         mgclients.Client
		svcRes         []mgclients.Client
		svcErr         error
		response       sdk.Thing
		err            errors.SDKError
	}{
		{
			desc:           "create new thing successfully",
			token:          validToken,
			createThingReq: createThingReq,
			svcReq:         convertThing(createThingReq),
			svcRes:         []mgclients.Client{convertThing(thing)},
			svcErr:         nil,
			response:       thing,
			err:            nil,
		},
		{
			desc:           "create new thing with invalid token",
			token:          invalidToken,
			createThingReq: createThingReq,
			svcReq:         convertThing(createThingReq),
			svcRes:         []mgclients.Client{},
			svcErr:         svcerr.ErrAuthentication,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:           "create new thing with empty token",
			token:          "",
			createThingReq: createThingReq,
			svcReq:         convertThing(createThingReq),
			svcRes:         []mgclients.Client{},
			svcErr:         nil,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:           "create an existing thing",
			token:          validToken,
			createThingReq: createThingReq,
			svcReq:         convertThing(createThingReq),
			svcRes:         []mgclients.Client{},
			svcErr:         svcerr.ErrCreateEntity,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "create a thing with name too long",
			token: validToken,
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
			desc:  "create a thing with invalid id",
			token: validToken,
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
			desc:  "create a thing with a request that can't be marshalled",
			token: validToken,
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
			svcCall := tsvc.On("CreateThings", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateThing(tc.createThingReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateThings", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestCreateThings(t *testing.T) {
	ts, tsvc := setupThings()
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
		token               string
		createThingsRequest []sdk.Thing
		svcReq              []mgclients.Client
		svcRes              []mgclients.Client
		svcErr              error
		response            []sdk.Thing
		err                 errors.SDKError
	}{
		{
			desc:                "create new things successfully",
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
			token:               invalidToken,
			createThingsRequest: things,
			svcReq:              convertThings(things...),
			svcRes:              []mgclients.Client{},
			svcErr:              svcerr.ErrAuthentication,
			response:            []sdk.Thing{},
			err:                 errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:                "create new things with empty token",
			token:               "",
			createThingsRequest: things,
			svcReq:              convertThings(things...),
			svcRes:              []mgclients.Client{},
			svcErr:              nil,
			response:            []sdk.Thing{},
			err:                 errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:                "create new things with a request that can't be marshalled",
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
			svcCall := tsvc.On("CreateThings", mock.Anything, tc.token, tc.svcReq[0], tc.svcReq[1], tc.svcReq[2]).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateThings(tc.createThingsRequest, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateThings", mock.Anything, tc.token, tc.svcReq[0], tc.svcReq[1], tc.svcReq[2])
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListThings(t *testing.T) {
	ts, tsvc := setupThings()
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
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		svcReq   mgclients.Page
		svcRes   mgclients.ClientsPage
		svcErr   error
		response sdk.ThingsPage
		err      errors.SDKError
	}{
		{
			desc:  "list all things successfully",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list all things with limit greater than max",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  1000,
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
				Offset: 0,
				Limit:  100,
				Name:   strings.Repeat("a", 1025),
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
				Offset: 0,
				Limit:  100,
				Status: mgclients.DisabledStatus.String(),
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
				Offset: 0,
				Limit:  100,
				Tag:    "tag1",
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
			svcCall := tsvc.On("ListClients", mock.Anything, tc.token, mock.Anything, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Things(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClients", mock.Anything, tc.token, mock.Anything, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListThingsByChannel(t *testing.T) {
	ts, tsvc := setupThings()
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
		desc      string
		token     string
		channelID string
		pageMeta  sdk.PageMetadata
		svcReq    mgclients.Page
		svcRes    mgclients.MembersPage
		svcErr    error
		response  sdk.ThingsPage
		err       errors.SDKError
	}{
		{
			desc:      "list things successfully",
			token:     validToken,
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes:   mgclients.MembersPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "list things with empty token",
			token:     "",
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.MembersPage{},
			svcErr:   nil,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:      "list things with status",
			token:     validToken,
			channelID: validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
				Status: mgclients.DisabledStatus.String(),
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
				Offset: 0,
				Limit:  100,
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
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
			svcCall := tsvc.On("ListClientsByGroup", mock.Anything, tc.token, tc.channelID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ThingsByChannel(tc.channelID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClientsByGroup", mock.Anything, tc.token, tc.channelID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestViewThing(t *testing.T) {
	ts, tsvc := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		thingID  string
		svcRes   mgclients.Client
		svcErr   error
		response sdk.Thing
		err      errors.SDKError
	}{
		{
			desc:     "view thing successfully",
			token:    validToken,
			thingID:  thing.ID,
			svcRes:   convertThing(thing),
			svcErr:   nil,
			response: thing,
			err:      nil,
		},
		{
			desc:     "view thing with an invalid token",
			token:    invalidToken,
			thingID:  thing.ID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view thing with empty token",
			token:    "",
			thingID:  thing.ID,
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "view thing with an invalid thing id",
			token:    validToken,
			thingID:  wrongID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view thing with empty thing id",
			token:    validToken,
			thingID:  "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:    "view thing with response that can't be unmarshalled",
			token:   validToken,
			thingID: thing.ID,
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
			svcCall := tsvc.On("ViewClient", mock.Anything, tc.token, tc.thingID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Thing(tc.thingID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewClient", mock.Anything, tc.token, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestViewThingPermissions(t *testing.T) {
	ts, tsvc := setupThings()
	defer ts.Close()

	thing := sdk.Thing{
		Permissions: []string{auth.ViewPermission},
	}
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		thingID  string
		svcRes   []string
		svcErr   error
		response sdk.Thing
		err      errors.SDKError
	}{
		{
			desc:     "view thing permissions successfully",
			token:    validToken,
			thingID:  validID,
			svcRes:   []string{auth.ViewPermission},
			svcErr:   nil,
			response: thing,
			err:      nil,
		},
		{
			desc:     "view thing permissions with an invalid token",
			token:    invalidToken,
			thingID:  validID,
			svcRes:   []string{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "view thing permissions with empty token",
			token:    "",
			thingID:  thing.ID,
			svcRes:   []string{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "view thing permissions with an invalid thing id",
			token:    validToken,
			thingID:  wrongID,
			svcRes:   []string{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view thing permissions with empty thing id",
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
			svcCall := tsvc.On("ViewClientPerms", mock.Anything, tc.token, tc.thingID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ThingPermissions(tc.thingID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewClientPerms", mock.Anything, tc.token, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUpdateThing(t *testing.T) {
	ts, tsvc := setupThings()
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
		desc           string
		token          string
		updateThingReq sdk.Thing
		svcReq         mgclients.Client
		svcRes         mgclients.Client
		svcErr         error
		response       sdk.Thing
		err            errors.SDKError
	}{
		{
			desc:           "update thing successfully",
			token:          validToken,
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         convertThing(updatedThing),
			svcErr:         nil,
			response:       updatedThing,
			err:            nil,
		},
		{
			desc:           "update thing with an invalid token",
			token:          invalidToken,
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         mgclients.Client{},
			svcErr:         svcerr.ErrAuthorization,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:           "update thing with empty token",
			token:          "",
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         mgclients.Client{},
			svcErr:         nil,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "update thing with an invalid thing id",
			token: validToken,
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
			desc:  "update thing with empty thing id",
			token: validToken,
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
			desc:  "update thing with a request that can't be marshalled",
			token: validToken,
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
			svcCall := tsvc.On("UpdateClient", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateThing(tc.updateThingReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClient", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUpdateThingTags(t *testing.T) {
	ts, tsvc := setupThings()
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
		desc           string
		token          string
		updateThingReq sdk.Thing
		svcReq         mgclients.Client
		svcRes         mgclients.Client
		svcErr         error
		response       sdk.Thing
		err            errors.SDKError
	}{
		{
			desc:           "update thing tags successfully",
			token:          validToken,
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         convertThing(updatedThing),
			svcErr:         nil,
			response:       updatedThing,
			err:            nil,
		},
		{
			desc:           "update thing tags with an invalid token",
			token:          invalidToken,
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         mgclients.Client{},
			svcErr:         svcerr.ErrAuthorization,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:           "update thing tags with empty token",
			token:          "",
			updateThingReq: updateThingReq,
			svcReq:         convertThing(updateThingReq),
			svcRes:         mgclients.Client{},
			svcErr:         nil,
			response:       sdk.Thing{},
			err:            errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "update thing tags with an invalid thing id",
			token: validToken,
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
			desc:  "update thing tags with empty thing id",
			token: validToken,
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
			desc:  "update thing tags with a request that can't be marshalled",
			token: validToken,
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
			svcCall := tsvc.On("UpdateClientTags", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateThingTags(tc.updateThingReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClientTags", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUpdateThingSecret(t *testing.T) {
	ts, tsvc := setupThings()
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
		desc      string
		token     string
		thingID   string
		newSecret string
		svcRes    mgclients.Client
		svcErr    error
		response  sdk.Thing
		err       errors.SDKError
	}{
		{
			desc:      "update thing secret successfully",
			token:     validToken,
			thingID:   thing.ID,
			newSecret: newSecret,
			svcRes:    convertThing(updatedThing),
			svcErr:    nil,
			response:  updatedThing,
			err:       nil,
		},
		{
			desc:      "update thing secret with an invalid token",
			token:     invalidToken,
			thingID:   thing.ID,
			newSecret: newSecret,
			svcRes:    mgclients.Client{},
			svcErr:    svcerr.ErrAuthorization,
			response:  sdk.Thing{},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "update thing secret with empty token",
			token:     "",
			thingID:   thing.ID,
			newSecret: newSecret,
			svcRes:    mgclients.Client{},
			svcErr:    nil,
			response:  sdk.Thing{},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:      "update thing secret with an invalid thing id",
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
			svcCall := tsvc.On("UpdateClientSecret", mock.Anything, tc.token, tc.thingID, tc.newSecret).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateThingSecret(tc.thingID, tc.newSecret, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClientSecret", mock.Anything, tc.token, tc.thingID, tc.newSecret)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestEnableThing(t *testing.T) {
	ts, tsvc := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	enabledThing := thing
	enabledThing.Status = mgclients.EnabledStatus.String()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		thingID  string
		svcRes   mgclients.Client
		svcErr   error
		response sdk.Thing
		err      errors.SDKError
	}{
		{
			desc:     "enable thing successfully",
			token:    validToken,
			thingID:  thing.ID,
			svcRes:   convertThing(enabledThing),
			svcErr:   nil,
			response: enabledThing,
			err:      nil,
		},
		{
			desc:     "enable thing with an invalid token",
			token:    invalidToken,
			thingID:  thing.ID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "enable thing with an invalid thing id",
			token:    validToken,
			thingID:  wrongID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrEnableClient,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrEnableClient, http.StatusUnprocessableEntity),
		},
		{
			desc:     "enable thing with empty thing id",
			token:    validToken,
			thingID:  "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:    "enable thing with a response that can't be unmarshalled",
			token:   validToken,
			thingID: thing.ID,
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
			svcCall := tsvc.On("EnableClient", mock.Anything, tc.token, tc.thingID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.EnableThing(tc.thingID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "EnableClient", mock.Anything, tc.token, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDisableThing(t *testing.T) {
	ts, tsvc := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)
	disabledThing := thing
	disabledThing.Status = mgclients.DisabledStatus.String()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		thingID  string
		svcRes   mgclients.Client
		svcErr   error
		response sdk.Thing
		err      errors.SDKError
	}{
		{
			desc:     "disable thing successfully",
			token:    validToken,
			thingID:  thing.ID,
			svcRes:   convertThing(disabledThing),
			svcErr:   nil,
			response: disabledThing,
			err:      nil,
		},
		{
			desc:     "disable thing with an invalid token",
			token:    invalidToken,
			thingID:  thing.ID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "disable thing with an invalid thing id",
			token:    validToken,
			thingID:  wrongID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrDisableClient,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrDisableClient, http.StatusInternalServerError),
		},
		{
			desc:     "disable thing with empty thing id",
			token:    validToken,
			thingID:  "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.Thing{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:    "disable thing with a response that can't be unmarshalled",
			token:   validToken,
			thingID: thing.ID,
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
			svcCall := tsvc.On("DisableClient", mock.Anything, tc.token, tc.thingID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DisableThing(tc.thingID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DisableClient", mock.Anything, tc.token, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestShareThing(t *testing.T) {
	ts, tsvc := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		thingID  string
		shareReq sdk.UsersRelationRequest
		svcErr   error
		err      errors.SDKError
	}{
		{
			desc:    "share thing successfully",
			token:   validToken,
			thingID: thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:    "share thing with an invalid token",
			token:   invalidToken,
			thingID: thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:    "share thing with empty token",
			token:   "",
			thingID: thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "share thing with an invalid thing id",
			token:   validToken,
			thingID: wrongID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: svcerr.ErrUpdateEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:    "share thing with empty thing id",
			token:   validToken,
			thingID: "",
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:    "share thing with empty relation",
			token:   validToken,
			thingID: thing.ID,
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
			svcCall := tsvc.On("Share", mock.Anything, tc.token, tc.thingID, tc.shareReq.Relation, tc.shareReq.UserIDs[0]).Return(tc.svcErr)
			err := mgsdk.ShareThing(tc.thingID, tc.shareReq, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Share", mock.Anything, tc.token, tc.thingID, tc.shareReq.Relation, tc.shareReq.UserIDs[0])
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUnshareThing(t *testing.T) {
	ts, tsvc := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		thingID  string
		shareReq sdk.UsersRelationRequest
		svcErr   error
		err      errors.SDKError
	}{
		{
			desc:    "unshare thing successfully",
			token:   validToken,
			thingID: thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:    "unshare thing with an invalid token",
			token:   invalidToken,
			thingID: thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:    "unshare thing with empty token",
			token:   "",
			thingID: thing.ID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "unshare thing with an invalid thing id",
			token:   validToken,
			thingID: wrongID,
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: svcerr.ErrUpdateEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:    "unshare thing with empty thing id",
			token:   validToken,
			thingID: "",
			shareReq: sdk.UsersRelationRequest{
				UserIDs:  []string{validID},
				Relation: auth.EditorRelation,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := tsvc.On("Unshare", mock.Anything, tc.token, tc.thingID, tc.shareReq.Relation, tc.shareReq.UserIDs[0]).Return(tc.svcErr)
			err := mgsdk.UnshareThing(tc.thingID, tc.shareReq, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Unshare", mock.Anything, tc.token, tc.thingID, tc.shareReq.Relation, tc.shareReq.UserIDs[0])
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDeleteThing(t *testing.T) {
	ts, tsvc := setupThings()
	defer ts.Close()

	thing := generateTestThing(t)

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc    string
		token   string
		thingID string
		svcErr  error
		err     errors.SDKError
	}{
		{
			desc:    "delete thing successfully",
			token:   validToken,
			thingID: thing.ID,
			svcErr:  nil,
			err:     nil,
		},
		{
			desc:    "delete thing with an invalid token",
			token:   invalidToken,
			thingID: thing.ID,
			svcErr:  svcerr.ErrAuthorization,
			err:     errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:    "delete thing with empty token",
			token:   "",
			thingID: thing.ID,
			svcErr:  svcerr.ErrAuthentication,
			err:     errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:    "delete thing with an invalid thing id",
			token:   validToken,
			thingID: wrongID,
			svcErr:  svcerr.ErrRemoveEntity,
			err:     errors.NewSDKErrorWithStatus(svcerr.ErrRemoveEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:    "delete thing with empty thing id",
			token:   validToken,
			thingID: "",
			svcErr:  nil,
			err:     errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := tsvc.On("DeleteClient", mock.Anything, tc.token, tc.thingID).Return(tc.svcErr)
			err := mgsdk.DeleteThing(tc.thingID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DeleteClient", mock.Anything, tc.token, tc.thingID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListUserThings(t *testing.T) {
	ts, tsvc := setupThings()
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
		desc     string
		token    string
		userID   string
		pageMeta sdk.PageMetadata
		svcReq   mgclients.Page
		svcRes   mgclients.ClientsPage
		svcErr   error
		response sdk.ThingsPage
		err      errors.SDKError
	}{
		{
			desc:   "list user things successfully",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
				Role:       mgclients.AllRole,
			},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.ThingsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:   "list user things with limit greater than max",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  1000,
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
				Offset: 0,
				Limit:  100,
				Name:   strings.Repeat("a", 1025),
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
				Offset: 0,
				Limit:  100,
				Status: mgclients.DisabledStatus.String(),
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
				Offset: 0,
				Limit:  100,
				Tag:    "tag1",
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
				Offset: 0,
				Limit:  100,
			},
			svcReq: mgclients.Page{
				Offset:     0,
				Limit:      100,
				Permission: auth.ViewPermission,
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
			svcCall := tsvc.On("ListClients", mock.Anything, tc.token, tc.userID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ListUserThings(tc.userID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClients", mock.Anything, tc.token, tc.userID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
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
		Credentials: sdk.Credentials{
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
