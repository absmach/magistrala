// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/groups"
	gmocks "github.com/absmach/magistrala/internal/groups/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	httpapi "github.com/absmach/magistrala/users/api"
	"github.com/absmach/magistrala/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	idProvider     = uuid.New()
	secret         = "strongsecret"
	validCMetadata = mgclients.Metadata{"role": "client"}
	client         = mgclients.Client{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mgclients.Credentials{Identity: "clientidentity@example.com", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	validToken        = "valid"
	inValidToken      = "invalid"
	inValid           = "invalid"
	validContentType  = "application/json"
	validID           = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	ErrPasswordFormat = errors.New("password does not meet the requirements")
	namesgen          = namegenerator.NewNameGenerator()
)

const contentType = "application/json"

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
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

	req.Header.Set("Referer", "http://localhost")

	return tr.client.Do(req)
}

func newUsersServer() (*httptest.Server, *mocks.Service) {
	gRepo := new(gmocks.Repository)
	auth := new(authmocks.Service)

	svc := new(mocks.Service)
	gsvc := groups.NewService(gRepo, idProvider, auth)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	httpapi.MakeHandler(svc, gsvc, mux, logger, "")

	return httptest.NewServer(mux), svc
}

func toJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestRegisterClient(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc:   "register  a new user with a valid token",
			client: client,
			token:  validToken,
			status: http.StatusCreated,
			err:    nil,
		},
		{
			desc:   "register an existing user",
			client: client,
			token:  validToken,
			status: http.StatusConflict,
			err:    errors.ErrConflict,
		},
		{
			desc:   "register a new user with an empty token",
			client: client,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc: "register a user with an  invalid ID",
			client: mgclients.Client{
				ID: inValid,
				Credentials: mgclients.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
			},
			token:  validToken,
			status: http.StatusInternalServerError,
			err:    apiutil.ErrValidation,
		},
		{
			desc: "register a user that can't be marshalled",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc: "register user with invalid status",
			client: mgclients.Client{
				Credentials: mgclients.Credentials{
					Identity: "newclientwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: mgclients.AllStatus,
			},
			token:  validToken,
			status: http.StatusInternalServerError,
			err:    svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users/", us.URL),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("RegisterClient", mock.Anything, tc.token, tc.client).Return(tc.client, tc.err)
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
		repoCall.Unset()
	}
}

func TestViewClient(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		token  string
		id     string
		status int
		err    error
	}{
		{
			desc:   "view user with valid token",
			token:  validToken,
			id:     client.ID,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "view user with invalid token",
			token:  inValidToken,
			id:     client.ID,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "view user with empty token",
			token:  "",
			id:     client.ID,
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: us.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/users/%s", us.URL, tc.id),
			token:  tc.token,
		}

		repoCall := svc.On("ViewClient", mock.Anything, tc.token, tc.id).Return(mgclients.Client{}, tc.err)
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
		repoCall.Unset()
	}
}

func TestViewProfile(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		token  string
		id     string
		status int
		err    error
	}{
		{
			desc:   "view profile with valid token",
			token:  validToken,
			id:     client.ID,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "view profile with invalid token",
			token:  inValidToken,
			id:     client.ID,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "view profile with empty token",
			token:  "",
			id:     client.ID,
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: us.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/users/profile", us.URL),
			token:  tc.token,
		}

		repoCall := svc.On("ViewProfile", mock.Anything, tc.token).Return(mgclients.Client{}, tc.err)
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
		repoCall.Unset()
	}
}

func TestListClients(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	num := 3
	var items []mgclients.Client
	for i := 0; i < num; i++ {
		client := mgclients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: namesgen.Generate(),
			Credentials: mgclients.Credentials{
				Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
				Secret:   secret,
			},
			Metadata: mgclients.Metadata{},
			Status:   mgclients.EnabledStatus,
		}
		items = append(items, client)
	}

	cases := []struct {
		desc   string
		data   string
		token  string
		status int
		err    error
		len    int
	}{
		{
			desc:   "list users with valid token",
			data:   fmt.Sprintf(`{"limit": "%d"}`, 10),
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
			len:    3,
		},
		{
			desc:   "list users with empty token",
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
			len:    0,
		},
		{
			desc:   "list users with invalid token",
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
			len:    0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: us.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/users/", us.URL),
			token:  tc.token,
		}

		repoCall := svc.On("ListClients", mock.Anything, tc.token, mock.Anything, mock.Anything).Return(mgclients.ClientsPage{Clients: items}, tc.err)
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
		repoCall.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc:   "update user with valid token",
			client: client,
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "update user with invalid token",
			client: client,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "update user with invalid id",
			client: mgclients.Client{
				ID: "invalid",
			},
			token:  validToken,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
		{
			desc: "update user with invalid status",
			client: mgclients.Client{
				ID:     client.ID,
				Status: mgclients.AllStatus,
			},
			token:  validToken,
			status: http.StatusInternalServerError,
			err:    svcerr.ErrInvalidStatus,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/users/%s", us.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("UpdateClient", mock.Anything, tc.token, mock.Anything).Return(tc.client, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var resBody respBody
		err = json.NewDecoder(res.Body).Decode(&resBody)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if resBody.Err != "" || resBody.Message != "" {
			err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
		}
		if err == nil {
			assert.Equal(t, tc.client.ID, resBody.ID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.ID, resBody.ID))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc: "update user tags with valid token",
			client: mgclients.Client{
				ID:   client.ID,
				Tags: []string{"tag3", "tag"},
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc: "update user tags with invalid token",
			client: mgclients.Client{
				ID:   client.ID,
				Tags: []string{"tag3", "tag"},
			},
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "update user tags with invalid id",
			client: mgclients.Client{
				ID: "invalid",
			},
			token:  validToken,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/users/%s/tags", us.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("UpdateClientTags", mock.Anything, tc.token, mock.Anything).Return(tc.client, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var resBody respBody
		err = json.NewDecoder(res.Body).Decode(&resBody)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if resBody.Err != "" || resBody.Message != "" {
			err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
		}
		if err == nil {
			assert.Equal(t, tc.client.Tags, resBody.Tags, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Tags, resBody.Tags))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestUpdateClientIdentity(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc     string
		client   mgclients.Client
		token    string
		status   int
		err      error
		identity string
	}{
		{
			desc: "update client identity with valid token",
			client: mgclients.Client{
				ID:          client.ID,
				Credentials: mgclients.Credentials{Identity: "newidentity@example.com", Secret: secret},
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc: "update client identity with invalid token",
			client: mgclients.Client{
				ID:          client.ID,
				Credentials: mgclients.Credentials{Identity: "newidentity@example.com", Secret: secret},
			},
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "update client identity with invalid id",
			client: mgclients.Client{
				ID:          "invalid",
				Credentials: mgclients.Credentials{Identity: "newidentity@example.com", Secret: secret},
			},
			token:  validToken,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
		{
			desc: "update client identity with empty token",
			client: mgclients.Client{
				ID:          validID,
				Credentials: mgclients.Credentials{Identity: "newidentity@example.com", Secret: secret},
			},
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/users/%s/identity", us.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("UpdateClientIdentity", mock.Anything, tc.token, mock.Anything, mock.Anything).Return(mgclients.Client{}, tc.err)
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
		repoCall.Unset()
	}
}

func TestPasswordResetRequest(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	testemail := "test@example.com"
	testhost := "example.com"

	cases := []struct {
		desc   string
		data   string
		status int
		err    error
	}{
		{
			desc:   "password reset request with valid email",
			data:   fmt.Sprintf(`{"email": "%s", "host": "%s"}`, testemail, testhost),
			status: http.StatusCreated,
			err:    nil,
		},
		{
			desc:   "password reset request with empty email",
			data:   fmt.Sprintf(`{"email": "%s", "host": "%s"}`, "", testhost),
			status: http.StatusInternalServerError,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "password reset request with empty host",
			data:   fmt.Sprintf(`{"email": "%s", "host": "%s"}`, testemail, ""),
			status: http.StatusInternalServerError,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "password reset request with invalid email",
			data:   fmt.Sprintf(`{"email": "%s", "host": "%s"}`, "invalid", testhost),
			status: http.StatusNotFound,
			err:    errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users/password/reset-request", us.URL),
			contentType: validContentType,
			body:        strings.NewReader(tc.data),
		}

		repoCall := svc.On("GenerateResetToken", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestPasswordReset(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	strongPass := "StrongPassword"

	cases := []struct {
		desc   string
		data   string
		token  string
		status int
		err    error
	}{
		{
			desc:   "password reset with valid token",
			data:   fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, validToken, strongPass, strongPass),
			token:  validToken,
			status: http.StatusCreated,
			err:    nil,
		},
		{
			desc:   "password reset with invalid token",
			data:   fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, inValidToken, strongPass, strongPass),
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "password reset to weak password",
			data:   fmt.Sprintf(`{"token": "%s", "password": "%s", "confirm_password": "%s"}`, validToken, "weak", "weak"),
			token:  validToken,
			status: http.StatusInternalServerError,
			err:    ErrPasswordFormat,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/users/password/reset", us.URL),
			contentType: validContentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		repoCall := svc.On("ResetSecret", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestUpdateClientRole(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc: "update client role with valid token",
			client: mgclients.Client{
				ID:   client.ID,
				Role: mgclients.AdminRole,
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc: "update client role with invalid token",
			client: mgclients.Client{
				ID:   client.ID,
				Role: mgclients.AdminRole,
			},
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "update client role with invalid id",
			client: mgclients.Client{
				ID:   "invalid",
				Role: mgclients.AdminRole,
			},
			token:  validToken,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
		{
			desc: "update client role with empty token",
			client: mgclients.Client{
				ID:   client.ID,
				Role: mgclients.AdminRole,
			},
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/users/%s/role", us.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("UpdateClientRole", mock.Anything, tc.token, tc.client).Return(tc.client, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var resBody respBody
		err = json.NewDecoder(res.Body).Decode(&resBody)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if resBody.Err != "" || resBody.Message != "" {
			err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
		}
		if err == nil {
			assert.Equal(t, tc.client.Role, resBody.Role, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client.Role, resBody.Role))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		data   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc: "update thing secret with valid token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: mgclients.Client{
				ID: client.ID,
				Credentials: mgclients.Credentials{
					Identity: "clientname",
					Secret:   "strongersecret",
				},
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc: "update thing secret with empty token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: mgclients.Client{
				ID: client.ID,
				Credentials: mgclients.Credentials{
					Identity: "clientname",
					Secret:   "strongersecret",
				},
			},
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc: "update thing secret with invalid token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: mgclients.Client{
				ID: client.ID,
				Credentials: mgclients.Credentials{
					Identity: "clientname",
					Secret:   "strongersecret",
				},
			},
			token:  inValid,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/users/secret", us.URL),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		repoCall := svc.On("UpdateClientSecret", mock.Anything, tc.token, mock.Anything, mock.Anything).Return(tc.client, tc.err)

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
		repoCall.Unset()
	}
}

func TestIssueToken(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	validIdentity := "valid"

	cases := []struct {
		desc   string
		data   string
		status int
		err    error
	}{
		{
			desc:   "issue token with valid identity and secret",
			data:   fmt.Sprintf(`{"identity": "%s", "secret": "%s", "domainID": "%s"}`, validIdentity, secret, validID),
			status: http.StatusCreated,
			err:    nil,
		},
		{
			desc:   "issue token with empty identity",
			data:   fmt.Sprintf(`{"identity": "%s", "secret": "%s", "domainID": "%s"}`, "", secret, validID),
			status: http.StatusInternalServerError,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "issue token with empty secret",
			data:   fmt.Sprintf(`{"identity": "%s", "secret": "%s", "domainID": "%s"}`, validIdentity, "", validID),
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "issue token with empty domain",
			data:   fmt.Sprintf(`{"identity": "%s", "secret": "%s", "domainID": "%s"}`, validIdentity, secret, ""),
			status: http.StatusInternalServerError,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "issue token with invalid identity",
			data:   fmt.Sprintf(`{"identity": "%s", "secret": "%s", "domainID": "%s"}`, "invalid", secret, validID),
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users/tokens/issue", us.URL),
			contentType: contentType,
			body:        strings.NewReader(tc.data),
		}

		repoCall := svc.On("IssueToken", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&magistrala.Token{}, tc.err)
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
		repoCall.Unset()
	}
}

func TestRefreshToken(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		data   string
		status int
		err    error
	}{
		{
			desc:   "refresh token with valid token",
			data:   fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, validToken, validID),
			status: http.StatusCreated,
			err:    nil,
		},
		{
			desc:   "refresh token with invalid token",
			data:   fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, inValidToken, validID),
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "refresh token with empty token",
			data:   fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, "", validID),
			status: http.StatusUnauthorized,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "refresh token with invalid domain",
			data:   fmt.Sprintf(`{"refresh_token": "%s", "domain_id": "%s"}`, validToken, "invalid"),
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users/tokens/refresh", us.URL),
			contentType: contentType,
			body:        strings.NewReader(tc.data),
		}

		repoCall := svc.On("RefreshToken", mock.Anything, mock.Anything, mock.Anything).Return(&magistrala.Token{}, tc.err)
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
		repoCall.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc:   "enable client with valid token",
			client: client,
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "enable client with invalid token",
			client: client,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "enable client with invalid id",
			client: mgclients.Client{
				ID: "invalid",
			},
			token:  validToken,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users/%s/enable", us.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("EnableClient", mock.Anything, mock.Anything, mock.Anything).Return(mgclients.Client{}, tc.err)
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
		repoCall.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc:   "disable client with valid token",
			client: client,
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "disable client with invalid token",
			client: client,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "disable client with invalid id",
			client: mgclients.Client{
				ID: "invalid",
			},
			token:  validToken,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      us.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/users/%s/disable", us.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("DisableClient", mock.Anything, mock.Anything, mock.Anything).Return(mgclients.Client{}, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestListUsersByUserGroupId(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc    string
		token   string
		groupID string
		page    mgclients.Page
		status  int
		err     error
	}{
		{
			desc:    "list users by user group id with valid token",
			token:   validToken,
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:    "list users by user group id with invalid token",
			token:   inValidToken,
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:    "list users by user group id with empty token",
			token:   "",
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:    "list users by user group id with empty id",
			token:   validToken,
			groupID: "",
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: us.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/groups/%s/users", us.URL, tc.groupID),
			token:  tc.token,
		}

		repoCall := svc.On("ListMembers", mock.Anything, tc.token, mock.Anything, mock.Anything, mock.Anything).Return(mgclients.MembersPage{Page: tc.page}, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestListUsersByChannelID(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc    string
		token   string
		groupID string
		page    mgclients.Page
		status  int
		err     error
	}{
		{
			desc:    "list users by channel id with valid token",
			token:   validToken,
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:    "list users by channel id with invalid token",
			token:   inValidToken,
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:    "list users by channel id with empty token",
			token:   "",
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:    "list users by channel id with empty id",
			token:   validToken,
			groupID: "",
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: us.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/channels/%s/users", us.URL, validID),
			token:  tc.token,
		}

		repoCall := svc.On("ListMembers", mock.Anything, tc.token, mock.Anything, mock.Anything, mock.Anything).Return(mgclients.MembersPage{}, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestListUsersByDomainID(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc    string
		token   string
		groupID string
		page    mgclients.Page
		status  int
		err     error
	}{
		{
			desc:    "list users by domain id with valid token",
			token:   validToken,
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:    "list users by domain id with invalid token",
			token:   inValidToken,
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:    "list users by domain id with empty token",
			token:   "",
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:    "list users by domain id with empty id",
			token:   validToken,
			groupID: "",
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: us.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/domains/%s/users", us.URL, validID),
			token:  tc.token,
		}

		repoCall := svc.On("ListMembers", mock.Anything, tc.token, mock.Anything, mock.Anything, mock.Anything).Return(mgclients.MembersPage{}, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode)
		repoCall.Unset()
	}
}

func TestListUsersByThingID(t *testing.T) {
	us, svc := newUsersServer()
	defer us.Close()

	cases := []struct {
		desc    string
		token   string
		groupID string
		page    mgclients.Page
		status  int
		err     error
	}{
		{
			desc:    "list users by thing id with valid token",
			token:   validToken,
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:    "list users by thing id with invalid token",
			token:   inValidToken,
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:    "list users by thing id with empty token",
			token:   "",
			groupID: validID,
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:    "list users by thing id with empty id",
			token:   validToken,
			groupID: "",
			page: mgclients.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
			},
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: us.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s/users", us.URL, validID),
			token:  tc.token,
		}

		repoCall := svc.On("ListMembers", mock.Anything, tc.token, mock.Anything, mock.Anything, mock.Anything).Return(mgclients.MembersPage{}, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode)
		repoCall.Unset()
	}
}

type respBody struct {
	Err     string           `json:"error"`
	Message string           `json:"message"`
	Total   int              `json:"total"`
	ID      string           `json:"id"`
	Tags    []string         `json:"tags"`
	Role    mgclients.Role   `json:"role"`
	Status  mgclients.Status `json:"status"`
}
