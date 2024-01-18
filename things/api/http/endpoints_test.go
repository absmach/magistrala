// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0x6flab/namegenerator"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/groups"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	httpapi "github.com/absmach/magistrala/things/api/http"
	"github.com/absmach/magistrala/things/mocks"
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

func toJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func newThingsServer() (*httptest.Server, *mocks.Service) {
	gRepo := new(gmocks.Repository)
	auth := new(authmocks.Service)

	svc := new(mocks.Service)
	gsvc := groups.NewService(gRepo, idProvider, auth)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	httpapi.MakeHandler(svc, gsvc, mux, logger, "")

	return httptest.NewServer(mux), svc
}

func TestCreateThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		client      mgclients.Client
		token       string
		contentType string
		status      int
		err         error
	}{
		{
			desc:        "register  a new thing with a valid token",
			client:      client,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "register an existing thing",
			client:      client,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusConflict,
			err:         errors.ErrConflict,
		},
		{
			desc:        "register a new thing with an empty token",
			client:      client,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc: "register a thing with an  invalid ID",
			client: mgclients.Client{
				ID: inValid,
				Credentials: mgclients.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
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
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc: "register thing with invalid status",
			client: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: "newclientwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: mgclients.AllStatus,
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrInvalidStatus,
		},
		{
			desc: "create thing with invalid contentype",
			client: mgclients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: mgclients.Credentials{
					Identity: "example@example.com",
					Secret:   secret,
				},
			},

			token:       validToken,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/", ts.URL),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		svcCall := svc.On("CreateThings", mock.Anything, tc.token, tc.client).Return([]mgclients.Client{tc.client}, tc.err)
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
		desc        string
		client      []mgclients.Client
		token       string
		contentType string
		status      int
		err         error
		len         int
	}{
		{
			desc:        "create things with valid token",
			client:      items,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
			len:         3,
		},
		{
			desc:        "create things with invalid token",
			client:      items,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
			len:         0,
		},
		{
			desc:        "create things with empty token",
			client:      items,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
			len:         0,
		},
		{
			desc:        "create things with empty request",
			client:      []mgclients.Client{},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrEmptyList,
			len:         0,
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
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrInvalidIDFormat,
		},
		{
			desc: "create thing with invalid contentype",
			client: []mgclients.Client{
				{
					ID: testsutil.GenerateUUID(t),
				},
			},
			token:       validToken,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
		{
			desc: "register a thing that can't be marshalled",
			client: []mgclients.Client{
				{
					ID: testsutil.GenerateUUID(t),
					Credentials: mgclients.Credentials{
						Identity: "user@example.com",
						Secret:   "12345678",
					},
					Metadata: map[string]interface{}{
						"test": make(chan int),
					},
				},
			},
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.client)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/bulk", ts.URL),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		svcCall := svc.On("CreateThings", mock.Anything, tc.token, mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.err)
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
		svcCall.Unset()
	}
}

func TestListThings(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc               string
		query              string
		token              string
		listThingsResponse mgclients.ClientsPage
		status             int
		err                error
	}{
		{
			desc:   "list things with valid token",
			token:  validToken,
			status: http.StatusOK,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			err: nil,
		},
		{
			desc:   "list things with empty token",
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "list things with invalid token",
			token:  inValidToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:  "list things with offset",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 1,
					Total:  1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  "offset=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with invalid offset",
			token:  validToken,
			query:  "offset=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:  "list things with limit",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Limit: 1,
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  "limit=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with invalid limit",
			token:  validToken,
			query:  "limit=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list things with limit greater than max",
			token:  validToken,
			query:  fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:  "list things with owner_id",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  fmt.Sprintf("owner_id=%s", validID),
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with duplicate owner_id",
			token:  validToken,
			query:  "owner_id=1&owner_id=2",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:   "list things with invalid owner_id",
			token:  validToken,
			query:  "owner_id=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:  "list things with name",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  "name=clientname",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with invalid name",
			token:  validToken,
			query:  "name=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list things with duplicate name",
			token:  validToken,
			query:  "name=1&name=2",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list things with status",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  "status=enabled",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with invalid status",
			token:  validToken,
			query:  "status=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list things with duplicate status",
			token:  validToken,
			query:  "status=enabled&status=disabled",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list things with tags",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  "tag=tag1,tag2",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with invalid tags",
			token:  validToken,
			query:  "tag=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list things with duplicate tags",
			token:  validToken,
			query:  "tag=tag1&tag=tag2",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list things with metadata",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with invalid metadata",
			token:  validToken,
			query:  "metadata=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list things with duplicate metadata",
			token:  validToken,
			query:  "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list things with permissions",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  "permission=view",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with invalid permissions",
			token:  validToken,
			query:  "permission=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list things with duplicate permissions",
			token:  validToken,
			query:  "permission=view&permission=view",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list things with list perms",
			token: validToken,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:  "list_perms=true",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list things with invalid list perms",
			token:  validToken,
			query:  "list_perms=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list things with duplicate list perms",
			token:  validToken,
			query:  "list_perms=true&listPerms=true",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodGet,
			url:         ts.URL + "/things?" + tc.query,
			contentType: contentType,
			token:       tc.token,
		}

		svcCall := svc.On("ListClients", mock.Anything, tc.token, "", mock.Anything).Return(tc.listThingsResponse, tc.err)
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

		svcCall := svc.On("ViewClient", mock.Anything, tc.token, tc.id).Return(mgclients.Client{}, tc.err)
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

		svcCall := svc.On("ViewClientPerms", mock.Anything, tc.token, tc.thingID).Return(tc.response, tc.err)
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
		svcCall.Unset()
	}
}

func TestUpdateThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	newName := "newname"
	newTag := "newtag"
	newMetadata := mgclients.Metadata{"newkey": "newvalue"}

	cases := []struct {
		desc           string
		id             string
		data           string
		clientResponse mgclients.Client
		token          string
		contentType    string
		status         int
		err            error
	}{
		{
			desc:        "update thing with valid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			contentType: contentType,
			clientResponse: mgclients.Client{
				ID:       client.ID,
				Name:     newName,
				Tags:     []string{newTag},
				Metadata: newMetadata,
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:        "update thing with invalid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update thing with empty token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update thing with invalid id",
			id:          inValid,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			contentType: contentType,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "update thing with invalid contentype",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update thing with malformed data",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":%s}`, "invalid"),
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update thing with empty id",
			id:          " ",
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		svcCall := svc.On("UpdateClient", mock.Anything, tc.token, mock.Anything).Return(tc.clientResponse, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var resBody respBody
		err = json.NewDecoder(res.Body).Decode(&resBody)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if resBody.Err != "" || resBody.Message != "" {
			err = errors.Wrap(errors.New(resBody.Err), errors.New(resBody.Message))
		}

		if err == nil {
			assert.Equal(t, tc.clientResponse.ID, resBody.ID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.clientResponse, resBody.ID))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestUpdateThingsTags(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	newTag := "newtag"

	cases := []struct {
		desc           string
		id             string
		data           string
		contentType    string
		clientResponse mgclients.Client
		token          string
		status         int
		err            error
	}{
		{
			desc:        "update thing tags with valid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			clientResponse: mgclients.Client{
				ID:   client.ID,
				Tags: []string{newTag},
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:        "update thing tags with empty token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update thing tags with invalid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       inValidToken,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update thing tags with invalid id",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       validToken,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "update thing tags with invalid contentype",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: "application/xml",
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update things tags with empty id",
			id:          "",
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update things with malfomed data",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":[%s]}`, newTag),
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/tags", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		svcCall := svc.On("UpdateClientTags", mock.Anything, tc.token, mock.Anything).Return(tc.clientResponse, tc.err)
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
	}
}

func TestUpdateClientSecret(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		client      mgclients.Client
		contentType string
		token       string
		status      int
		err         error
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
			contentType: contentType,
			token:       validToken,
			status:      http.StatusOK,
			err:         nil,
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
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
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
			contentType: contentType,
			token:       inValid,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
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
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
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
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrBearerKey,
		},
		{
			desc: "update thing secret with invalid contentype",
			data: fmt.Sprintf(`{"secret": "%s"}`, ""),
			client: mgclients.Client{
				ID: client.ID,
				Credentials: mgclients.Credentials{
					Identity: "clientname",
					Secret:   "",
				},
			},
			contentType: "application/xml",
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
		{
			desc: "update thing secret with malformed data",
			data: fmt.Sprintf(`{"secret": %s}`, "invalid"),
			client: mgclients.Client{
				ID: client.ID,
				Credentials: mgclients.Credentials{
					Identity: "clientname",
					Secret:   "",
				},
			},
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/secret", ts.URL, tc.client.ID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		svcCall := svc.On("UpdateClientSecret", mock.Anything, tc.token, tc.client.ID, mock.Anything).Return(tc.client, tc.err)

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

		svcCall := svc.On("EnableClient", mock.Anything, tc.token, tc.client.ID).Return(tc.response, tc.err)
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
		svcCall.Unset()
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

		svcCall := svc.On("DisableClient", mock.Anything, tc.token, tc.client.ID).Return(tc.response, tc.err)
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
		svcCall.Unset()
	}
}

func TestShareThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		thingID     string
		token       string
		contentType string
		status      int
		err         error
	}{
		{
			desc:        "share thing with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "share thing with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "share thing with empty token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "share thing with empty id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     " ",
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "share thing with missing relation",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingRelation,
		},
		{
			desc:        "share thing with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [%s, "%s"]}`, "editor", "invalid", validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "share thing with empty thing id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     "",
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "share thing with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingRelation,
		},
		{
			desc:        "share thing with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [" ", " "]}`, "editor"),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "share thing with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/share", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		svcCall := svc.On("Share", mock.Anything, tc.token, tc.thingID, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestUnShareThing(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		thingID     string
		token       string
		contentType string
		status      int
		err         error
	}{
		{
			desc:        "unshare thing with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusNoContent,
			err:         nil,
		},
		{
			desc:        "unshare thing with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "unshare thing with empty token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "unshare thing with empty id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     " ",
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "unshare thing with missing relation",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingRelation,
		},
		{
			desc:        "unshare thing with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [%s, "%s"]}`, "editor", "invalid", validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with empty thing id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     "",
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingRelation,
		},
		{
			desc:        "unshare thing with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [" ", " "]}`, "editor"),
			thingID:     client.ID,
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/unshare", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		svcCall := svc.On("Unshare", mock.Anything, tc.token, tc.thingID, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
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

		svcCall := svc.On("DeleteClient", mock.Anything, tc.token, tc.id).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestListMembers(t *testing.T) {
	ts, svc := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc                string
		query               string
		token               string
		listMembersResponse mgclients.MembersPage
		status              int
		err                 error
		groupdID            string
	}{
		{
			desc:     "list members with valid token",
			token:    validToken,
			groupdID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list members with empty token",
			token:    "",
			groupdID: client.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list members with invalid token",
			token:    inValidToken,
			groupdID: client.ID,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list members with offset",
			token:    validToken,
			query:    "offset=1",
			groupdID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Offset: 1,
					Total:  1,
				},
				Members: []mgclients.Client{client},
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list members with invalid offset",
			token:    validToken,
			query:    "offset=invalid",
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with limit",
			token:    validToken,
			query:    "limit=1",
			groupdID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Limit: 1,
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list members with invalid limit",
			token:    validToken,
			query:    "limit=invalid",
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with limit greater than 100",
			token:    validToken,
			query:    fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with channel_id",
			token:    validToken,
			query:    fmt.Sprintf("channel_id=%s", validID),
			groupdID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list members with invalid channel_id",
			token:    validToken,
			query:    "channel_id=invalid",
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate channel_id",
			token:    validToken,
			query:    fmt.Sprintf("channel_id=%s&channel_id=%s", validID, validID),
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with connected set",
			token:    validToken,
			query:    "connected=true",
			groupdID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list members with invalid connected set",
			token:    validToken,
			query:    "connected=invalid",
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:   "list members with duplicate connected set",
			token:  validToken,
			query:  "connected=true&connected=false",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:     "list members with invalid group id",
			token:    validToken,
			query:    "",
			groupdID: "invalid",
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "list members with empty group id",
			token:    validToken,
			query:    "",
			groupdID: "",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:  "list members with status",
			query: fmt.Sprintf("status=%s", mgclients.EnabledStatus),
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			token:    validToken,
			groupdID: client.ID,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "list members with invalid status",
			query:    "status=invalid",
			token:    validToken,
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate status",
			query:    fmt.Sprintf("status=%s&status=%s", mgclients.EnabledStatus, mgclients.DisabledStatus),
			token:    validToken,
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:  "list members with metadata",
			token: validToken,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			groupdID: client.ID,
			query:    "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "list members with invalid metadata",
			query:    "metadata=invalid",
			groupdID: client.ID,
			token:    validToken,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate metadata",
			query:    "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			groupdID: client.ID,
			token:    validToken,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list members with permission",
			query: fmt.Sprintf("permission=%s", "read"),
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			token:    validToken,
			groupdID: client.ID,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "list members with invalid permission",
			query:    "permission=invalid",
			token:    validToken,
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate permission",
			query:    fmt.Sprintf("permission=%s&permission=%s", "read", "write"),
			token:    validToken,
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:  "list members with list permission",
			query: "list_perms=true",
			token: validToken,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			groupdID: client.ID,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "list members with invalid list permission",
			query:    "list_perms=invalid",
			token:    validToken,
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate list permission",
			query:    "list_perms=true&list_perms=false",
			token:    validToken,
			groupdID: client.ID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list members with all query params",
			query:    fmt.Sprintf("offset=1&limit=1&channel_id=%s&connected=true&status=%s&metadata=%s&permission=%s&list_perms=true", validID, mgclients.EnabledStatus, "%7B%22domain%22%3A%20%22example.com%22%7D", "read"),
			token:    validToken,
			groupdID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Offset: 1,
					Limit:  1,
					Total:  1,
				},
				Members: []mgclients.Client{client},
			},
			status: http.StatusOK,
			err:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodGet,
			url:         ts.URL + fmt.Sprintf("/channels/%s/things?", tc.groupdID) + tc.query,
			contentType: contentType,
			token:       tc.token,
		}

		svcCall := svc.On("ListClientsByGroup", mock.Anything, tc.token, mock.Anything, mock.Anything).Return(tc.listMembersResponse, tc.err)
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
