// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/groups"
	gmocks "github.com/absmach/magistrala/groups/mocks"
	internalapi "github.com/absmach/magistrala/internal/api"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	oauth2mocks "github.com/absmach/magistrala/pkg/oauth2/mocks"
	policies "github.com/absmach/magistrala/pkg/policies"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/users"
	"github.com/absmach/magistrala/users/api"
	umocks "github.com/absmach/magistrala/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	id       = generateUUID(&testing.T{})
	domainID = "c717fa97-ffd9-40cb-8cf9-7c2859059395"
)

func setupUsers() (*httptest.Server, *umocks.Service, *authnmocks.Authentication) {
	usvc := new(umocks.Service)
	gsvc := new(gmocks.Service)
	logger := mglog.NewMock()
	mux := chi.NewRouter()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	authn := new(authnmocks.Authentication)
	token := new(authmocks.TokenServiceClient)
	api.MakeHandler(usvc, authn, token, true, gsvc, mux, logger, "", passRegex, provider)

	return httptest.NewServer(mux), usvc, authn
}

func TestCreateUser(t *testing.T) {
	ts, svc, _ := setupUsers()
	defer ts.Close()

	createSdkUserReq := sdk.User{
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
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
		svcReq           users.User
		svcRes           users.User
		svcErr           error
		response         sdk.User
		err              errors.SDKError
	}{
		{
			desc:             "register new user successfully",
			token:            validToken,
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertUser(createSdkUserReq),
			svcRes:           convertUser(user),
			svcErr:           nil,
			response:         user,
			err:              nil,
		},
		{
			desc:             "register existing user",
			token:            validToken,
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertUser(createSdkUserReq),
			svcRes:           users.User{},
			svcErr:           svcerr.ErrCreateEntity,
			response:         sdk.User{},
			err:              errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:             "register user with invalid token",
			token:            invalidToken,
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertUser(createSdkUserReq),
			svcRes:           users.User{},
			svcErr:           svcerr.ErrAuthentication,
			response:         sdk.User{},
			err:              errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:             "register user with empty token",
			token:            "",
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertUser(createSdkUserReq),
			svcRes:           users.User{},
			svcErr:           svcerr.ErrAuthentication,
			response:         sdk.User{},
			err:              errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "register empty credentials user",
			token: validToken,
			createSdkUserReq: sdk.User{
				FirstName: createSdkUserReq.FirstName,
				LastName:  createSdkUserReq.LastName,
				Email:     createSdkUserReq.Email,
				Credentials: sdk.Credentials{
					Username: "",
					Secret:   "",
				},
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingUsername), http.StatusBadRequest),
		},
		{
			desc:  "register user with first name too long",
			token: validToken,
			createSdkUserReq: sdk.User{
				FirstName:   strings.Repeat("a", 1025),
				Credentials: createSdkUserReq.Credentials,
				Metadata:    createSdkUserReq.Metadata,
				Tags:        createSdkUserReq.Tags,
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:  "register user with empty userName",
			token: validToken,
			createSdkUserReq: sdk.User{
				FirstName: createSdkUserReq.FirstName,
				LastName:  createSdkUserReq.LastName,
				Email:     createSdkUserReq.Email,
				Credentials: sdk.Credentials{
					Username: "",
					Secret:   createSdkUserReq.Credentials.Secret,
				},
				Metadata: createSdkUserReq.Metadata,
				Tags:     createSdkUserReq.Tags,
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingUsername), http.StatusBadRequest),
		},
		{
			desc:  "register user with empty secret",
			token: validToken,
			createSdkUserReq: sdk.User{
				FirstName: createSdkUserReq.FirstName,
				LastName:  createSdkUserReq.LastName,
				Email:     createSdkUserReq.Email,
				Credentials: sdk.Credentials{
					Username: createSdkUserReq.Credentials.Username,
					Secret:   "",
				},
				Metadata: createSdkUserReq.Metadata,
				Tags:     createSdkUserReq.Tags,
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
		{
			desc:  "register user with secret that is too short",
			token: validToken,
			createSdkUserReq: sdk.User{
				FirstName: createSdkUserReq.FirstName,
				LastName:  createSdkUserReq.LastName,
				Email:     createSdkUserReq.Email,
				Credentials: sdk.Credentials{
					Username: createSdkUserReq.Credentials.Username,
					Secret:   "weak",
				},
				Metadata: createSdkUserReq.Metadata,
				Tags:     createSdkUserReq.Tags,
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrPasswordFormat), http.StatusBadRequest),
		},
		{
			desc:  "register a user with request that can't be marshalled",
			token: validToken,
			createSdkUserReq: sdk.User{
				Credentials: sdk.Credentials{
					Username: "user",
					Secret:   "12345678",
				},
				FirstName: createSdkUserReq.FirstName,
				LastName:  createSdkUserReq.LastName,
				Email:     createSdkUserReq.Email,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:             "register a user with response that can't be unmarshalled",
			token:            validToken,
			createSdkUserReq: createSdkUserReq,
			svcReq:           convertUser(createSdkUserReq),
			svcRes: users.User{
				ID:        id,
				FirstName: createSdkUserReq.FirstName,
				LastName:  createSdkUserReq.LastName,
				Email:     createSdkUserReq.Email,
				Credentials: users.Credentials{
					Username: createSdkUserReq.Credentials.Username,
					Secret:   createSdkUserReq.Credentials.Secret,
				},
				Metadata: users.Metadata{
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
			svcCall := svc.On("Register", mock.Anything, mgauthn.Session{}, tc.svcReq, true).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateUser(tc.createSdkUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Register", mock.Anything, authn.Session{}, tc.svcReq, true)
				assert.True(t, ok)
			}
			svcCall.Unset()
		})
	}
}

func TestListUsers(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	var cls []sdk.User
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		cl := sdk.User{
			ID:        generateUUID(t),
			FirstName: fmt.Sprintf("user_%d", i),
			Credentials: sdk.Credentials{
				Username: fmt.Sprintf("Username_%d", i),
				Secret:   fmt.Sprintf("password_%d", i),
			},
			Metadata: sdk.Metadata{"name": fmt.Sprintf("user_%d", i)},
			Status:   users.EnabledStatus.String(),
			Role:     users.UserRole.String(),
		}
		if i == 50 {
			cl.Status = users.DisabledStatus.String()
			cl.Tags = []string{"tag1", "tag2"}
		}
		cls = append(cls, cl)
	}

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		pageMeta        sdk.PageMetadata
		svcReq          users.Page
		svcRes          users.UsersPage
		svcErr          error
		authenticateErr error
		response        sdk.UsersPage
		err             errors.SDKError
	}{
		{
			desc:  "list users successfully",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: users.Page{
				Offset: offset,
				Limit:  limit,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: users.UsersPage{
				Page: users.Page{
					Total: uint64(len(cls[offset:limit])),
				},
				Users: convertUsers(cls[offset:limit]),
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
			desc:    "list users with invalid token",
			token:   invalidToken,
			session: mgauthn.Session{},
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq: users.Page{
				Offset: offset,
				Limit:  limit,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes:          users.UsersPage{},
			svcErr:          svcerr.ErrAuthentication,
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.UsersPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "list users with empty token",
			token: "",
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  limit,
			},
			svcReq:          users.Page{},
			svcRes:          users.UsersPage{},
			svcErr:          nil,
			authenticateErr: apiutil.ErrBearerToken,
			response:        sdk.UsersPage{},
			err:             errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "list users with zero limit",
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  0,
			},
			svcReq: users.Page{
				Offset: offset,
				Limit:  10,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: users.UsersPage{
				Page: users.Page{
					Total: uint64(len(cls[offset:10])),
				},
				Users: convertUsers(cls[offset:10]),
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
			token: validToken,
			pageMeta: sdk.PageMetadata{
				Offset: offset,
				Limit:  101,
			},
			svcReq:   users.Page{},
			svcRes:   users.UsersPage{},
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
				Metadata: sdk.Metadata{"name": "user_99"},
			},
			svcReq: users.Page{
				Offset:   offset,
				Limit:    limit,
				Metadata: users.Metadata{"name": "user_99"},
				Order:    internalapi.DefOrder,
				Dir:      internalapi.DefDir,
			},
			svcRes: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{convertUser(cls[89])},
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
				Status: users.DisabledStatus.String(),
			},
			svcReq: users.Page{
				Offset: offset,
				Limit:  limit,
				Status: users.DisabledStatus,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{convertUser(cls[50])},
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
			svcReq: users.Page{
				Offset: offset,
				Limit:  limit,
				Tag:    "tag1",
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{convertUser(cls[50])},
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
			svcReq: users.Page{
				Offset: offset,
				Limit:  limit,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes:   users.UsersPage{},
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
			svcReq: users.Page{
				Offset: offset,
				Limit:  limit,
				Order:  internalapi.DefOrder,
				Dir:    internalapi.DefDir,
			},
			svcRes: users.UsersPage{
				Page: users.Page{
					Total: uint64(len(cls[offset:limit])),
				},
				Users: []users.User{
					{
						ID:        id,
						FirstName: "user_99",
						Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("ListUsers", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Users(tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListUsers", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestSearchUsers(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	var cls []sdk.User
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		cl := sdk.User{
			ID:        generateUUID(t),
			FirstName: fmt.Sprintf("user_%d", i),
			Email:     fmt.Sprintf("email_%d", i),
			Credentials: sdk.Credentials{
				Username: fmt.Sprintf("Username_%d", i),
				Secret:   fmt.Sprintf("password_%d", i),
			},
			Metadata: sdk.Metadata{"name": fmt.Sprintf("user_%d", i)},
			Status:   users.EnabledStatus.String(),
			Role:     users.UserRole.String(),
		}
		if i == 50 {
			cl.Status = users.DisabledStatus.String()
			cl.Tags = []string{"tag1", "tag2"}
		}
		cls = append(cls, cl)
	}

	cases := []struct {
		desc            string
		token           string
		page            sdk.PageMetadata
		response        []sdk.User
		searchreturn    users.UsersPage
		err             errors.SDKError
		authenticateErr error
	}{
		{
			desc:  "search for users",
			token: validToken,
			err:   nil,
			page: sdk.PageMetadata{
				Offset:   offset,
				Limit:    limit,
				Username: "user_20",
			},
			response: []sdk.User{cls[10]},
			searchreturn: users.UsersPage{
				Users: []users.User{convertUser(cls[10])},
				Page: users.Page{
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
				Offset:   offset,
				Limit:    limit,
				Username: "user_10",
			},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response:        nil,
			authenticateErr: svcerr.ErrAuthentication,
		},
		{
			desc:  "search for users with empty token",
			token: "",
			page: sdk.PageMetadata{
				Offset:   offset,
				Limit:    limit,
				Username: "user_10",
			},
			err:             errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
			response:        nil,
			authenticateErr: svcerr.ErrAuthentication,
		},
		{
			desc:  "search for users with empty query",
			token: validToken,
			page: sdk.PageMetadata{
				Offset:    offset,
				Limit:     limit,
				FirstName: "",
			},
			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptySearchQuery), http.StatusBadRequest),
		},
		{
			desc:  "search for users with invalid length of query",
			token: validToken,
			page: sdk.PageMetadata{
				Offset:   offset,
				Limit:    limit,
				Username: "a",
			},
			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrLenSearchQuery, apiutil.ErrValidation), http.StatusBadRequest),
		},
		{
			desc:  "search for users with invalid limit",
			token: validToken,
			page: sdk.PageMetadata{
				Offset:   offset,
				Limit:    0,
				Username: "user_10",
			},
			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}, tc.authenticateErr)
			svcCall := svc.On("SearchUsers", mock.Anything, mock.Anything).Return(tc.searchreturn, tc.err)
			page, err := mgsdk.SearchUsers(tc.page, tc.token)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %v, got %v", tc.desc, tc.err, err))
			assert.Equal(t, tc.response, page.Users, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page.Users))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewUser(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		userID          string
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:     "view user successfully",
			token:    validToken,
			userID:   user.ID,
			svcRes:   convertUser(user),
			svcErr:   nil,
			response: user,
			err:      nil,
		},
		{
			desc:            "view user with invalid token",
			token:           invalidToken,
			userID:          user.ID,
			svcRes:          users.User{},
			svcErr:          svcerr.ErrAuthentication,
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view user with empty token",
			token:    "",
			userID:   user.ID,
			svcRes:   users.User{},
			svcErr:   svcerr.ErrAuthentication,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "view user with invalid id",
			token:    validToken,
			userID:   wrongID,
			svcRes:   users.User{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "view user with empty id",
			token:    validToken,
			userID:   "",
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:   "view user with response that can't be unmarshalled",
			token:  validToken,
			userID: user.ID,
			svcRes: users.User{
				ID:        id,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("View", mock.Anything, tc.session, tc.userID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.User(tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "View", mock.Anything, tc.session, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUserProfile(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:     "view user profile successfully",
			token:    validToken,
			svcRes:   convertUser(user),
			svcErr:   nil,
			response: user,
			err:      nil,
		},
		{
			desc:            "view user profile with invalid token",
			token:           invalidToken,
			svcRes:          users.User{},
			svcErr:          nil,
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "view user profile with empty token",
			token:    "",
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "view user profile with response that can't be unmarshalled",
			token: validToken,
			svcRes: users.User{
				ID:        id,
				FirstName: user.FirstName,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("ViewProfile", mock.Anything, tc.session).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UserProfile(tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewProfile", mock.Anything, tc.session)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateUser(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedName := "updatedName"
	updatedUser := user
	updatedUser.FirstName = updatedName

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		updateUserReq   sdk.User
		svcReq          users.User
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update user name with valid token",
			token: validToken,
			updateUserReq: sdk.User{
				ID:        user.ID,
				FirstName: updatedName,
			},
			svcReq: users.User{
				ID:        user.ID,
				FirstName: updatedName,
			},
			svcRes:   convertUser(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update user name with invalid token",
			token: invalidToken,
			updateUserReq: sdk.User{
				ID:        user.ID,
				FirstName: updatedName,
			},
			svcReq: users.User{
				ID:        user.ID,
				FirstName: updatedName,
			},
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update user name with invalid id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:        wrongID,
				FirstName: updatedName,
			},
			svcReq: users.User{
				ID:        wrongID,
				FirstName: updatedName,
			},
			svcRes:   users.User{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update user name with empty token",
			token: "",
			updateUserReq: sdk.User{
				ID:        user.ID,
				FirstName: updatedName,
			},
			svcReq: users.User{
				ID:        user.ID,
				FirstName: updatedName,
			},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "update user name with empty id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:        "",
				FirstName: updatedName,
			},
			svcReq: users.User{
				ID:        "",
				FirstName: updatedName,
			},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:  "update user with request that can't be marshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID: generateUUID(t),
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "update user with response that can't be unmarshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID:        user.ID,
				FirstName: updatedName,
			},
			svcReq: users.User{
				ID:        user.ID,
				FirstName: updatedName,
			},
			svcRes: users.User{
				ID:        id,
				FirstName: updatedName,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("Update", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUser(tc.updateUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Update", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateUserTags(t *testing.T) {
	ts, svc, auth := setupUsers()
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
		session         mgauthn.Session
		updateUserReq   sdk.User
		svcReq          users.User
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update user tags with valid token",
			token: validToken,
			updateUserReq: sdk.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcReq: users.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcRes:   convertUser(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update user tags with invalid token",
			token: invalidToken,
			updateUserReq: sdk.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcReq: users.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update user tags with empty token",
			token: "",
			updateUserReq: sdk.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "update user tags with invalid id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:   wrongID,
				Tags: updatedTags,
			},
			svcReq: users.User{
				ID:   wrongID,
				Tags: updatedTags,
			},
			svcRes:   users.User{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update user tags with empty id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:   "",
				Tags: updatedTags,
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "update user tags with request that can't be marshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID: generateUUID(t),
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "update user tags with response that can't be unmarshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcReq: users.User{
				ID:   user.ID,
				Tags: updatedTags,
			},
			svcRes: users.User{
				ID:   id,
				Tags: updatedTags,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("UpdateTags", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUserTags(tc.updateUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateTags", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateUserEmail(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedEmail := "updatedEmail@email.com"
	updatedUser := user
	updatedUser.Email = updatedEmail

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		updateUserReq   sdk.User
		svcReq          string
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update email with valid token",
			token: validToken,
			updateUserReq: sdk.User{
				ID:    user.ID,
				Email: updatedEmail,
				Credentials: sdk.Credentials{
					Secret: user.Credentials.Secret,
				},
			},
			svcReq:   updatedEmail,
			svcRes:   convertUser(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update email with invalid token",
			token: invalidToken,
			updateUserReq: sdk.User{
				ID:    user.ID,
				Email: updatedEmail,
				Credentials: sdk.Credentials{
					Secret: user.Credentials.Secret,
				},
			},
			svcReq:          updatedEmail,
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update email with empty token",
			token: "",
			updateUserReq: sdk.User{
				ID:    user.ID,
				Email: updatedEmail,
				Credentials: sdk.Credentials{
					Secret: user.Credentials.Secret,
				},
			},
			svcReq:   updatedEmail,
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "update email with invalid id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:    wrongID,
				Email: updatedEmail,
				Credentials: sdk.Credentials{
					Secret: user.Credentials.Secret,
				},
			},
			svcReq:   updatedEmail,
			svcRes:   users.User{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update email with empty id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:    "",
				Email: updatedEmail,
				Credentials: sdk.Credentials{
					Secret: user.Credentials.Secret,
				},
			},
			svcReq:   updatedEmail,
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "update email with response that can't be unmarshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID:    user.ID,
				Email: updatedEmail,
				Credentials: sdk.Credentials{
					Secret: user.Credentials.Secret,
				},
			},
			svcReq: updatedEmail,
			svcRes: users.User{
				ID:        id,
				FirstName: updatedEmail,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("UpdateEmail", mock.Anything, tc.session, tc.updateUserReq.ID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUserEmail(tc.updateUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateEmail", mock.Anything, tc.session, tc.updateUserReq.ID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestResetPasswordRequest(t *testing.T) {
	ts, svc, _ := setupUsers()
	defer ts.Close()

	defHost := "http://localhost"

	conf := sdk.Config{
		UsersURL: ts.URL,
		HostURL:  defHost,
	}
	mgsdk := sdk.NewSDK(conf)

	validEmail := "test@email.com"

	cases := []struct {
		desc     string
		email    string
		svcRes   users.User
		svcErr   error
		issueRes *magistrala.Token
		issueErr error
		err      errors.SDKError
	}{
		{
			desc:     "reset password request with valid email",
			email:    validEmail,
			svcRes:   convertUser(user),
			svcErr:   nil,
			issueRes: &magistrala.Token{AccessToken: validToken, RefreshToken: &validToken},
			err:      nil,
		},
		{
			desc:     "reset password request with invalid email",
			email:    "invalidemail",
			svcRes:   users.User{},
			svcErr:   svcerr.ErrViewEntity,
			issueRes: &magistrala.Token{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:     "reset password request with empty email",
			email:    "",
			svcRes:   users.User{},
			svcErr:   nil,
			issueRes: &magistrala.Token{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingEmail), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("GenerateResetToken", mock.Anything, tc.email, defHost).Return(tc.svcErr)
			svcCall1 := svc.On("SendPasswordReset", mock.Anything, mock.Anything, tc.email, user.Credentials.Username, tc.issueRes.AccessToken).Return(nil)
			err := mgsdk.ResetPasswordRequest(tc.email)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "GenerateResetToken", mock.Anything, tc.email, defHost)
				assert.True(t, ok)
			}
			svcCall.Unset()
			svcCall1.Unset()
		})
	}
}

func TestResetPassword(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	newPassword := "newPassword"

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		newPassword     string
		confPassword    string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:         "reset password successfully",
			token:        validToken,
			session:      mgauthn.Session{UserID: validID, DomainID: domainID},
			newPassword:  newPassword,
			confPassword: newPassword,
			svcErr:       nil,
			err:          nil,
		},
		{
			desc:            "reset password with invalid token",
			token:           invalidToken,
			newPassword:     newPassword,
			confPassword:    newPassword,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:         "reset password with empty token",
			token:        "",
			newPassword:  newPassword,
			confPassword: newPassword,
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:         "reset password with empty new password",
			token:        validToken,
			session:      mgauthn.Session{UserID: validID, DomainID: domainID},
			newPassword:  "",
			confPassword: newPassword,
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
		{
			desc:         "reset password with empty confirm password",
			token:        validToken,
			session:      mgauthn.Session{UserID: validID, DomainID: domainID},
			newPassword:  newPassword,
			confPassword: "",
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingConfPass), http.StatusBadRequest),
		},
		{
			desc:         "reset password with new password not matching confirm password",
			token:        validToken,
			session:      mgauthn.Session{UserID: validID, DomainID: domainID},
			newPassword:  newPassword,
			confPassword: "wrongPassword",
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidResetPass), http.StatusBadRequest),
		},
		{
			desc:         "reset password with weak password",
			token:        validToken,
			session:      mgauthn.Session{UserID: validID, DomainID: domainID},
			newPassword:  "weak",
			confPassword: "weak",
			svcErr:       nil,
			err:          errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrPasswordFormat), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("ResetSecret", mock.Anything, tc.session, tc.newPassword).Return(tc.svcErr)
			err := mgsdk.ResetPassword(tc.newPassword, tc.confPassword, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ResetSecret", mock.Anything, tc.session, tc.newPassword)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdatePassword(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	newPassword := "newPassword"
	updatedUser := user
	updatedUser.Credentials.Secret = newPassword

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		oldPassword     string
		newPassword     string
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:        "update password successfully",
			token:       validToken,
			oldPassword: secret,
			newPassword: newPassword,
			svcRes:      convertUser(updatedUser),
			svcErr:      nil,
			response:    updatedUser,
			err:         nil,
		},
		{
			desc:            "update password with invalid token",
			token:           invalidToken,
			oldPassword:     secret,
			newPassword:     newPassword,
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:        "update password with empty token",
			token:       "",
			oldPassword: secret,
			newPassword: newPassword,
			svcRes:      users.User{},
			svcErr:      nil,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:        "update password with empty old password",
			token:       validToken,
			oldPassword: "",
			newPassword: newPassword,
			svcRes:      users.User{},
			svcErr:      nil,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
		{
			desc:        "update password with empty new password",
			token:       validToken,
			oldPassword: secret,
			newPassword: "",
			svcRes:      users.User{},
			svcErr:      nil,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPass), http.StatusBadRequest),
		},
		{
			desc:        "update password with invalid new password",
			token:       validToken,
			oldPassword: secret,
			newPassword: "weak",
			svcRes:      users.User{},
			svcErr:      nil,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrPasswordFormat), http.StatusBadRequest),
		},
		{
			desc:        "update password with invalid old password",
			token:       validToken,
			oldPassword: "wrongPassword",
			newPassword: newPassword,
			svcRes:      users.User{},
			svcErr:      svcerr.ErrLogin,
			response:    sdk.User{},
			err:         errors.NewSDKErrorWithStatus(svcerr.ErrLogin, http.StatusUnauthorized),
		},
		{
			desc:        "update password with response that can't be unmarshalled",
			token:       validToken,
			oldPassword: secret,
			newPassword: newPassword,
			svcRes: users.User{
				ID:        id,
				FirstName: user.FirstName,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("UpdateSecret", mock.Anything, tc.session, tc.oldPassword, tc.newPassword).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdatePassword(tc.oldPassword, tc.newPassword, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateSecret", mock.Anything, tc.session, tc.oldPassword, tc.newPassword)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateUserRole(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedUser := user
	updatedRole := users.AdminRole.String()
	updatedUser.Role = updatedRole

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		updateUserReq   sdk.User
		svcReq          users.User
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update user role with valid token",
			token: validToken,
			updateUserReq: sdk.User{
				ID:    user.ID,
				Role:  updatedRole,
				Email: user.Email,
			},
			svcReq: users.User{
				ID:   user.ID,
				Role: users.AdminRole,
			},
			svcRes:   convertUser(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update user role with invalid token",
			token: invalidToken,
			updateUserReq: sdk.User{
				ID:   user.ID,
				Role: updatedRole,
			},
			svcReq: users.User{
				ID:   user.ID,
				Role: users.AdminRole,
			},
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update user role with empty token",
			token: "",
			updateUserReq: sdk.User{
				ID:   user.ID,
				Role: updatedRole,
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "update user role with invalid id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:   wrongID,
				Role: updatedRole,
			},
			svcReq: users.User{
				ID:   wrongID,
				Role: users.AdminRole,
			},
			svcRes:   users.User{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update user role with empty id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:   "",
				Role: updatedRole,
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "update user role with request that can't be marshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID: generateUUID(t),
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "update user role with response that can't be unmarshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID:   user.ID,
				Role: updatedRole,
			},
			svcReq: users.User{
				ID:   user.ID,
				Role: users.AdminRole,
			},
			svcRes: users.User{
				ID:   id,
				Role: users.AdminRole,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("UpdateRole", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUserRole(tc.updateUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateRole", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateUsername(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedUser := user
	updatedUsername := "updatedUsername"
	updatedUser.Credentials.Username = updatedUsername

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		updateUserReq   sdk.User
		svcReq          users.User
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update username with valid token",
			token: validToken,
			updateUserReq: sdk.User{
				ID: user.ID,
				Credentials: sdk.Credentials{
					Username: updatedUsername,
				},
			},
			svcReq: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: updatedUsername,
				},
			},
			svcRes:   convertUser(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update username with invalid token",
			token: invalidToken,
			updateUserReq: sdk.User{
				ID: user.ID,
				Credentials: sdk.Credentials{
					Username: updatedUsername,
				},
			},
			svcReq: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: updatedUsername,
				},
			},
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update username with empty token",
			token: "",
			updateUserReq: sdk.User{
				ID: user.ID,
				Credentials: sdk.Credentials{
					Username: updatedUsername,
				},
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "update username with invalid id",
			token: validToken,
			updateUserReq: sdk.User{
				ID: wrongID,
				Credentials: sdk.Credentials{
					Username: updatedUsername,
				},
			},
			svcReq: users.User{
				ID: wrongID,
				Credentials: users.Credentials{
					Username: updatedUsername,
				},
			},
			svcRes:   users.User{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update username with empty id",
			token: validToken,
			updateUserReq: sdk.User{
				ID: "",
				Credentials: sdk.Credentials{
					Username: updatedUsername,
				},
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "update username with response that can't be unmarshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID: user.ID,
				Credentials: sdk.Credentials{
					Username: updatedUsername,
				},
			},
			svcReq: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: updatedUsername,
				},
			},
			svcRes: users.User{
				ID: id,
				Credentials: users.Credentials{
					Username: updatedUsername,
				},
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("UpdateUsername", mock.Anything, tc.session, tc.svcReq.ID, tc.svcReq.Credentials.Username).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateUsername(tc.updateUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateUsername", mock.Anything, tc.session, tc.svcReq.ID, tc.svcReq.Credentials.Username)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateProfilePicture(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	updatedProfilePicture := "http://updated.com/profile.jpg"
	updatedUser := user
	updatedUser.Email = updatedProfilePicture

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		updateUserReq   sdk.User
		svcReq          users.User
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:  "update profile picture with valid token",
			token: validToken,
			updateUserReq: sdk.User{
				ID:             user.ID,
				ProfilePicture: updatedProfilePicture,
			},
			svcReq: users.User{
				ID:             user.ID,
				ProfilePicture: updatedProfilePicture,
			},
			svcRes:   convertUser(updatedUser),
			svcErr:   nil,
			response: updatedUser,
			err:      nil,
		},
		{
			desc:  "update profile picture with invalid token",
			token: invalidToken,
			updateUserReq: sdk.User{
				ID:             user.ID,
				ProfilePicture: updatedProfilePicture,
			},
			svcReq: users.User{
				ID:             user.ID,
				ProfilePicture: updatedProfilePicture,
			},
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:  "update profile picture with empty token",
			token: "",
			updateUserReq: sdk.User{
				ID:             user.ID,
				ProfilePicture: updatedProfilePicture,
			},
			svcReq: users.User{
				ID:             user.ID,
				ProfilePicture: updatedProfilePicture,
			},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:  "update profile picture with invalid id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:             wrongID,
				ProfilePicture: updatedProfilePicture,
			},
			svcReq: users.User{
				ID:             wrongID,
				ProfilePicture: updatedProfilePicture,
			},
			svcRes:   users.User{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:  "update profile picture with empty id",
			token: validToken,
			updateUserReq: sdk.User{
				ID:             "",
				ProfilePicture: updatedProfilePicture,
			},
			svcReq: users.User{
				ID:             "",
				ProfilePicture: updatedProfilePicture,
			},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:  "update profile picture with request that can't be marshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID: generateUUID(t),
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   users.User{},
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:  "update profile picture with response that can't be unmarshalled",
			token: validToken,
			updateUserReq: sdk.User{
				ID:             user.ID,
				ProfilePicture: updatedProfilePicture,
			},
			svcReq: users.User{
				ID:             user.ID,
				ProfilePicture: updatedProfilePicture,
			},
			svcRes: users.User{
				ID: id,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("UpdateProfilePicture", mock.Anything, tc.session, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateProfilePicture(tc.updateUserReq, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateProfilePicture", mock.Anything, tc.session, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableUser(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	enabledUser := user
	enabledUser.Status = users.EnabledStatus.String()

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		userID          string
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:     "enable user with valid token",
			token:    validToken,
			userID:   user.ID,
			svcRes:   convertUser(enabledUser),
			svcErr:   nil,
			response: enabledUser,
			err:      nil,
		},
		{
			desc:            "enable user with invalid token",
			token:           invalidToken,
			userID:          user.ID,
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "enable user with empty token",
			token:    "",
			userID:   user.ID,
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("Enable", mock.Anything, tc.session, tc.userID).Return(tc.svcRes, tc.svcErr)

			resp, err := mgsdk.EnableUser(tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Enable", mock.Anything, tc.session, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableUser(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	disabledUser := user
	disabledUser.Status = users.DisabledStatus.String()

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		userID          string
		svcRes          users.User
		svcErr          error
		authenticateErr error
		response        sdk.User
		err             errors.SDKError
	}{
		{
			desc:   "disable user with valid token",
			token:  validToken,
			userID: user.ID,
			svcRes: convertUser(disabledUser),
			svcErr: nil,

			response: disabledUser,
			err:      nil,
		},
		{
			desc:            "disable user with invalid token",
			token:           invalidToken,
			userID:          user.ID,
			svcRes:          users.User{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.User{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "disable user with empty token",
			token:    "",
			userID:   user.ID,
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "disable user with invalid id",
			token:    validToken,
			userID:   wrongID,
			svcRes:   users.User{},
			svcErr:   svcerr.ErrUpdateEntity,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrUpdateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:     "disable user with empty id",
			token:    validToken,
			userID:   "",
			svcRes:   users.User{},
			svcErr:   nil,
			response: sdk.User{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:   "disable user with response that can't be unmarshalled",
			token:  validToken,
			userID: user.ID,
			svcRes: users.User{
				ID:     id,
				Status: users.DisabledStatus,
				Metadata: users.Metadata{
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("Disable", mock.Anything, tc.session, tc.userID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DisableUser(tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Disable", mock.Anything, tc.session, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListMembers(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	member := generateTestUser(t)
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		groupID         string
		pageMeta        sdk.PageMetadata
		svcReq          users.Page
		svcRes          users.MembersPage
		svcErr          error
		authenticateErr error
		response        sdk.UsersPage
		err             errors.SDKError
	}{
		{
			desc:    "list members successfully",
			token:   validToken,
			groupID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq: users.Page{
				Offset:     0,
				Limit:      10,
				Permission: policies.ViewPermission,
			},
			svcRes: users.MembersPage{
				Page: users.Page{
					Total: 1,
				},
				Members: []users.User{convertUser(member)},
			},
			svcErr: nil,
			response: sdk.UsersPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Users: []sdk.User{member},
			},
		},
		{
			desc:    "list members with invalid token",
			token:   invalidToken,
			groupID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq: users.Page{
				Offset:     0,
				Limit:      10,
				Permission: policies.ViewPermission,
			},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.UsersPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:    "list members with empty token",
			token:   "",
			groupID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq:   users.Page{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:    "list members with invalid group id",
			token:   validToken,
			groupID: wrongID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq: users.Page{
				Offset:     0,
				Limit:      10,
				Permission: policies.ViewPermission,
			},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:    "list members with empty group id",
			token:   validToken,
			groupID: "",
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq:   users.Page{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:    "list members with page metadata that can't be marshalled",
			token:   validToken,
			groupID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   users.Page{},
			svcRes:   users.MembersPage{},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:    "list members with response that can't be unmarshalled",
			token:   validToken,
			groupID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq: users.Page{
				Offset:     0,
				Limit:      10,
				Permission: policies.ViewPermission,
			},
			svcRes: users.MembersPage{
				Page: users.Page{
					Total: 1,
				},
				Members: []users.User{{
					ID:        member.ID,
					FirstName: member.FirstName,
					Metadata: map[string]interface{}{
						"key": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.UsersPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("ListMembers", mock.Anything, tc.session, "groups", tc.groupID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Members(tc.groupID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListMembers", mock.Anything, tc.session, "groups", tc.groupID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteUser(t *testing.T) {
	ts, svc, auth := setupUsers()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		userID          string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:   "delete user successfully",
			token:  validToken,
			userID: validID,
			svcErr: nil,
			err:    nil,
		},
		{
			desc:            "delete user with invalid token",
			token:           invalidToken,
			userID:          validID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:   "delete user with empty token",
			token:  "",
			userID: validID,
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
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
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("Delete", mock.Anything, tc.session, tc.userID).Return(tc.svcErr)
			err := mgsdk.DeleteUser(tc.userID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Delete", mock.Anything, tc.session, tc.userID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListUserGroups(t *testing.T) {
	ts, svc, auth := setupGroups()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	group := generateTestGroup(t)
	cases := []struct {
		desc            string
		token           string
		session         mgauthn.Session
		userID          string
		pageMeta        sdk.PageMetadata
		svcReq          groups.Page
		svcRes          groups.Page
		svcErr          error
		authenticateErr error
		response        sdk.GroupsPage
		err             errors.SDKError
	}{
		{
			desc:   "list user groups successfully",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
				Permission: policies.ViewPermission,
				Direction:  -1,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{convertGroup(group)},
			},
			svcErr: nil,
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Groups: []sdk.Group{group},
			},
			err: nil,
		},
		{
			desc:   "list user groups with invalid token",
			token:  invalidToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
				Permission: policies.ViewPermission,
				Direction:  -1,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{convertGroup(group)},
			},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.GroupsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:   "list user groups with empty token",
			token:  "",
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:   "list user groups with invalid user id",
			token:  validToken,
			userID: wrongID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
				Permission: policies.ViewPermission,
				Direction:  -1,
			},
			svcRes:   groups.Page{},
			svcErr:   svcerr.ErrViewEntity,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:   "list user groups with page metadata that can't be marshalled",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			svcReq:   groups.Page{},
			svcRes:   groups.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:   "list user groups with response that can't be unmarshalled",
			token:  validToken,
			userID: validID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
			},
			svcReq: groups.Page{
				PageMeta: groups.PageMeta{
					Offset: 0,
					Limit:  10,
				},
				Permission: policies.ViewPermission,
				Direction:  -1,
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{{
					ID:   group.ID,
					Name: group.Name,
					Metadata: map[string]interface{}{
						"key": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := svc.On("ListGroups", mock.Anything, tc.session, "users", tc.userID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ListUserGroups(tc.userID, tc.pageMeta, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, "users", tc.userID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}
