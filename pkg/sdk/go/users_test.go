// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/absmach/magistrala/auth"
	internalapi "github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	oauth2mocks "github.com/absmach/magistrala/pkg/oauth2/mocks"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/users/api"
	umocks "github.com/absmach/magistrala/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	id         = generateUUID(&testing.T{})
	validToken = "token"
	validID    = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	wrongID    = testsutil.GenerateUUID(&testing.T{})
)

func setupUsers() (*httptest.Server, *umocks.Service) {
	usvc := new(umocks.Service)
	gsvc := new(gmocks.Service)
	logger := mglog.NewMock()
	mux := chi.NewRouter()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	api.MakeHandler(usvc, gsvc, mux, logger, "", passRegex, provider)

	return httptest.NewServer(mux), usvc
}

func TestCreateUser(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	createSdkUserReq := sdk.User{
		Name:        user.Name,
		Tags:        user.Tags,
		Credentials: user.Credentials,
		Metadata:    user.Metadata,
		Status:      user.Status,
	}

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc             string
		token            string
		createSdkUserReq sdk.User
		svcReq           mgclients.Client
		svcRes           mgclients.Client
		svcErr           error
		response         sdk.User
		err              errors.SDKError
	}{
		{
			desc:             "register new user successfully",
			token:            validToken,
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertClient(createSdkUserReq),
			svcRes:           convertClient(user),
			svcErr:           nil,
			response:         user,
			err:              nil,
		},
		{
			desc:             "register existing user",
			token:            validToken,
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertClient(createSdkUserReq),
			svcRes:           mgclients.Client{},
			svcErr:           svcerr.ErrCreateEntity,
			response:         sdk.User{},
			err:              errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:             "register user with invalid token",
			token:            invalidToken,
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertClient(createSdkUserReq),
			svcRes:           mgclients.Client{},
			svcErr:           svcerr.ErrAuthentication,
			response:         sdk.User{},
			err:              errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:             "register user with empty token",
			token:            "",
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertClient(createSdkUserReq),
			svcRes:           mgclients.Client{},
			svcErr:           svcerr.ErrAuthentication,
			response:         sdk.User{},
			err:              errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:             "register empty user",
			token:            validToken,
			createSdkUserReq: sdk.User{},
			svcReq:           mgclients.Client{},
			svcRes:           mgclients.Client{},
			svcErr:           nil,
			response:         sdk.User{},
			err:              errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusBadRequest),
		},
		{
			desc:  "register user with name too long",
			token: validToken,
			createSdkUserReq: sdk.User{
				Name:        strings.Repeat("a", 1025),
				Credentials: createSdkUserReq.Credentials,
				Metadata:    createSdkUserReq.Metadata,
				Tags:        createSdkUserReq.Tags,
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:  "register user with empty identity",
			token: validToken,
			createSdkUserReq: sdk.User{
				Name: createSdkUserReq.Name,
				Credentials: sdk.Credentials{
					Identity: "",
					Secret:   createSdkUserReq.Credentials.Secret,
				},
				Metadata: createSdkUserReq.Metadata,
				Tags:     createSdkUserReq.Tags,
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingIdentity), http.StatusBadRequest),
		},
		{
			desc:  "register user with empty secret",
			token: validToken,
			createSdkUserReq: sdk.User{
				Name: createSdkUserReq.Name,
				Credentials: sdk.Credentials{
					Identity: createSdkUserReq.Credentials.Identity,
					Secret:   "",
				},
				Metadata: createSdkUserReq.Metadata,
				Tags:     createSdkUserReq.Tags,
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
		{
			desc:  "register user with secret that is too short",
			token: validToken,
			createSdkUserReq: sdk.User{
				Name: createSdkUserReq.Name,
				Credentials: sdk.Credentials{
					Identity: createSdkUserReq.Credentials.Identity,
					Secret:   "weak",
				},
				Metadata: createSdkUserReq.Metadata,
				Tags:     createSdkUserReq.Tags,
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrPasswordFormat), http.StatusBadRequest),
		},
		{
			desc:  "register a user with request that can't be marshalled",
			token: validToken,
			createSdkUserReq: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:             "register a user with response that can't be unmarshalled",
			token:            validToken,
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertClient(createSdkUserReq),
			svcRes: mgclients.Client{
				ID:   id,
				Name: createSdkUserReq.Name,
				Credentials: mgclients.Credentials{
					Identity: createSdkUserReq.Credentials.Identity,
					Secret:   createSdkUserReq.Credentials.Secret,
				},
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RegisterClient", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateUser(tc.createSdkUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "RegisterClient", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListUsers(t *testing.T) {
	ts, svc := setupUsers()
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
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		svcReq   mgclients.Page
		svcRes   mgclients.ClientsPage
		svcErr   error
		response sdk.UsersPage
		err      errors.SDKError
	}{
		{
			desc:  "list users successfully",
			token: token,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(cls[offset:limit])),
				},
				Clients: convertClients(cls[offset:limit]),
			},
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(cls[offset:limit])),
				},
				Users: cls[offset:limit],
			},
			err: nil,
		},
		{
			desc:  "list users with invalid token",
			token: invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				Permission: auth.ViewPermission,
			},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list users with empty token",
			token: "",
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "list users with zero limit",
			token: token,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  0,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      10,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(cls[offset:10])),
				},
				Clients: convertClients(cls[offset:10]),
			},
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(cls[offset:10])),
				},
				Users: cls[offset:10],
			},
			err: nil,
		},
		{
			desc:  "list users with limit greater than max",
			token: token,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  101,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:  "list users with given metadata",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   offset,
				Limit:    limit,
				Metadata: sdk.Metadata{"name": "client_99"},
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Metadata:   mgclients.Metadata{"name": "client_99"},
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{convertClient(cls[89])},
			},
			svcErr: nil,
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Users: []sdk.User{cls[89]},
			},
			err: nil,
		},
		{
			desc:  "list users with given status",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Status: mgclients.DisabledStatus.String(),
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Status:     mgclients.DisabledStatus,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{convertClient(cls[50])},
			},
			svcErr: nil,
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Users: []sdk.User{cls[50]},
			},
			err: nil,
		},
		{
			desc:  "list users with given tag",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Tag:    "tag1",
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Tag:        "tag1",
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{convertClient(cls[50])},
			},
			svcErr: nil,
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Users: []sdk.User{cls[50]},
			},
			err: nil,
		},
		{
			desc:  "list users with request that can't be marshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Metadata: sdk.Metadata{
					"test": make(chan int),
				},
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				Permission: auth.ViewPermission,
			},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "list users with response that can't be unmarshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(cls[offset:limit])),
				},
				Clients: []mgclients.Client{
					{
						ID:   id,
						Name: "client_99",
						Metadata: mgclients.Metadata{
							"key": make(chan int),
						},
					},
				},
			},
			response: sdk.UsersPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListClients", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Users(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClients", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListChannelUsers(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	var cls []sdk.User
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		cl := sdk.User{
			ID:     generateUUID(t),
			Name:   fmt.Sprintf("client_%d", i),
			Status: mgclients.EnabledStatus.String(),
		}
		cls = append(cls, cl)
	}

	cases := []struct {
		desc     string
		token    string
		pageMeta sdk.PageMetadata
		svcReq   mgclients.Page
		svcRes   mgclients.ClientsPage
		svcErr   error
		response sdk.UsersPage
		err      errors.SDKError
	}{
		{
			desc:  "list users successfully",
			token: token,
			pageMeta: sdk.PageMetadata{
				Offset:  offset,
				Limit:   limit,
				Channel: validID,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				EntityType: auth.GroupType,
				EntityID:   validID,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(cls[offset:limit])),
				},
				Clients: convertClients(cls[offset:limit]),
			},
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(cls[offset:limit])),
				},
				Users: cls[offset:limit],
			},
			err: nil,
		},
		{
			desc:  "list users with invalid token",
			token: invalidToken,
			pageMeta: sdk.PageMetadata{
				Offset:  offset,
				Limit:   limit,
				Channel: validID,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				EntityType: auth.GroupType,
				EntityID:   validID,
				Permission: auth.ViewPermission,
			},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list users with empty token",
			token: "",
			pageMeta: sdk.PageMetadata{
				Offset:  offset,
				Limit:   limit,
				Channel: validID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "list users with zero limit",
			token: token,
			pageMeta: sdk.PageMetadata{
				Offset:  offset,
				Limit:   0,
				Channel: validID,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      10,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				EntityType: auth.GroupType,
				EntityID:   validID,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(cls[offset:10])),
				},
				Clients: convertClients(cls[offset:10]),
			},
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(cls[offset:10])),
				},
				Users: cls[offset:10],
			},
			err: nil,
		},
		{
			desc:  "list users with limit greater than max",
			token: token,
			pageMeta: sdk.PageMetadata{
				Offset:  offset,
				Limit:   101,
				Channel: validID,
			},
			svcReq:   mgclients.Page{},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:  "list users with given metadata",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:   offset,
				Limit:    limit,
				Metadata: sdk.Metadata{"name": "client_99"},
				Channel:  validID,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Metadata:   mgclients.Metadata{"name": "client_99"},
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				EntityType: auth.GroupType,
				EntityID:   validID,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{convertClient(cls[89])},
			},
			svcErr: nil,
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Users: []sdk.User{cls[89]},
			},
			err: nil,
		},
		{
			desc:  "list users with given status",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:  offset,
				Limit:   limit,
				Status:  mgclients.DisabledStatus.String(),
				Channel: validID,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Status:     mgclients.DisabledStatus,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				EntityType: auth.GroupType,
				EntityID:   validID,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{convertClient(cls[50])},
			},
			svcErr: nil,
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Users: []sdk.User{cls[50]},
			},
			err: nil,
		},
		{
			desc:  "list users with given tag",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:  offset,
				Limit:   limit,
				Tag:     "tag1",
				Channel: validID,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Tag:        "tag1",
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				EntityType: auth.GroupType,
				EntityID:   validID,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{convertClient(cls[50])},
			},
			svcErr: nil,
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Users: []sdk.User{cls[50]},
			},
			err: nil,
		},
		{
			desc:  "list users with request that can't be marshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Metadata: sdk.Metadata{
					"test": make(chan int),
				},
				Channel: validID,
			},
			svcReq: mgclients.Page{
				Offset: offset,
				Limit:  limit,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes:   mgclients.ClientsPage{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "list users with response that can't be unmarshalled",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset:  offset,
				Limit:   limit,
				Channel: validID,
			},
			svcReq: mgclients.Page{
				Offset:     offset,
				Limit:      limit,
				Order:      internalapi.DefOrder,
				Dir:        internalapi.DefDir,
				EntityType: auth.GroupType,
				EntityID:   validID,
				Permission: auth.ViewPermission,
			},
			svcRes: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: uint64(len(cls[offset:limit])),
				},
				Clients: []mgclients.Client{
					{
						ID:   id,
						Name: "client_99",
						Metadata: mgclients.Metadata{
							"key": make(chan int),
						},
					},
				},
			},
			response: sdk.UsersPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListClients", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Users(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListClients", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestSearchClients(t *testing.T) {
	ts, svc := setupUsers()
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
		desc         string
		token        string
		page         sdk.PageMetadata
		response     []sdk.User
		searchreturn mgclients.ClientsPage
		err          errors.SDKError
		identifyErr  error
	}{
		{
			desc:  "search for users",
			token: validToken,
			err:   nil,
			page: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Name:   "client_10",
			},
			response: []sdk.User{cls[10]},
			searchreturn: mgclients.ClientsPage{
				Clients: []mgclients.Client{convertClient(cls[10])},
				Page: mgclients.Page{
					Total:  1,
					Offset: offset,
					Limit:  limit,
				},
			},
		},
		{
			desc:  "search for users with invalid token",
			token: invalidToken,
			page: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Name:   "client_10",
			},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response:    nil,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:  "search for users with empty token",
			token: "",
			page: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Name:   "client_10",
			},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			response:    nil,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:  "search for users with empty query",
			token: validToken,
			page: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Name:   "",
			},
			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptySearchQuery), http.StatusBadRequest),
		},
		{
			desc:  "search for users with invalid length of query",
			token: validToken,
			page: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
				Name:   "a",
			},
			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrLenSearchQuery, apiutil.ErrValidation), http.StatusBadRequest),
		},
		{
			desc:  "search for users with invalid limit",
			token: validToken,
			page: sdk.PageMetadata{
				Offset: offset,
				Limit:  0,
				Name:   "client_10",
			},
			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := svc.On("SearchUsers", mock.Anything, mock.Anything, mock.Anything).Return(tc.searchreturn, tc.err)
		page, err := mgsdk.SearchUsers(tc.page, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Users, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
	}
}

func TestViewUser(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		userID   string
		svcRes   mgclients.Client
		svcErr   error
		response sdk.User
		err      errors.SDKError
	}{
		{
			desc:     "view user successfully",
			token:    validToken,
			userID:   user.ID,
			svcRes:   convertClient(user),
			svcErr:   nil,
			response: user,
			err:      nil,
		},
		{
			desc:     "view user with invalid token",
			token:    invalidToken,
			userID:   user.ID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view user with empty token",
			token:    "",
			userID:   user.ID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "view user with invalid id",
			token:    validToken,
			userID:   wrongID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view user with empty id",
			token:    validToken,
			userID:   "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:   "view user with response that can't be unmarshalled",
			token:  validToken,
			userID: user.ID,
			svcRes: mgclients.Client{
				ID:   id,
				Name: user.Name,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewClient", mock.Anything, tc.token, tc.userID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.User(tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewClient", mock.Anything, tc.token, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUserProfile(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		svcRes   mgclients.Client
		svcErr   error
		response sdk.User
		err      errors.SDKError
	}{
		{
			desc:     "view user profile successfully",
			token:    validToken,
			svcRes:   convertClient(user),
			svcErr:   nil,
			response: user,
			err:      nil,
		},
		{
			desc:     "view user profile with invalid token",
			token:    invalidToken,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view user profile with empty token",
			token:    "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "view user profile with response that can't be unmarshalled",
			token: validToken,
			svcRes: mgclients.Client{
				ID:   id,
				Name: user.Name,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewProfile", mock.Anything, tc.token).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UserProfile(tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewProfile", mock.Anything, tc.token)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUpdateUser(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedName := "updatedName"
	updatedUser := user
	updatedUser.Name = updatedName

	cases := []struct {
		desc            string
		token           string
		updateClientReq sdk.User
		svcReq          mgclients.Client
		svcRes          mgclients.Client
		svcErr          error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update client name with valid token",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Name: updatedName,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Name: updatedName,
			},
			svcRes:   convertClient(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update client name with invalid token",
			token: invalidToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Name: updatedName,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Name: updatedName,
			},
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update client name with invalid id",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   wrongID,
				Name: updatedName,
			},
			svcReq: mgclients.Client{
				ID:   wrongID,
				Name: updatedName,
			},
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update client name with empty token",
			token: "",
			updateClientReq: sdk.User{
				ID:   user.ID,
				Name: updatedName,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Name: updatedName,
			},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "update client name with empty id",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   "",
				Name: updatedName,
			},
			svcReq: mgclients.Client{
				ID:   "",
				Name: updatedName,
			},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:  "update client with request that can't be marshalled",
			token: validToken,
			updateClientReq: sdk.User{
				ID: generateUUID(t),
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "update client with response that can't be unmarshalled",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Name: updatedName,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Name: updatedName,
			},
			svcRes: mgclients.Client{
				ID:   id,
				Name: updatedName,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateClient", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUser(tc.updateClientReq, tc.token)
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

func TestUpdateUserTags(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedTags := []string{"updatedTag1", "updatedTag2"}

	updatedUser := user
	updatedUser.Tags = updatedTags

	cases := []struct {
		desc            string
		token           string
		updateClientReq sdk.User
		svcReq          mgclients.Client
		svcRes          mgclients.Client
		svcErr          error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update client tags with valid token",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcRes:   convertClient(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update client tags with invalid token",
			token: invalidToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update client tags with empty token",
			token: "",
			updateClientReq: sdk.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "update client tags with invalid id",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   wrongID,
				Tags: updatedTags,
			},
			svcReq: mgclients.Client{
				ID:   wrongID,
				Tags: updatedTags,
			},
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update client tags with empty id",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   "",
				Tags: updatedTags,
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "update client tags with request that can't be marshalled",
			token: validToken,
			updateClientReq: sdk.User{
				ID: generateUUID(t),
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "update client tags with response that can't be unmarshalled",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcRes: mgclients.Client{
				ID:   id,
				Tags: updatedTags,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateClientTags", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUserTags(tc.updateClientReq, tc.token)
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

func TestUpdateUserIdentity(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedIdentity := "updatedIdentity@email.com"
	updatedUser := user
	updatedUser.Credentials.Identity = updatedIdentity

	cases := []struct {
		desc            string
		token           string
		updateClientReq sdk.User
		svcReq          string
		svcRes          mgclients.Client
		svcErr          error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update client identity with valid token",
			token: validToken,
			updateClientReq: sdk.User{
				ID: user.ID,
				Credentials: sdk.Credentials{
					Identity: updatedIdentity,
					Secret:   user.Credentials.Secret,
				},
			},
			svcReq:   updatedIdentity,
			svcRes:   convertClient(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update client identity with invalid token",
			token: invalidToken,
			updateClientReq: sdk.User{
				ID: user.ID,
				Credentials: sdk.Credentials{
					Identity: updatedIdentity,
					Secret:   user.Credentials.Secret,
				},
			},
			svcReq:   updatedIdentity,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update client identity with empty token",
			token: "",
			updateClientReq: sdk.User{
				ID: user.ID,
				Credentials: sdk.Credentials{
					Identity: updatedIdentity,
					Secret:   user.Credentials.Secret,
				},
			},
			svcReq:   updatedIdentity,
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "update client identity with invalid id",
			token: validToken,
			updateClientReq: sdk.User{
				ID: wrongID,
				Credentials: sdk.Credentials{
					Identity: updatedIdentity,
					Secret:   user.Credentials.Secret,
				},
			},
			svcReq:   updatedIdentity,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update client identity with empty id",
			token: validToken,
			updateClientReq: sdk.User{
				ID: "",
				Credentials: sdk.Credentials{
					Identity: updatedIdentity,
					Secret:   user.Credentials.Secret,
				},
			},
			svcReq:   updatedIdentity,
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "update client identity with response that can't be unmarshalled",
			token: validToken,
			updateClientReq: sdk.User{
				ID: user.ID,
				Credentials: sdk.Credentials{
					Identity: updatedIdentity,
					Secret:   user.Credentials.Secret,
				},
			},
			svcReq: updatedIdentity,
			svcRes: mgclients.Client{
				ID:   id,
				Name: user.Name,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateClientIdentity", mock.Anything, tc.token, tc.updateClientReq.ID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUserIdentity(tc.updateClientReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClientIdentity", mock.Anything, tc.token, tc.updateClientReq.ID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestResetPasswordRequest(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	defHost := "http://localhost"

	conf := sdk.Config{
		UsersURL: ts.URL,
		HostURL:  defHost,
	}
	mgsdk := sdk.NewSDK(conf)

	validEmail := "test@email.com"

	cases := []struct {
		desc   string
		email  string
		svcErr error
		err    errors.SDKError
	}{
		{
			desc:   "reset password request with valid email",
			email:  validEmail,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "reset password request with invalid email",
			email:  "invalidemail",
			svcErr: svcerr.ErrViewEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:   "reset password request with empty email",
			email:  "",
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingEmail), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("GenerateResetToken", mock.Anything, tc.email, defHost).Return(tc.svcErr)
			err := mgsdk.ResetPasswordRequest(tc.email)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "GenerateResetToken", mock.Anything, tc.email, defHost)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestResetPassword(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	newPassword := "newPassword"

	cases := []struct {
		desc         string
		token        string
		newPassword  string
		confPassword string
		svcErr       error
		err          errors.SDKError
	}{
		{
			desc:         "reset password successfully",
			token:        validToken,
			newPassword:  newPassword,
			confPassword: newPassword,
			svcErr:       nil,
			err:          nil,
		},
		{
			desc:         "reset password with invalid token",
			token:        invalidToken,
			newPassword:  newPassword,
			confPassword: newPassword,
			svcErr:       svcerr.ErrAuthentication,
			err:          errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:         "reset password with empty token",
			token:        "",
			newPassword:  newPassword,
			confPassword: newPassword,
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:         "reset password with empty new password",
			token:        validToken,
			newPassword:  "",
			confPassword: newPassword,
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
		{
			desc:         "reset password with empty confirm password",
			token:        validToken,
			newPassword:  newPassword,
			confPassword: "",
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingConfPass), http.StatusBadRequest),
		},
		{
			desc:         "reset password with new password not matching confirm password",
			token:        validToken,
			newPassword:  newPassword,
			confPassword: "wrongPassword",
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidResetPass), http.StatusBadRequest),
		},
		{
			desc:         "reset password with weak password",
			token:        validToken,
			newPassword:  "weak",
			confPassword: "weak",
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrPasswordFormat), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ResetSecret", mock.Anything, tc.token, tc.newPassword).Return(tc.svcErr)
			err := mgsdk.ResetPassword(tc.newPassword, tc.confPassword, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ResetSecret", mock.Anything, tc.token, tc.newPassword)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUpdatePassword(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	newPassword := "newPassword"
	updatedUser := user
	updatedUser.Credentials.Secret = newPassword

	cases := []struct {
		desc        string
		token       string
		oldPassword string
		newPassword string
		svcRes      mgclients.Client
		svcErr      error
		response    sdk.User
		err         errors.SDKError
	}{
		{
			desc:        "update password successfully",
			token:       validToken,
			oldPassword: secret,
			newPassword: newPassword,
			svcRes:      convertClient(updatedUser),
			svcErr:      nil,
			response:    updatedUser,
			err:         nil,
		},
		{
			desc:        "update password with invalid token",
			token:       invalidToken,
			oldPassword: secret,
			newPassword: newPassword,
			svcRes:      mgclients.Client{},
			svcErr:      svcerr.ErrAuthentication,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "update password with empty token",
			token:       "",
			oldPassword: secret,
			newPassword: newPassword,
			svcRes:      mgclients.Client{},
			svcErr:      nil,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:        "update password with empty old password",
			token:       validToken,
			oldPassword: "",
			newPassword: newPassword,
			svcRes:      mgclients.Client{},
			svcErr:      nil,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
		{
			desc:        "update password with empty new password",
			token:       validToken,
			oldPassword: secret,
			newPassword: "",
			svcRes:      mgclients.Client{},
			svcErr:      nil,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
		{
			desc:        "update password with invalid new password",
			token:       validToken,
			oldPassword: secret,
			newPassword: "weak",
			svcRes:      mgclients.Client{},
			svcErr:      nil,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrPasswordFormat), http.StatusBadRequest),
		},
		{
			desc:        "update password with invalid old password",
			token:       validToken,
			oldPassword: "wrongPassword",
			newPassword: newPassword,
			svcRes:      mgclients.Client{},
			svcErr:      svcerr.ErrLogin,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrLogin, http.StatusUnauthorized),
		},
		{
			desc:        "update password with response that can't be unmarshalled",
			token:       validToken,
			oldPassword: secret,
			newPassword: newPassword,
			svcRes: mgclients.Client{
				ID:   id,
				Name: user.Name,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateClientSecret", mock.Anything, tc.token, tc.oldPassword, tc.newPassword).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdatePassword(tc.oldPassword, tc.newPassword, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClientSecret", mock.Anything, tc.token, tc.oldPassword, tc.newPassword)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestUpdateUserRole(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedRole := mgclients.AdminRole.String()
	updatedUser := user
	updatedUser.Role = updatedRole

	cases := []struct {
		desc            string
		token           string
		updateClientReq sdk.User
		svcReq          mgclients.Client
		svcRes          mgclients.Client
		svcErr          error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update client role with valid token",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Role: updatedRole,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Role: mgclients.AdminRole,
			},
			svcRes:   convertClient(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update client role with invalid token",
			token: invalidToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Role: updatedRole,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Role: mgclients.AdminRole,
			},
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update client role with empty token",
			token: "",
			updateClientReq: sdk.User{
				ID:   user.ID,
				Role: updatedRole,
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:  "update client role with invalid id",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   wrongID,
				Role: updatedRole,
			},
			svcReq: mgclients.Client{
				ID:   wrongID,
				Role: mgclients.AdminRole,
			},
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update client role with empty id",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   "",
				Role: updatedRole,
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "update client role with request that can't be marshalled",
			token: validToken,
			updateClientReq: sdk.User{
				ID: generateUUID(t),
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   mgclients.Client{},
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "update client role with response that can't be unmarshalled",
			token: validToken,
			updateClientReq: sdk.User{
				ID:   user.ID,
				Role: updatedRole,
			},
			svcReq: mgclients.Client{
				ID:   user.ID,
				Role: mgclients.AdminRole,
			},
			svcRes: mgclients.Client{
				ID:   id,
				Role: mgclients.AdminRole,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateClientRole", mock.Anything, tc.token, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUserRole(tc.updateClientReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateClientRole", mock.Anything, tc.token, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestEnableUser(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enabledUser := user
	enabledUser.Status = mgclients.EnabledStatus.String()

	cases := []struct {
		desc     string
		token    string
		userID   string
		svcRes   mgclients.Client
		svcErr   error
		response sdk.User
		err      errors.SDKError
	}{
		{
			desc:     "enable user with valid token",
			token:    validToken,
			userID:   user.ID,
			svcRes:   convertClient(enabledUser),
			svcErr:   nil,
			response: enabledUser,
			err:      nil,
		},
		{
			desc:     "enable user with invalid token",
			token:    invalidToken,
			userID:   user.ID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "enable user with empty token",
			token:    "",
			userID:   user.ID,
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("EnableClient", mock.Anything, tc.token, tc.userID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.EnableUser(tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "EnableClient", mock.Anything, tc.token, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDisableUser(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	disabledUser := user
	disabledUser.Status = mgclients.DisabledStatus.String()

	cases := []struct {
		desc     string
		token    string
		userID   string
		svcRes   mgclients.Client
		svcErr   error
		response sdk.User
		err      errors.SDKError
	}{
		{
			desc:     "disable user with valid token",
			token:    validToken,
			userID:   user.ID,
			svcRes:   convertClient(disabledUser),
			svcErr:   nil,
			response: disabledUser,
			err:      nil,
		},
		{
			desc:     "disable user with invalid token",
			token:    invalidToken,
			userID:   user.ID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "disable user with empty token",
			token:    "",
			userID:   user.ID,
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "disable user with invalid id",
			token:    validToken,
			userID:   wrongID,
			svcRes:   mgclients.Client{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "disable user with empty id",
			token:    validToken,
			userID:   "",
			svcRes:   mgclients.Client{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:   "disable user with response that can't be unmarshalled",
			token:  validToken,
			userID: user.ID,
			svcRes: mgclients.Client{
				ID:     id,
				Status: mgclients.DisabledStatus,
				Metadata: mgclients.Metadata{
					"key": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DisableClient", mock.Anything, tc.token, tc.userID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DisableUser(tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DisableClient", mock.Anything, tc.token, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestDeleteUser(t *testing.T) {
	ts, svc := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc   string
		token  string
		userID string
		svcErr error
		err    errors.SDKError
	}{
		{
			desc:   "delete user successfully",
			token:  validToken,
			userID: validID,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:   "delete user with invalid token",
			token:  invalidToken,
			userID: validID,
			svcErr: svcerr.ErrAuthentication,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:   "delete user with empty token",
			token:  "",
			userID: validID,
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:   "delete user with invalid id",
			token:  validToken,
			userID: wrongID,
			svcErr: svcerr.ErrRemoveEntity,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrRemoveEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:   "delete user with empty id",
			token:  validToken,
			userID: "",
			svcErr: nil,
			err:    errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DeleteClient", mock.Anything, tc.token, tc.userID).Return(tc.svcErr)
			err := mgsdk.DeleteUser(tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DeleteClient", mock.Anything, tc.token, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}
