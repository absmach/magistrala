// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/groups"

	"github.com/absmach/magistrala/auth/jwt"
	"github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	httpapi "github.com/absmach/magistrala/auth/api/http"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	idProvider     = uuid.New()
	secret         = "strongsecret"
	validCMetadata = mgclients.Metadata{"role": "client"}
	ID             = testsutil.GenerateUUID(&testing.T{})
	client         = mgclients.Client{
		ID:          ID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: mgclients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      mgclients.EnabledStatus,
	}
	validToken   = "token"
	inValidToken = "invalid"
	inValid      = "invalid"
	validID      = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	namesgen     = namegenerator.NewNameGenerator()
)

const (
	contentType     = "application/json"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
)

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

func toJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func newService() (auth.Service) {
	krepo := new(mocks.KeyRepository)
	prepo := new(mocks.PolicyAgent)
	drepo := new(mocks.DomainsRepository)
	idProvider := uuid.NewMock()

	t := jwt.New([]byte(secret))

	return auth.New(krepo, drepo, idProvider, t, prepo, loginDuration, refreshDuration, invalidDuration)
}
func newDomainsServer() (*httptest.Server, auth.Service) {
	
	svc := newService()
	logger := mglog.NewMock()
	mux := chi.NewRouter()
	httpapi.MakeHandler(svc, logger, "")

	return httptest.NewServer(mux), svc
}

func TestCreateDomain(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc:   "register  a new thing with a valid token",
			client: client,
			token:  validToken,
			status: http.StatusCreated,
			err:    nil,
		},
		{
			desc:   "register an existing thing",
			client: client,
			token:  validToken,
			status: http.StatusConflict,
			err:    errors.ErrConflict,
		},
		{
			desc:   "register a new thing with an empty token",
			client: client,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc: "register a hing with an  invalid ID",
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
			desc: "register a thing that can't be marshalled",
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
			err:    errors.ErrMalformedEntity,
		},
		{
			desc: "register thing with invalid status",
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
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/", ts.URL),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("CreateThings", mock.Anything, tc.token, tc.client).Return([]mgclients.Client{tc.client}, tc.err)
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

func TestCreateThings(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

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
		client []mgclients.Client
		token  string
		status int
		err    error
		len    int
	}{
		{
			desc:   "create things with valid token",
			client: items,
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
			len:    3,
		},
		{
			desc:   "create things with empty token",
			client: items,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
			len:    0,
		},
		{
			desc:   "create things with empty request",
			client: []mgclients.Client{},
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrEmptyList,
			len:    0,
		},
		{
			desc: "create things with invalid IDs",
			client: []mgclients.Client{
				{
					ID: inValid,
				},
				{
					ID: validID,
				},
				{
					ID: validID,
				},
			},
			token:  validToken,
			status: http.StatusInternalServerError,
			err:    apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/bulk", ts.URL),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("CreateThings", mock.Anything, tc.token, mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var bodyRes respBody
		err = json.NewDecoder(res.Body).Decode(&bodyRes)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if bodyRes.Err != "" || bodyRes.Message != "" {
			err = errors.Wrap(errors.New(bodyRes.Err), errors.New(bodyRes.Message))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.len, bodyRes.Total, fmt.Sprintf("%s: expected %d got %d", tc.desc, tc.len, bodyRes.Total))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestListThings(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

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
			desc:   "list things with valid token",
			data:   fmt.Sprintf(`{"limit": "%d"}`, 10),
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
			len:    3,
		},
		{
			desc:   "list things with empty token",
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
			len:    0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/", ts.URL),
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

func TestViewThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc   string
		token  string
		id     string
		status int
		err    error
	}{
		{
			desc:   "view client with valid token",
			token:  validToken,
			id:     client.ID,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "view client with invalid token",
			token:  inValidToken,
			id:     client.ID,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "view client with empty token",
			token:  "",
			id:     client.ID,
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
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

func TestViewThingPerms(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		token    string
		thingID  string
		response []string
		status   int
		err      error
	}{
		{
			desc:     "view thing permissions with valid token",
			token:    validToken,
			thingID:  client.ID,
			response: []string{"view", "delete", "membership"},
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "view thing permissions with invalid token",
			token:    inValidToken,
			thingID:  client.ID,
			response: []string{},
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view thing permissions with empty token",
			token:    "",
			thingID:  client.ID,
			response: []string{},
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view thing permissions with invalid id",
			token:    validToken,
			thingID:  inValid,
			response: []string{},
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s/permissions", ts.URL, tc.thingID),
			token:  tc.token,
		}

		repoCall := svc.On("ViewClientPerms", mock.Anything, tc.token, tc.thingID).Return(tc.response, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var resBody respBody
		err = json.NewDecoder(res.Body).Decode(&resBody)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if resBody.Err != "" || resBody.Message != "" {
			err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
		}
		assert.Equal(t, len(tc.response), len(resBody.Permissions), fmt.Sprintf("%s: expected %d got %d", tc.desc, len(tc.response), len(resBody.Permissions)))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestUpdateThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc:   "update thing with valid token",
			client: client,
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "update thing with invalid token",
			client: client,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "update thing with empty token",
			client: client,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc: "update thing with invalid id",
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
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s", ts.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("UpdateClient", mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var resBody respBody
		err = json.NewDecoder(res.Body).Decode(&resBody)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if resBody.Err != "" || resBody.Message != "" {
			err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
		}

		if err == nil {
			assert.Equal(t, tc.client.ID, resBody.ID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.client, resBody.ID))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestUpdateThingsTags(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc   string
		client mgclients.Client
		token  string
		status int
		err    error
	}{
		{
			desc: "update thing tags with valid token",
			client: mgclients.Client{
				ID:   client.ID,
				Tags: []string{"tag3", "tag"},
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc: "update thing tags with empty token",
			client: mgclients.Client{
				ID:   client.ID,
				Tags: []string{"tag3", "tag"},
			},
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc: "update thing tags with invalid token",
			client: mgclients.Client{
				ID:   client.ID,
				Tags: []string{"tag3", "tag"},
			},
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "update thing tags with invalid id",
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
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/tags", ts.URL, tc.client.ID),
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

func TestUpdateClientSecret(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

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
		{
			desc: "update thing secret with empty id",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: mgclients.Client{
				ID: "",
				Credentials: mgclients.Credentials{
					Identity: "clientname",
					Secret:   "strongersecret",
				},
			},
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
		{
			desc: "update thing secret with empty secret",
			data: fmt.Sprintf(`{"secret": "%s"}`, ""),
			client: mgclients.Client{
				ID: client.ID,
				Credentials: mgclients.Credentials{
					Identity: "clientname",
					Secret:   "",
				},
			},
			token:  validToken,
			status: http.StatusInternalServerError,
			err:    apiutil.ErrBearerKey,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/secret", ts.URL, tc.client.ID),
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

func TestEnableThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		client   mgclients.Client
		response mgclients.Client
		token    string
		status   int
		err      error
	}{
		{
			desc:   "enable thing with valid token",
			client: client,
			response: mgclients.Client{
				ID:     client.ID,
				Status: mgclients.EnabledStatus,
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "enable thing with invalid token",
			client: client,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "enable thing with empty id",
			client: mgclients.Client{
				ID: "",
			},
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
		{
			desc: "enable thing with invalid id",
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
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/enable", ts.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("EnableClient", mock.Anything, mock.Anything, mock.Anything).Return(tc.response, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var resBody respBody
		err = json.NewDecoder(res.Body).Decode(&resBody)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if resBody.Err != "" || resBody.Message != "" {
			err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
		}
		if err == nil {
			assert.Equal(t, tc.response.Status, resBody.Status, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.response.Status, resBody.Status))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestDisableThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		client   mgclients.Client
		response mgclients.Client
		token    string
		status   int
		err      error
	}{
		{
			desc:   "disable thing with valid token",
			client: client,
			response: mgclients.Client{
				ID:     client.ID,
				Status: mgclients.DisabledStatus,
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "disable thing with invalid token",
			client: client,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "disable thing with empty id",
			client: mgclients.Client{
				ID: "",
			},
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
		{
			desc: "disable thing with invalid id",
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
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/disable", ts.URL, tc.client.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		repoCall := svc.On("DisableClient", mock.Anything, mock.Anything, mock.Anything).Return(tc.response, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var resBody respBody
		err = json.NewDecoder(res.Body).Decode(&resBody)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if resBody.Err != "" || resBody.Message != "" {
			err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
		}
		if err == nil {
			assert.Equal(t, tc.response.Status, resBody.Status, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.response.Status, resBody.Status))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestShareThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc    string
		data    string
		thingID string
		token   string
		status  int
		err     error
	}{
		{
			desc:    "share thing with valid token",
			data:    fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID: client.ID,
			token:   validToken,
			status:  http.StatusCreated,
			err:     nil,
		},
		{
			desc:    "share thing with invalid token",
			data:    fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID: client.ID,
			token:   inValidToken,
			status:  http.StatusUnauthorized,
			err:     svcerr.ErrAuthentication,
		},
		{
			desc:    "share thing with empty token",
			data:    fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID: client.ID,
			token:   "",
			status:  http.StatusUnauthorized,
			err:     apiutil.ErrBearerToken,
		},
		{
			desc:    "share thing with empty id",
			data:    fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID: " ",
			token:   validToken,
			status:  http.StatusBadRequest,
			err:     apiutil.ErrMissingID,
		},
		{
			desc:    "share thing with missing relation",
			data:    fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID: client.ID,
			token:   validToken,
			status:  http.StatusBadRequest,
			err:     apiutil.ErrMissingRelation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/share", ts.URL, tc.thingID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		repoCall := svc.On("Share", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestUnShareThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc    string
		data    string
		thingID string
		token   string
		status  int
		err     error
	}{
		{
			desc:    "unshare thing with valid token",
			data:    fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID: client.ID,
			token:   validToken,
			status:  http.StatusNoContent,
			err:     nil,
		},
		{
			desc:    "unshare thing with invalid token",
			data:    fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID: client.ID,
			token:   inValidToken,
			status:  http.StatusUnauthorized,
			err:     svcerr.ErrAuthentication,
		},
		{
			desc:    "unshare thing with empty token",
			data:    fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID: client.ID,
			token:   "",
			status:  http.StatusUnauthorized,
			err:     apiutil.ErrBearerToken,
		},
		{
			desc:    "unshare thing with empty id",
			data:    fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID: " ",
			token:   validToken,
			status:  http.StatusBadRequest,
			err:     apiutil.ErrMissingID,
		},
		{
			desc:    "unshare thing with missing relation",
			data:    fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID: client.ID,
			token:   validToken,
			status:  http.StatusBadRequest,
			err:     apiutil.ErrMissingRelation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/unshare", ts.URL, tc.thingID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		repoCall := svc.On("Unshare", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

func TestDeleteThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
		err    error
	}{
		{
			desc:   "delete thing with valid token",
			id:     client.ID,
			token:  validToken,
			status: http.StatusNoContent,
			err:    nil,
		},
		{
			desc:   "delete thing with invalid token",
			id:     client.ID,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "delete thing with empty token",
			id:     client.ID,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "delete thing with empty id",
			id:     " ",
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
		{
			desc:   "delete thing with invalid id",
			id:     "invalid",
			token:  validToken,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			token:  tc.token,
		}

		repoCall := svc.On("DeleteClient", mock.Anything, tc.token, tc.id).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repoCall.Unset()
	}
}

type respBody struct {
	Err         string           `json:"error"`
	Message     string           `json:"message"`
	Total       int              `json:"total"`
	Permissions []string         `json:"permissions"`
	ID          string           `json:"id"`
	Tags        []string         `json:"tags"`
	Status      mgclients.Status `json:"status"`
}
