// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	authmocks "github.com/absmach/supermq/auth/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	oauth2mocks "github.com/absmach/supermq/pkg/oauth2/mocks"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/absmach/supermq/users"
	usersapi "github.com/absmach/supermq/users/api"
	"github.com/absmach/supermq/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	secret         = "strongsecret"
	validCMetadata = users.Metadata{"role": "user"}
	user           = users.User{
		ID:              testsutil.GenerateUUID(&testing.T{}),
		LastName:        "doe",
		FirstName:       "jane",
		Tags:            []string{"foo", "bar"},
		Email:           "useremail@example.com",
		Credentials:     users.Credentials{Username: "username", Secret: secret},
		Metadata:        validCMetadata,
		PrivateMetadata: validCMetadata,
		Status:          users.EnabledStatus,
	}
	validToken      = "valid"
	inValidToken    = "invalid"
	inValid         = "invalid"
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	passRegex       = regexp.MustCompile("^.{8,}$")
	testReferer     = "http://localhost"
	domainID        = testsutil.GenerateUUID(&testing.T{})
	verifiedSession = smqauthn.Session{UserID: validID, DomainID: domainID, Verified: true}
)

const contentType = "application/json"

type testRequest struct {
	user        *http.Client
	method      string
	url         string
	contentType string
	referer     string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}

	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	req.Header.Set("Referer", tr.referer)

	return tr.user.Do(req)
}

func newUsersServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	logger := smqlog.NewMock()
	mux := chi.NewRouter()
	idp := uuid.NewMock()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	authn := new(authnmocks.Authentication)
	am := smqauthn.NewAuthNMiddleware(authn)
	token := new(authmocks.TokenServiceClient)
	usersapi.MakeHandler(svc, am, token, true, mux, logger, "", passRegex, idp, provider)

	return httptest.NewServer(mux), svc, authn
}

func toJSON(data any) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestRegister(t *testing.T) {
	us, svc, _ := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc        string
		user        users.User
		token       string
		contentType string
		status      int
		err         error
	}{
		{
			desc:        "register a new user with a valid token",
			user:        user,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "register an existing user",
			user:        user,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrConflict,
		},
		{
			desc: "register a user that can't be marshalled",
			user: users.User{
				Email: "user@example.com",
				Credentials: users.Credentials{
					Secret: "12345678",
				},
				PrivateMetadata: map[string]any{
					"test": make(chan int),
				},
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
		{
			desc: "register user with invalid status",
			user: users.User{
				Email:     "newclientwithinvalidstatus@example.com",
				FirstName: "newclientwithinvalidstatus",
				LastName:  "newclientwithinvalidstatus",
				Credentials: users.Credentials{
					Username: "username",
					Secret:   secret,
				},
				Status: users.AllStatus,
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrInvalidStatus,
		},
		{
			desc: "register a user with name too long",
			user: users.User{
				FirstName: strings.Repeat("a", 1025),
				LastName:  "newuserwithnametoolong",
				Email:     "newuserwithinvalidname@example.com",
				Credentials: users.Credentials{
					Secret: secret,
				},
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrNameSize,
		},
		{
			desc:        "register user with invalid content type",
			user:        user,
			token:       validToken,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "register user with empty request body",
			user:        users.User{},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingFirstName,
		},
		{
			desc: "register user with invalid username",
			user: users.User{
				FirstName: "newuserwithinvalidusername",
				LastName:  "newuserwithinvalidusername",
				Email:     "user@example.com",
				Credentials: users.Credentials{
					Username: "invalid username",
					Secret:   secret,
				},
				Status: users.EnabledStatus,
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrInvalidUsername,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.user)
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/users/", us.URL),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			svcCall := svc.On("Register", mock.Anything, smqauthn.Session{}, tc.user, true).Return(tc.user, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
		})
	}
}

func TestView(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		status   int
		authnRes smqauthn.Session
		authnErr error
		svcErr   error
		err      error
	}{
		{
			desc:     "view user as admin with valid token",
			token:    validToken,
			id:       user.ID,
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "view user with invalid token",
			token:    inValidToken,
			id:       user.ID,
			status:   http.StatusUnauthorized,
			authnRes: smqauthn.Session{},
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view user with empty token",
			token:    "",
			id:       user.ID,
			status:   http.StatusUnauthorized,
			authnRes: smqauthn.Session{},
			authnErr: svcerr.ErrAuthentication,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view user as normal user successfully",
			token:    validToken,
			id:       user.ID,
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "view user with invalid ID",
			token:    validToken,
			id:       inValid,
			status:   http.StatusUnprocessableEntity,
			authnRes: verifiedSession,
			svcErr:   svcerr.ErrViewEntity,
			err:      svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:   us.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/users/%s", us.URL, tc.id),
				token:  tc.token,
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("View", mock.Anything, tc.authnRes, tc.id).Return(users.User{}, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestViewProfile(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		status   int
		authnRes smqauthn.Session
		authnErr error
		svcErr   error
		err      error
	}{
		{
			desc:     "view profile with valid token",
			token:    validToken,
			id:       user.ID,
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "view profile with invalid token",
			token:    inValidToken,
			id:       user.ID,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			authnRes: smqauthn.Session{},
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view profile with empty token",
			token:    "",
			id:       user.ID,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			authnRes: smqauthn.Session{},
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view profile with service error",
			token:    validToken,
			id:       user.ID,
			status:   http.StatusUnprocessableEntity,
			authnRes: verifiedSession,
			svcErr:   svcerr.ErrViewEntity,
			err:      svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:   us.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/users/profile", us.URL),
				token:  tc.token,
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ViewProfile", mock.Anything, tc.authnRes).Return(users.User{}, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestListUsers(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc              string
		query             string
		token             string
		pageMeta          users.Page
		listUsersResponse users.UsersPage
		status            int
		authnRes          smqauthn.Session
		authnErr          error
		err               error
	}{
		{
			desc:   "list users as admin with valid token",
			token:  validToken,
			status: http.StatusOK,
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with empty token",
			token:    "",
			status:   http.StatusUnauthorized,
			authnRes: smqauthn.Session{},
			authnErr: svcerr.ErrAuthentication,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list users with invalid token",
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnRes: smqauthn.Session{},
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:  "list users with offset",
			token: validToken,
			pageMeta: users.Page{
				Offset: 1,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Offset: 1,
					Total:  1,
				},
				Users: []users.User{user},
			},
			query:    "offset=1",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with invalid offset",
			token:    validToken,
			query:    "offset=invalid",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list users with limit",
			token: validToken,
			pageMeta: users.Page{
				Offset: 0,
				Limit:  1,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Limit: 1,
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "limit=1",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with invalid limit",
			token:    validToken,
			query:    "limit=invalid",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list users with limit greater than max",
			token:    validToken,
			query:    fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrLimitSize,
		},
		{
			desc:  "list users with username",
			token: validToken,
			pageMeta: users.Page{
				Offset:   0,
				Limit:    10,
				Order:    api.DefOrder,
				Dir:      api.DefDir,
				Username: "username",
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "username=username",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with duplicate username",
			token:    validToken,
			query:    "username=1&username=2",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list users with first name",
			token: validToken,
			pageMeta: users.Page{
				Offset:    0,
				Limit:     10,
				Order:     api.DefOrder,
				Dir:       api.DefDir,
				FirstName: "firstname",
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "first_name=firstname",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with duplicate firstname",
			token:    validToken,
			query:    "status=invalid",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      svcerr.ErrInvalidStatus,
		},
		{
			desc:     "list users with duplicate status",
			token:    validToken,
			query:    "status=enabled&status=disabled",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list users with lastname",
			token: validToken,
			pageMeta: users.Page{
				Offset:   0,
				Limit:    10,
				Order:    api.DefOrder,
				Dir:      api.DefDir,
				LastName: "lastname",
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "last_name=lastname",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with duplicate lastname",
			token:    validToken,
			query:    "last_name=lastname1&last_name=lastname2",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list users with status",
			token: validToken,
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Status: users.EnabledStatus,
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "status=enabled",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with invalid status",
			token:    validToken,
			query:    "status=invalid",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      svcerr.ErrInvalidStatus,
		},
		{
			desc:     "list users with duplicate status",
			token:    validToken,
			query:    "status=enabled&status=disabled",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list users with single tag",
			token: validToken,
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Tags:   users.TagsQuery{Elements: []string{"tag1"}, Operator: users.OrOp},
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "tags=tag1",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:  "list users with multiple tags and OR operator",
			token: validToken,
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Tags:   users.TagsQuery{Elements: []string{"tag1", "tag2", "tag3"}, Operator: users.OrOp},
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "tags=tag1,tag2,tag3",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:  "list users with multiple tags and AND operator",
			token: validToken,
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Tags:   users.TagsQuery{Elements: []string{"tag1", "tag2", "tag3"}, Operator: users.AndOp},
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "tags=tag1%2Btag2%2Btag3",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with duplicate tags",
			token:    validToken,
			query:    "tags=tag1&tags=tag2",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list users with metadata",
			token: validToken,
			pageMeta: users.Page{
				Offset:   0,
				Limit:    10,
				Order:    api.DefOrder,
				Dir:      api.DefDir,
				Metadata: users.Metadata{"domain": "example.com"},
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:    "metadata=" + url.PathEscape(`{"domain": "example.com"}`),
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with invalid metadata",
			token:    validToken,
			query:    "metadata=invalid",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list users with duplicate metadata",
			token:    validToken,
			query:    fmt.Sprintf("metadata=%s&metadata=%s", url.PathEscape(`{"domain": "example.com"}`), url.PathEscape(`{"domain": "example.com"}`)),
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list users with email",
			token: validToken,
			query: fmt.Sprintf("email=%s", user.Email),
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Email:  user.Email,
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with duplicate email",
			token:    validToken,
			query:    "email=1&email=2",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc: "list users with order",
			pageMeta: users.Page{
				Offset: 0,
				Limit:  10,
				Order:  "username",
				Dir:    api.DefDir,
			},
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{
					user,
				},
			},
			token:    validToken,
			query:    "order=username",
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "list users with duplicate order",
			token:    validToken,
			query:    "order=name&order=name",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list users with invalid order direction",
			token:    validToken,
			query:    "dir=invalid",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidDirection,
		},
		{
			desc:     "list users with duplicate order direction",
			token:    validToken,
			query:    "dir=asc&dir=asc",
			status:   http.StatusBadRequest,
			authnRes: verifiedSession,
			err:      apiutil.ErrInvalidQueryParams,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodGet,
				url:         us.URL + "/users?" + tc.query,
				contentType: contentType,
				token:       tc.token,
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ListUsers", mock.Anything, tc.authnRes, tc.pageMeta).Return(tc.listUsersResponse, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var bodyRes respBody
			err = json.NewDecoder(res.Body).Decode(&bodyRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if bodyRes.Err != "" || bodyRes.Message != "" {
				err = errors.Wrap(errors.New(bodyRes.Err), errors.New(bodyRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestSearchUsers(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc              string
		token             string
		page              users.Page
		status            int
		query             string
		listUsersResponse users.UsersPage
		authnErr          error
		svcErr            error
		err               error
	}{
		{
			desc:   "search users with valid token",
			token:  validToken,
			status: http.StatusOK,
			query:  "username=username",
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Total: 1,
				},
				Users: []users.User{user},
			},
			err: nil,
		},
		{
			desc:     "search users with empty token",
			token:    "",
			query:    "username=username",
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "search users with invalid token",
			token:    inValidToken,
			query:    "username=username",
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:  "search users with offset",
			token: validToken,
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Offset: 1,
					Total:  1,
				},
				Users: []users.User{user},
			},
			query:  "username=username&offset=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "search users with invalid offset",
			token:  validToken,
			query:  "username=username&offset=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:  "search users with limit",
			token: validToken,
			listUsersResponse: users.UsersPage{
				Page: users.Page{
					Limit: 1,
					Total: 1,
				},
				Users: []users.User{user},
			},
			query:  "username=username&limit=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "search users with invalid limit",
			token:  validToken,
			query:  "username=username&limit=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "search users with empty query",
			token:  validToken,
			query:  "",
			status: http.StatusBadRequest,
			err:    apiutil.ErrEmptySearchQuery,
		},
		{
			desc:   "search users with invalid length of query",
			token:  validToken,
			query:  "username=a",
			status: http.StatusBadRequest,
			err:    apiutil.ErrLenSearchQuery,
		},
		{
			desc:   "serach users with service error",
			token:  validToken,
			query:  "username=username",
			status: http.StatusUnprocessableEntity,
			svcErr: svcerr.ErrViewEntity,
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:   us.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/users/search?", us.URL) + tc.query,
				token:  tc.token,
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(verifiedSession, tc.authnErr)
			svcCall := svc.On("SearchUsers", mock.Anything, mock.Anything).Return(
				users.UsersPage{
					Page:  tc.listUsersResponse.Page,
					Users: tc.listUsersResponse.Users,
				},
				tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestUpdate(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	newName := "newname"
	newMetadata := users.Metadata{"newkey": "newvalue"}

	cases := []struct {
		desc         string
		id           string
		data         string
		userResponse users.User
		token        string
		authnRes     smqauthn.Session
		authnErr     error
		contentType  string
		status       int
		err          error
	}{
		{
			desc:        "update as admin user with valid token",
			id:          user.ID,
			data:        fmt.Sprintf(`{"name":"%s","metadata":%s, "private_metadata":%s}`, newName, toJSON(newMetadata), toJSON(newMetadata)),
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			userResponse: users.User{
				ID:              user.ID,
				FirstName:       newName,
				Metadata:        newMetadata,
				PrivateMetadata: newMetadata,
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:        "update as normal user with valid token",
			id:          user.ID,
			data:        fmt.Sprintf(`{"name":"%s","metadata":%s}`, newName, toJSON(newMetadata)),
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			userResponse: users.User{
				ID:        user.ID,
				FirstName: newName,
				Metadata:  newMetadata,
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:        "update user with invalid token",
			id:          user.ID,
			data:        fmt.Sprintf(`{"name":"%s","metadata":%s}`, newName, toJSON(newMetadata)),
			token:       inValidToken,
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update user with empty token",
			id:          user.ID,
			data:        fmt.Sprintf(`{"name":"%s","metadata":%s}`, newName, toJSON(newMetadata)),
			token:       "",
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update user with invalid id",
			id:          inValid,
			data:        fmt.Sprintf(`{"name":"%s","metadata":%s}`, newName, toJSON(newMetadata)),
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "update user with invalid contentype",
			id:          user.ID,
			data:        fmt.Sprintf(`{"name":"%s","metadata":%s}`, newName, toJSON(newMetadata)),
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update user with malformed data",
			id:          user.ID,
			data:        fmt.Sprintf(`{"name":%s}`, "invalid"),
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
		{
			desc:        "update user with empty id",
			id:          " ",
			data:        fmt.Sprintf(`{"name":"%s","metadata":%s}`, newName, toJSON(newMetadata)),
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusUnprocessableEntity,
			err:         svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/users/%s", us.URL, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Update", mock.Anything, tc.authnRes, tc.id, mock.Anything).Return(tc.userResponse, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var resBody respBody
			err = json.NewDecoder(res.Body).Decode(&resBody)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if resBody.Err != "" || resBody.Message != "" {
				err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestUpdateTags(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	defer us.Close()
	newTag := "newtag"

	cases := []struct {
		desc         string
		id           string
		data         string
		contentType  string
		userResponse users.User
		token        string
		authnRes     smqauthn.Session
		authnErr     error
		status       int
		err          error
	}{
		{
			desc:        "updateuser tags as admin with valid token",
			id:          user.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			userResponse: users.User{
				ID:   user.ID,
				Tags: []string{newTag},
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:        "updateuser tags as normal user with valid token",
			id:          user.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			userResponse: users.User{
				ID:   user.ID,
				Tags: []string{newTag},
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:        "update user tags with empty token",
			id:          user.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       "",
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update user tags with invalid token",
			id:          user.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       inValidToken,
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update user tags with invalid id",
			id:          user.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "update user tags with invalid contentype",
			id:          user.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: "application/xml",
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update user tags with empty id",
			id:          "",
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "update user with malfomed data",
			id:          user.ID,
			data:        fmt.Sprintf(`{"tags":%s}`, newTag),
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/users/%s/tags", us.URL, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("UpdateTags", mock.Anything, tc.authnRes, tc.id, mock.Anything).Return(tc.userResponse, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var resBody respBody
			err = json.NewDecoder(res.Body).Decode(&resBody)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if resBody.Err != "" || resBody.Message != "" {
				err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
			}
			if err == nil {
				assert.Equal(t, tc.userResponse.Tags, resBody.Tags, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.userResponse.Tags, resBody.Tags))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestUpdateEmail(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	newuseremail := "newuseremail@example.com"

	cases := []struct {
		desc        string
		data        string
		user        users.User
		contentType string
		token       string
		authnRes    smqauthn.Session
		authnErr    error
		status      int
		svcErr      error
		err         error
	}{
		{
			desc: "update user email as admin with valid token",
			data: fmt.Sprintf(`{"email": "%s"}`, newuseremail),
			user: users.User{
				ID:    user.ID,
				Email: newuseremail,
				Credentials: users.Credentials{
					Secret: "secret",
				},
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc: "update user email as normal user with valid token",
			data: fmt.Sprintf(`{"email": "%s"}`, newuseremail),
			user: users.User{
				ID:    user.ID,
				Email: newuseremail,
				Credentials: users.Credentials{
					Secret: "secret",
				},
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID},
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc: "update user email with empty token",
			data: fmt.Sprintf(`{"email": "%s"}`, newuseremail),
			user: users.User{
				ID:    user.ID,
				Email: newuseremail,
				Credentials: users.Credentials{
					Secret: "secret",
				},
			},
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc: "update user email with invalid token",
			data: fmt.Sprintf(`{"email": "%s"}`, newuseremail),
			user: users.User{
				ID:    user.ID,
				Email: newuseremail,
				Credentials: users.Credentials{
					Secret: "secret",
				},
			},
			contentType: contentType,
			token:       inValid,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "update user email with empty id",
			data: fmt.Sprintf(`{"email": "%s"}`, newuseremail),
			user: users.User{
				ID:    "",
				Email: newuseremail,
				Credentials: users.Credentials{
					Secret: "secret",
				},
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc: "update user email with invalid contentype",
			data: fmt.Sprintf(`{"email": "%s"}`, ""),
			user: users.User{
				ID:    user.ID,
				Email: newuseremail,
				Credentials: users.Credentials{
					Secret: "secret",
				},
			},
			contentType: "application/xml",
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc: "update user email with malformed data",
			data: fmt.Sprintf(`{"email": %s}`, "invalid"),
			user: users.User{
				ID:    user.ID,
				Email: "",
				Credentials: users.Credentials{
					Secret: "secret",
				},
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
		{
			desc: "update user email with service error",
			data: fmt.Sprintf(`{"email": "%s"}`, newuseremail),
			user: users.User{
				ID:    user.ID,
				Email: newuseremail,
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusUnprocessableEntity,
			svcErr:      svcerr.ErrUpdateEntity,
			err:         svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/users/%s/email", us.URL, tc.user.ID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("UpdateEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.user, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var resBody respBody
			err = json.NewDecoder(res.Body).Decode(&resBody)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if resBody.Err != "" || resBody.Message != "" {
				err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestUpdateUsername(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	newusername := "newusername"

	cases := []struct {
		desc        string
		data        string
		user        users.User
		contentType string
		token       string
		authnRes    smqauthn.Session
		authnErr    error
		status      int
		err         error
	}{
		{
			desc: "update username as admin with valid token",
			data: fmt.Sprintf(`{"username": "%s"}`, newusername),
			user: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: newusername,
				},
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc: "update username with empty token",
			data: fmt.Sprintf(`{"username": "%s"}`, newusername),
			user: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: newusername,
				},
			},
			authnRes:    smqauthn.Session{Type: smqauthn.AccessToken, Verified: true},
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc: "update username with invalid token",
			data: fmt.Sprintf(`{"username": "%s"}`, newusername),
			user: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: newusername,
				},
			},
			authnRes:    smqauthn.Session{Type: smqauthn.AccessToken, Verified: true},
			contentType: contentType,
			token:       inValid,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "update username with empty id",
			data: fmt.Sprintf(`{"username": "%s"}`, newusername),
			user: users.User{
				ID: "",
				Credentials: users.Credentials{
					Username: newusername,
				},
			},
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc: "update username with invalid contentype",
			data: fmt.Sprintf(`{"username": "%s"}`, ""),
			user: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: newusername,
				},
			},
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			contentType: "application/xml",
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc: "update user email with malformed data",
			data: fmt.Sprintf(`{"email": %s}`, "invalid"),
			user: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: newusername,
				},
			},
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
		{
			desc: "update username with invalid username",
			data: fmt.Sprintf(`{"username": "%s"}`, "invalid"),
			user: users.User{
				ID: user.ID,
				Credentials: users.Credentials{
					Username: newusername,
				},
			},
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			contentType: contentType,
			token:       validToken,
			status:      http.StatusUnprocessableEntity,
			err:         svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/users/%s/username", us.URL, tc.user.ID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("UpdateUsername", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.user, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var resBody respBody
			err = json.NewDecoder(res.Body).Decode(&resBody)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if resBody.Err != "" || resBody.Message != "" {
				err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestUpdateProfilePicture(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	newprofilepicture := "https://example.com/newprofilepicture"

	cases := []struct {
		desc        string
		data        string
		user        users.User
		contentType string
		token       string
		authnRes    smqauthn.Session
		authnErr    error
		status      int
		svcErr      error
		err         error
	}{
		{
			desc: "update profile picture as admin with valid token",
			data: fmt.Sprintf(`{"profile_picture": "%s"}`, newprofilepicture),
			user: users.User{
				ID:             user.ID,
				ProfilePicture: newprofilepicture,
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    smqauthn.Session{UserID: validID, DomainID: domainID, Role: smqauthn.AdminRole},
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update profile picture with empty token",
			data:        fmt.Sprintf(`{"profile_picture": "%s"}`, newprofilepicture),
			user:        users.User{},
			authnRes:    smqauthn.Session{Type: smqauthn.AccessToken, Verified: true},
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update profile_picture with invalid token",
			data:        fmt.Sprintf(`{"profile_picture": "%s"}`, newprofilepicture),
			user:        users.User{},
			contentType: contentType,
			token:       inValid,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "update profile_picture with empty id",
			data: fmt.Sprintf(`{"profile_picture": "%s"}`, newprofilepicture),
			user: users.User{
				ID:             "",
				ProfilePicture: newprofilepicture,
			},

			contentType: contentType,
			token:       validToken,
			authnRes:    smqauthn.Session{UserID: validID, DomainID: validID, Verified: true},
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc: "update profile_picture with invalid contentype",
			data: fmt.Sprintf(`{"profile_picture": "%s"}`, ""),
			user: users.User{
				ID:             user.ID,
				ProfilePicture: newprofilepicture,
			},
			authnRes:    smqauthn.Session{Type: smqauthn.AccessToken, Verified: true},
			contentType: "application/xml",
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update profile picture with malformed data",
			data:        fmt.Sprintf(`{"profile_picture": %s}`, "invalid"),
			user:        users.User{},
			authnRes:    smqauthn.Session{Type: smqauthn.AccessToken, Verified: true},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
		{
			desc: "update profile picture with failed to update",
			data: fmt.Sprintf(`{"profile_picture": "%s"}`, "invalid"),
			user: users.User{
				ID: user.ID,
			},
			authnRes:    smqauthn.Session{Type: smqauthn.AccessToken, Verified: true},
			contentType: contentType,
			token:       validToken,
			status:      http.StatusUnprocessableEntity,
			svcErr:      svcerr.ErrUpdateEntity,
			err:         svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/users/%s/picture", us.URL, tc.user.ID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("UpdateProfilePicture", mock.Anything, tc.authnRes, tc.user.ID, mock.Anything).Return(tc.user, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var resBody respBody
			err = json.NewDecoder(res.Body).Decode(&resBody)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if resBody.Err != "" || resBody.Message != "" {
				err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestPasswordResetRequest(t *testing.T) {
	us, svc, _ := newUsersServer()
	defer us.Close()

	testemail := "test@example.com"
	testhost := "example.com"

	cases := []struct {
		desc        string
		data        string
		contentType string
		referer     string
		status      int
		generateErr error
		sendErr     error
		err         error
	}{
		{
			desc:        "password reset request with valid email",
			data:        fmt.Sprintf(`{"email": "%s", "host": "%s"}`, testemail, testhost),
			contentType: contentType,
			referer:     testReferer,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "password reset request with empty email",
			data:        fmt.Sprintf(`{"email": "%s", "host": "%s"}`, "", testhost),
			contentType: contentType,
			referer:     testReferer,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "password reset request with invalid email",
			data:        fmt.Sprintf(`{"email": "%s", "host": "%s"}`, "invalid", testhost),
			contentType: contentType,
			referer:     testReferer,
			status:      http.StatusNotFound,
			generateErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
		},
		{
			desc:        "password reset with malformed data",
			data:        fmt.Sprintf(`{"email": %s, "host": %s}`, testemail, testhost),
			contentType: contentType,
			referer:     testReferer,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "password reset with invalid content type",
			data:        fmt.Sprintf(`{"email": "%s", "host": "%s"}`, testemail, testhost),
			contentType: "application/xml",
			referer:     testReferer,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "password reset with failed to issue token",
			data:        fmt.Sprintf(`{"email": "%s", "host": "%s"}`, testemail, testhost),
			contentType: contentType,
			referer:     testReferer,
			status:      http.StatusUnauthorized,
			generateErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/password/reset-request", us.URL),
				contentType: tc.contentType,
				referer:     tc.referer,
				body:        strings.NewReader(tc.data),
			}
			svcCall := svc.On("SendPasswordReset", mock.Anything, mock.Anything).Return(tc.generateErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
		})
	}
}

func TestSendVerification(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc     string
		token    string
		status   int
		authnRes smqauthn.Session
		authnErr error
		svcErr   error
		err      error
	}{
		{
			desc:     "send verification with valid token",
			token:    validToken,
			status:   http.StatusOK,
			authnRes: verifiedSession,
			err:      nil,
		},
		{
			desc:     "send verification with invalid token",
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			authnRes: smqauthn.Session{},
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "send verification with empty token",
			token:    "",
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			authnRes: smqauthn.Session{},
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "send verification with service error",
			token:    validToken,
			status:   http.StatusUnprocessableEntity,
			authnRes: verifiedSession,
			svcErr:   svcerr.ErrCreateEntity,
			err:      svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:   us.Client(),
				method: http.MethodPost,
				url:    fmt.Sprintf("%s/users/send-verification", us.URL),
				token:  tc.token,
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("SendVerification", mock.Anything, tc.authnRes).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			body, err := io.ReadAll(res.Body)
			if err != nil {
				assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while reading response body: %s", tc.desc, err))
			}
			defer res.Body.Close()
			var errRes respBody
			if len(body) > 0 {
				if err := json.Unmarshal(body, &errRes); err != nil {
					assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while unmarshal response body: %s", tc.desc, err))
				}
			}
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestVerifyEmail(t *testing.T) {
	us, svc, _ := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		token  string
		status int
		svcErr error
		err    error
	}{
		{
			desc:   "verify email with valid token",
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "verify email with empty token",
			token:  "",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidVerification,
		},
		{
			desc:   "verify email with service error",
			token:  validToken,
			status: http.StatusUnprocessableEntity,
			svcErr: svcerr.ErrUpdateEntity,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:   us.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/verify-email?token=%s", us.URL, tc.token),
			}

			svcCall := svc.On("VerifyEmail", mock.Anything, mock.Anything).Return(users.User{}, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			body, err := io.ReadAll(res.Body)
			if err != nil {
				assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while reading response body: %s", tc.desc, err))
			}
			defer res.Body.Close()
			var errRes respBody
			if len(body) > 0 {
				if err := json.Unmarshal(body, &errRes); err != nil {
					assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while unmarshal response body: %s", tc.desc, err))
				}
			}
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
		})
	}
}

func TestPasswordReset(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	strongPass := "StrongPassword"

	cases := []struct {
		desc        string
		data        string
		token       string
		contentType string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		svcErr      error
		err         error
	}{
		{
			desc:        "password reset with valid token",
			data:        fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, validToken, strongPass, strongPass),
			token:       validToken,
			authnRes:    smqauthn.Session{Type: smqauthn.AccessToken, Verified: true},
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "password reset with forgotten password",
			data:        fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, validToken, strongPass, strongPass),
			token:       validToken,
			authnRes:    smqauthn.Session{Type: smqauthn.AccessToken, Verified: false},
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "password reset with invalid token",
			data:        fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, inValidToken, strongPass, strongPass),
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "password reset to weak password",
			data:        fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, validToken, "weak", "weak"),
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrPasswordFormat,
		},
		{
			desc:        "password reset with empty token",
			data:        fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, "", strongPass, strongPass),
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "password reset with empty password",
			data:        fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, validToken, "", ""),
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "password reset with malformed data",
			data:        fmt.Sprintf(`{"token": "%s", "password": %s, "confirm_password": %s}`, validToken, strongPass, strongPass),
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:   "password reset with invalid contentype",
			data:   fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, validToken, strongPass, strongPass),
			token:  validToken,
			status: http.StatusUnsupportedMediaType,
			err:    apiutil.ErrValidation,
		},
		{
			desc:        "password reset with service error",
			data:        fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, validToken, strongPass, strongPass),
			token:       validToken,
			contentType: contentType,
			status:      http.StatusUnprocessableEntity,
			svcErr:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPut,
				url:         fmt.Sprintf("%s/password/reset", us.URL),
				contentType: tc.contentType,
				referer:     testReferer,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ResetSecret", mock.Anything, tc.authnRes, mock.Anything).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestUpdateRole(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc        string
		data        string
		userID      string
		token       string
		contentType string
		authnRes    smqauthn.Session
		authnErr    error
		status      int
		svcErr      error
		err         error
	}{
		{
			desc:        "update user role as admin with valid token",
			data:        fmt.Sprintf(`{"role": "%s"}`, "admin"),
			userID:      user.ID,
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update user role as normal user with valid token",
			data:        fmt.Sprintf(`{"role": "%s"}`, "admin"),
			userID:      user.ID,
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update user role with invalid token",
			data:        fmt.Sprintf(`{"role": "%s"}`, "admin"),
			userID:      user.ID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update user role with empty token",
			data:        fmt.Sprintf(`{"role": "%s"}`, "admin"),
			userID:      user.ID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update user with invalid role",
			data:        fmt.Sprintf(`{"role": "%s"}`, "invalid"),
			userID:      user.ID,
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrInvalidRole,
		},
		{
			desc:        "update user with invalid contentype",
			data:        fmt.Sprintf(`{"role": "%s"}`, "admin"),
			userID:      user.ID,
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update user with malformed data",
			data:        fmt.Sprintf(`{"role": %s}`, "admin"),
			userID:      user.ID,
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
		{
			desc:        "update user with service error",
			data:        fmt.Sprintf(`{"role": "%s"}`, "admin"),
			userID:      user.ID,
			token:       validToken,
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusUnprocessableEntity,
			svcErr:      svcerr.ErrUpdateEntity,
			err:         svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/users/%s/role", us.URL, tc.userID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("UpdateRole", mock.Anything, tc.authnRes, mock.Anything).Return(users.User{}, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var resBody respBody
			err = json.NewDecoder(res.Body).Decode(&resBody)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if resBody.Err != "" || resBody.Message != "" {
				err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestUpdateSecret(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc        string
		data        string
		user        users.User
		contentType string
		token       string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc: "update user secret with valid token",
			data: `{"old_secret": "strongersecret", "new_secret": "strongersecret"}`,
			user: users.User{
				ID:    user.ID,
				Email: "username",
				Credentials: users.Credentials{
					Secret: "strongersecret",
				},
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc: "update user secret with empty token",
			data: `{"old_secret": "strongersecret", "new_secret": "strongersecret"}`,
			user: users.User{
				ID:    user.ID,
				Email: "username",
				Credentials: users.Credentials{
					Secret: "strongersecret",
				},
			},
			token:       "",
			authnRes:    verifiedSession,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc: "update user secret with invalid token",
			data: `{"old_secret": "strongersecret", "new_secret": "strongersecret"}`,
			user: users.User{
				ID:    user.ID,
				Email: "username",
				Credentials: users.Credentials{
					Secret: "strongersecret",
				},
			},
			contentType: contentType,
			token:       inValid,
			authnRes:    verifiedSession,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},

		{
			desc: "update user secret with empty secret",
			data: `{"old_secret": "", "new_secret": "strongersecret"}`,
			user: users.User{
				ID:    user.ID,
				Email: "username",
				Credentials: users.Credentials{
					Secret: "",
				},
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingPass,
		},
		{
			desc: "update user secret with invalid contentype",
			data: `{"old_secret": "strongersecret", "new_secret": "strongersecret"}`,
			user: users.User{
				ID:    user.ID,
				Email: "username",
				Credentials: users.Credentials{
					Secret: "",
				},
			},
			contentType: "application/xml",
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc: "update user secret with malformed data",
			data: fmt.Sprintf(`{"secret": %s}`, "invalid"),
			user: users.User{
				ID:    user.ID,
				Email: "username",
				Credentials: users.Credentials{
					Secret: "",
				},
			},
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/users/secret", us.URL),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("UpdateSecret", mock.Anything, tc.authnRes, mock.Anything, mock.Anything).Return(tc.user, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var resBody respBody
			err = json.NewDecoder(res.Body).Decode(&resBody)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if resBody.Err != "" || resBody.Message != "" {
				err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestIssueToken(t *testing.T) {
	us, svc, _ := newUsersServer()
	defer us.Close()

	validUsername := "valid"
	dataFormat := `{"username": "%s", "password": "%s"}`

	cases := []struct {
		desc        string
		data        string
		contentType string
		status      int
		err         error
	}{
		{
			desc:        "issue token with valid identity and secret",
			data:        fmt.Sprintf(dataFormat, validUsername, secret),
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "issue token with empty identity",
			data:        fmt.Sprintf(dataFormat, "", secret),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingUsernameEmail,
		},
		{
			desc:        "issue token with empty secret",
			data:        fmt.Sprintf(dataFormat, validUsername, ""),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingPass,
		},
		{
			desc:        "issue token with invalid email",
			data:        fmt.Sprintf(dataFormat, "invalid", secret),
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "issues token with malformed data",
			data:        fmt.Sprintf(dataFormat, validUsername, secret),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
		{
			desc:        "issue token with invalid contentype",
			data:        fmt.Sprintf(dataFormat, "invalid", secret),
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/users/tokens/issue", us.URL),
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}

			svcCall := svc.On("IssueToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&grpcTokenV1.Token{AccessToken: validToken}, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			if tc.err != nil {
				var resBody respBody
				err = json.NewDecoder(res.Body).Decode(&resBody)
				assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
				if resBody.Err != "" || resBody.Message != "" {
					err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
				}
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			}
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
		})
	}
}

func TestRefreshToken(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc        string
		data        string
		contentType string
		token       string
		authnRes    smqauthn.Session
		authnErr    error
		status      int
		refreshErr  error
		err         error
	}{
		{
			desc:        "refresh token with valid token",
			data:        fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, validToken, validID),
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "refresh token with invalid token",
			data:        fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, inValidToken, validID),
			contentType: contentType,
			token:       inValidToken,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "refresh token with empty token",
			data:        fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, "", validID),
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "refresh token with invalid domain",
			data:        fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, validToken, "invalid"),
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "refresh token with malformed data",
			data:        fmt.Sprintf(`{"refresh_token": %s, "domain_id": %s}`, validToken, validID),
			contentType: contentType,
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMalformedRequestBody,
		},
		{
			desc:        "refresh token with invalid contentype",
			data:        fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, validToken, validID),
			contentType: "application/xml",
			token:       validToken,
			authnRes:    verifiedSession,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/users/tokens/refresh", us.URL),
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
				token:       tc.token,
			}
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("RefreshToken", mock.Anything, tc.authnRes, tc.token, mock.Anything).Return(&grpcTokenV1.Token{AccessToken: validToken}, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			if tc.err != nil {
				var resBody respBody
				err = json.NewDecoder(res.Body).Decode(&resBody)
				assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
				if resBody.Err != "" || resBody.Message != "" {
					err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
				}
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			}
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestEnable(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()
	cases := []struct {
		desc     string
		user     users.User
		response users.User
		token    string
		authnRes smqauthn.Session
		authnErr error
		status   int
		svcErr   error
		err      error
	}{
		{
			desc: "enable user as admin with valid token",
			user: user,
			response: users.User{
				ID:     user.ID,
				Status: users.EnabledStatus,
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc: "enable user as normal user with valid token",
			user: user,
			response: users.User{
				ID:     user.ID,
				Status: users.EnabledStatus,
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "enable user with invalid token",
			user:     user,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "enable user with empty id",
			user: users.User{
				ID: "",
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "enable user with service error",
			user:     user,
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusUnprocessableEntity,
			svcErr:   svcerr.ErrEnableUser,
			err:      svcerr.ErrEnableUser,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.user)
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/users/%s/enable", us.URL, tc.user.ID),
				contentType: contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Enable", mock.Anything, tc.authnRes, mock.Anything).Return(tc.user, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			if tc.err != nil {
				var resBody respBody
				err = json.NewDecoder(res.Body).Decode(&resBody)
				assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
				if resBody.Err != "" || resBody.Message != "" {
					err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
				}
				assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			}
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestDisable(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc     string
		user     users.User
		response users.User
		token    string
		authnRes smqauthn.Session
		authnErr error
		status   int
		svcErr   error
		err      error
	}{
		{
			desc: "disable user as admin with valid token",
			user: user,
			response: users.User{
				ID:     user.ID,
				Status: users.DisabledStatus,
			},
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, SuperAdmin: true, Verified: true},
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc: "disable user as normal user with valid token",
			user: user,
			response: users.User{
				ID:     user.ID,
				Status: users.DisabledStatus,
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "disable user with invalid token",
			user:     user,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "disable user with empty id",
			user: users.User{
				ID: "",
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "disable user with service error",
			user:     user,
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusUnprocessableEntity,
			svcErr:   svcerr.ErrDisableUser,
			err:      svcerr.ErrDisableUser,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.user)
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/users/%s/disable", us.URL, tc.user.ID),
				contentType: contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Disable", mock.Anything, mock.Anything, mock.Anything).Return(tc.user, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestDelete(t *testing.T) {
	us, svc, authn := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc     string
		user     users.User
		response users.User
		token    string
		authnRes smqauthn.Session
		authnErr error
		status   int
		svcErr   error
		err      error
	}{
		{
			desc: "delete user as admin with valid token",
			user: user,
			response: users.User{
				ID: user.ID,
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "delete user with invalid token",
			user:     user,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "delete user with empty id",
			user: users.User{
				ID: "",
			},
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusMethodNotAllowed,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "delete user with service error",
			user:     user,
			token:    validToken,
			authnRes: verifiedSession,
			status:   http.StatusUnprocessableEntity,
			svcErr:   svcerr.ErrRemoveEntity,
			err:      svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.user)
			req := testRequest{
				user:        us.Client(),
				method:      http.MethodDelete,
				url:         fmt.Sprintf("%s/users/%s", us.URL, tc.user.ID),
				contentType: contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repoCall := svc.On("Delete", mock.Anything, tc.authnRes, tc.user.ID).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			repoCall.Unset()
			authnCall.Unset()
		})
	}
}

type respBody struct {
	Err     string       `json:"error"`
	Message string       `json:"message"`
	Total   int          `json:"total"`
	ID      string       `json:"id"`
	Tags    []string     `json:"tags"`
	Role    users.Role   `json:"role"`
	Status  users.Status `json:"status"`
}
