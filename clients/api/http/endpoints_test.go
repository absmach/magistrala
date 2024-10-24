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
	"github.com/absmach/magistrala/clients"
	httpapi "github.com/absmach/magistrala/clients/api/http"
	"github.com/absmach/magistrala/clients/mocks"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	secret         = "strongsecret"
	validCMetadata = clients.Metadata{"role": "client"}
	ID             = testsutil.GenerateUUID(&testing.T{})
	client         = clients.Client{
		ID:          ID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: clients.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      clients.EnabledStatus,
	}
	validToken   = "token"
	inValidToken = "invalid"
	inValid      = "invalid"
	validID      = testsutil.GenerateUUID(&testing.T{})
	domainID     = testsutil.GenerateUUID(&testing.T{})
	namesgen     = namegenerator.NewGenerator()
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

func newClientsServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	authn := new(authnmocks.Authentication)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	httpapi.MakeHandler(svc, authn, mux, logger, "")

	return httptest.NewServer(mux), svc, authn
}

func TestCreateClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		client      clients.Client
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "register  a new client with a valid token",
			client:      client,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "register an existing client",
			client:      client,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusConflict,
			err:         svcerr.ErrConflict,
		},
		{
			desc:        "register a new client with an empty token",
			client:      client,
			domainID:    domainID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc: "register a client with an  invalid ID",
			client: clients.Client{
				ID: inValid,
				Credentials: clients.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
			},
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc: "register a client that can't be marshalled",
			client: clients.Client{
				Credentials: clients.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc: "register client with invalid status",
			client: clients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: clients.Credentials{
					Identity: "newclientwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: clients.AllStatus,
			},
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrInvalidStatus,
		},
		{
			desc: "create client with invalid contentype",
			client: clients.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: clients.Credentials{
					Identity: "example@example.com",
					Secret:   secret,
				},
			},
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.client)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/", ts.URL, tc.domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("CreateClients", mock.Anything, tc.authnRes, tc.client).Return([]clients.Client{tc.client}, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestCreateClients(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	num := 3
	var items []clients.Client
	for i := 0; i < num; i++ {
		client := clients.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: namesgen.Generate(),
			Credentials: clients.Credentials{
				Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
				Secret:   secret,
			},
			Metadata: clients.Metadata{},
			Status:   clients.EnabledStatus,
		}
		items = append(items, client)
	}

	cases := []struct {
		desc        string
		client      []clients.Client
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
		len         int
	}{
		{
			desc:        "create clients with valid token",
			client:      items,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
			len:         3,
		},
		{
			desc:        "create clients with invalid token",
			client:      items,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			len:         0,
		},
		{
			desc:        "create clients with empty token",
			client:      items,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
			len:         0,
		},
		{
			desc:        "create clients with empty request",
			client:      []clients.Client{},
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
			len:         0,
		},
		{
			desc: "create clients with invalid IDs",
			client: []clients.Client{
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
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc: "create clients with invalid contentype",
			client: []clients.Client{
				{
					ID: testsutil.GenerateUUID(t),
				},
			},
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
		{
			desc: "create a client that can't be marshalled",
			client: []clients.Client{
				{
					ID: testsutil.GenerateUUID(t),
					Credentials: clients.Credentials{
						Identity: "user@example.com",
						Secret:   "12345678",
					},
					Metadata: map[string]interface{}{
						"test": make(chan int),
					},
				},
			},
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc:        "create clients with service error",
			client:      items,
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusUnprocessableEntity,
			err:         svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.client)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/bulk", ts.URL, domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("CreateClients", mock.Anything, tc.authnRes, mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestListClients(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc                string
		query               string
		domainID            string
		token               string
		listClientsResponse clients.ClientsPage
		status              int
		authnRes            mgauthn.Session
		authnErr            error
		err                 error
	}{
		{
			desc:     "list clients as admin with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			status:   http.StatusOK,
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			err: nil,
		},
		{
			desc:     "list clients as non admin with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			status:   http.StatusOK,
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			err: nil,
		},
		{
			desc:     "list clients with empty token",
			domainID: domainID,
			token:    "",
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list clients with invalid token",
			domainID: domainID,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list clients with offset",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Offset: 1,
					Total:  1,
				},
				Clients: []clients.Client{client},
			},
			query:  "offset=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list clients with invalid offset",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "offset=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with limit",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Limit: 1,
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			query:  "limit=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list clients with invalid limit",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "limit=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with limit greater than max",
			token:    validToken,
			domainID: domainID,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with name",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			query:  "name=clientname",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list clients with invalid name",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "name=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate name",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "name=1&name=2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with status",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			query:  "status=enabled",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list clients with invalid status",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "status=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate status",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "status=enabled&status=disabled",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with tags",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			query:  "tag=tag1,tag2",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list clients with invalid tags",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "tag=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate tags",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "tag=tag1&tag=tag2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			query:  "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list clients with invalid metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "metadata=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with permissions",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			query:  "permission=view",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list clients with invalid permissions",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "permission=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate permissions",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "permission=view&permission=view",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with list perms",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listClientsResponse: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
				},
				Clients: []clients.Client{client},
			},
			query:  "list_perms=true",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list clients with invalid list perms",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "list_perms=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate list perms",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "list_perms=true&listPerms=true",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodGet,
				url:         ts.URL + "/" + tc.domainID + "/clients?" + tc.query,
				contentType: contentType,
				token:       tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ListClients", mock.Anything, tc.authnRes, "", mock.Anything).Return(tc.listClientsResponse, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestViewClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		domainID string
		token    string
		id       string
		status   int
		authnRes mgauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:     "view client with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			id:       client.ID,
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "view client with invalid token",
			domainID: domainID,
			token:    inValidToken,
			id:       client.ID,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view client with empty token",
			domainID: domainID,
			token:    "",
			id:       client.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view client with invalid id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			id:       inValid,
			status:   http.StatusForbidden,

			err: svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/clients/%s", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("View", mock.Anything, tc.authnRes, tc.id).Return(clients.Client{}, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestViewClientPerms(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		domainID string
		token    string
		clientID string
		response []string
		status   int
		authnRes mgauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:     "view client permissions with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			clientID: client.ID,
			response: []string{"view", "delete", "membership"},
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "view client permissions with invalid token",
			domainID: domainID,
			token:    inValidToken,
			clientID: client.ID,
			response: []string{},
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view client permissions with empty token",
			domainID: domainID,
			token:    "",
			clientID: client.ID,
			response: []string{},
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view client permissions with invalid id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			clientID: inValid,
			response: []string{},
			status:   http.StatusForbidden,

			err: svcerr.ErrAuthorization,
		},
		{
			desc:     "view client permissions with empty id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			clientID: "",
			response: []string{},
			status:   http.StatusBadRequest,

			err: apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/clients/%s/permissions", ts.URL, tc.domainID, tc.clientID),
				token:  tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ViewPerms", mock.Anything, tc.authnRes, tc.clientID).Return(tc.response, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestUpdateClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	newName := "newname"
	newTag := "newtag"
	newMetadata := clients.Metadata{"newkey": "newvalue"}

	cases := []struct {
		desc           string
		id             string
		data           string
		clientResponse clients.Client
		domainID       string
		token          string
		contentType    string
		status         int
		authnRes       mgauthn.Session
		authnErr       error
		err            error
	}{
		{
			desc:        "update client with valid token",
			domainID:    domainID,
			id:          client.ID,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			contentType: contentType,
			clientResponse: clients.Client{
				ID:       client.ID,
				Name:     newName,
				Tags:     []string{newTag},
				Metadata: newMetadata,
			},
			status: http.StatusOK,

			err: nil,
		},
		{
			desc:        "update client with invalid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			domainID:    domainID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update client with empty token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			domainID:    domainID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update client with invalid contentype",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "update client with malformed data",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":%s}`, "invalid"),
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "update client with empty id",
			id:          " ",
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingID,
		},
		{
			desc:           "update client with name that is too long",
			id:             client.ID,
			authnRes:       mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			data:           fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, strings.Repeat("a", api.MaxNameSize+1), newTag, toJSON(newMetadata)),
			domainID:       domainID,
			token:          validToken,
			contentType:    contentType,
			clientResponse: clients.Client{},
			status:         http.StatusBadRequest,
			err:            apiutil.ErrNameSize,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/clients/%s", ts.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Update", mock.Anything, tc.authnRes, mock.Anything).Return(tc.clientResponse, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestUpdateClientsTags(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	newTag := "newtag"

	cases := []struct {
		desc           string
		id             string
		data           string
		contentType    string
		clientResponse clients.Client
		domainID       string
		token          string
		status         int
		authnRes       mgauthn.Session
		authnErr       error
		err            error
	}{
		{
			desc:        "update client tags with valid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			clientResponse: clients.Client{
				ID:   client.ID,
				Tags: []string{newTag},
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:        "update client tags with empty token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			domainID:    domainID,
			token:       "",
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update client tags with invalid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			domainID:    domainID,
			token:       inValidToken,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update client tags with invalid id",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusForbidden,

			err: svcerr.ErrAuthorization,
		},
		{
			desc:        "update client tags with invalid contentype",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: "application/xml",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update clients tags with empty id",
			id:          "",
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "update clients with malfomed data",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":[%s]}`, newTag),
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusBadRequest,

			err: errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/clients/%s/tags", ts.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("UpdateTags", mock.Anything, tc.authnRes, mock.Anything).Return(tc.clientResponse, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestUpdateClientSecret(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		client      clients.Client
		contentType string
		domainID    string
		token       string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc: "update client secret with valid token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: clients.Client{
				ID: client.ID,
				Credentials: clients.Credentials{
					Identity: "clientname",
					Secret:   "strongersecret",
				},
			},
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc: "update client secret with empty token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: clients.Client{
				ID: client.ID,
				Credentials: clients.Credentials{
					Identity: "clientname",
					Secret:   "strongersecret",
				},
			},
			contentType: contentType,
			domainID:    domainID,
			token:       "",
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc: "update client secret with invalid token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: clients.Client{
				ID: client.ID,
				Credentials: clients.Credentials{
					Identity: "clientname",
					Secret:   "strongersecret",
				},
			},
			contentType: contentType,
			domainID:    domainID,
			token:       inValid,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "update client secret with empty id",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: clients.Client{
				ID: "",
				Credentials: clients.Credentials{
					Identity: "clientname",
					Secret:   "strongersecret",
				},
			},
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc: "update client secret with empty secret",
			data: fmt.Sprintf(`{"secret": "%s"}`, ""),
			client: clients.Client{
				ID: client.ID,
				Credentials: clients.Credentials{
					Identity: "clientname",
					Secret:   "",
				},
			},
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc: "update client secret with invalid contentype",
			data: fmt.Sprintf(`{"secret": "%s"}`, ""),
			client: clients.Client{
				ID: client.ID,
				Credentials: clients.Credentials{
					Identity: "clientname",
					Secret:   "",
				},
			},
			contentType: "application/xml",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusUnsupportedMediaType,

			err: apiutil.ErrValidation,
		},
		{
			desc: "update client secret with malformed data",
			data: fmt.Sprintf(`{"secret": %s}`, "invalid"),
			client: clients.Client{
				ID: client.ID,
				Credentials: clients.Credentials{
					Identity: "clientname",
					Secret:   "",
				},
			},
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/clients/%s/secret", ts.URL, tc.domainID, tc.client.ID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("UpdateSecret", mock.Anything, tc.authnRes, tc.client.ID, mock.Anything).Return(tc.client, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestEnableClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		client   clients.Client
		response clients.Client
		domainID string
		token    string
		status   int
		authnRes mgauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:   "enable client with valid token",
			client: client,
			response: clients.Client{
				ID:     client.ID,
				Status: clients.EnabledStatus,
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "enable client with invalid token",
			client:   client,
			domainID: domainID,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "enable client with empty id",
			client: clients.Client{
				ID: "",
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.client)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/%s/enable", ts.URL, tc.domainID, tc.client.ID),
				contentType: contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Enable", mock.Anything, tc.authnRes, tc.client.ID).Return(tc.response, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestDisableClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		client   clients.Client
		response clients.Client
		domainID string
		token    string
		status   int
		authnRes mgauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:   "disable client with valid token",
			client: client,
			response: clients.Client{
				ID:     client.ID,
				Status: clients.DisabledStatus,
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "disable client with invalid token",
			client:   client,
			domainID: domainID,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "disable client with empty id",
			client: clients.Client{
				ID: "",
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.client)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/%s/disable", ts.URL, tc.domainID, tc.client.ID),
				contentType: contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Disable", mock.Anything, tc.authnRes, tc.client.ID).Return(tc.response, tc.err)
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
			authCall.Unset()
		})
	}
}

func TestShareClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		clientID    string
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "share client with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusCreated,

			err: nil,
		},
		{
			desc:        "share client with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "share client with empty token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "share client with empty id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    " ",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingID,
		},
		{
			desc:        "share client with missing relation",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingRelation,
		},
		{
			desc:        "share client with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [%s, "%s"]}`, "editor", "invalid", validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "share client with empty client id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    "",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "share client with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingRelation,
		},
		{
			desc:        "share client with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [" ", " "]}`, "editor"),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "share client with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,

			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/%s/share", ts.URL, tc.domainID, tc.clientID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Share", mock.Anything, tc.authnRes, tc.clientID, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUnShareClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		clientID    string
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "unshare client with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusNoContent,

			err: nil,
		},
		{
			desc:        "unshare client with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "unshare client with empty token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "unshare client with empty id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    " ",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingID,
		},
		{
			desc:        "unshare client with missing relation",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingRelation,
		},
		{
			desc:        "unshare client with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [%s, "%s"]}`, "editor", "invalid", validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "unshare client with empty client id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    "",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "unshare client with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingRelation,
		},
		{
			desc:        "unshare client with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [" ", " "]}`, "editor"),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "unshare client with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			clientID:    client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,

			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/%s/unshare", ts.URL, tc.domainID, tc.clientID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Unshare", mock.Anything, tc.authnRes, tc.clientID, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		id       string
		domainID string
		token    string
		status   int
		authnRes mgauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:     "delete client with valid token",
			id:       client.ID,
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusNoContent,

			err: nil,
		},
		{
			desc:     "delete client with invalid token",
			id:       client.ID,
			domainID: domainID,
			token:    inValidToken,
			authnRes: mgauthn.Session{},
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "delete client with empty token",
			id:       client.ID,
			domainID: domainID,
			token:    "",
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "delete client with empty id",
			id:       " ",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusBadRequest,

			err: apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/clients/%s", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Delete", mock.Anything, tc.authnRes, tc.id).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListMembers(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc                string
		query               string
		groupID             string
		domainID            string
		token               string
		listMembersResponse clients.MembersPage
		status              int
		authnRes            mgauthn.Session
		authnErr            error
		err                 error
	}{
		{
			desc:     "list members with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Total: 1,
				},
				Members: []clients.Client{client},
			},
			status: http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with empty token",
			domainID: domainID,
			token:    "",
			groupID:  client.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list members with invalid token",
			domainID: domainID,
			token:    inValidToken,
			groupID:  client.ID,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list members with offset",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "offset=1",
			groupID:  client.ID,
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Offset: 1,
					Total:  1,
				},
				Members: []clients.Client{client},
			},
			status: http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with invalid offset",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "offset=invalid",
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with limit",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "limit=1",
			groupID:  client.ID,
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Limit: 1,
					Total: 1,
				},
				Members: []clients.Client{client},
			},
			status: http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with invalid limit",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "limit=invalid",
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with limit greater than 100",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with channel_id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    fmt.Sprintf("channel_id=%s", validID),
			groupID:  client.ID,
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Total: 1,
				},
				Members: []clients.Client{client},
			},
			status: http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with invalid channel_id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "channel_id=invalid",
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate channel_id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    fmt.Sprintf("channel_id=%s&channel_id=%s", validID, validID),
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with connected set",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "connected=true",
			groupID:  client.ID,
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Total: 1,
				},
				Members: []clients.Client{client},
			},
			status: http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with invalid connected set",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "connected=invalid",
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate connected set",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "connected=true&connected=false",
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with empty group id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			query:    "",
			groupID:  "",
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:  "list members with status",
			query: fmt.Sprintf("status=%s", clients.EnabledStatus),
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Total: 1,
				},
				Members: []clients.Client{client},
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with invalid status",
			query:    "status=invalid",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate status",
			query:    fmt.Sprintf("status=%s&status=%s", clients.EnabledStatus, clients.DisabledStatus),
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Total: 1,
				},
				Members: []clients.Client{client},
			},
			groupID: client.ID,
			query:   "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status:  http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with invalid metadata",
			query:    "metadata=invalid",
			groupID:  client.ID,
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate metadata",
			query:    "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			groupID:  client.ID,
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusBadRequest,

			err: apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list members with permission",
			query: fmt.Sprintf("permission=%s", "view"),
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Total: 1,
				},
				Members: []clients.Client{client},
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with duplicate permission",
			query:    fmt.Sprintf("permission=%s&permission=%s", "view", "edit"),
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with list permission",
			query:    "list_perms=true",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Total: 1,
				},
				Members: []clients.Client{client},
			},
			groupID: client.ID,
			status:  http.StatusOK,

			err: nil,
		},
		{
			desc:     "list members with invalid list permission",
			query:    "list_perms=invalid",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with duplicate list permission",
			query:    "list_perms=true&list_perms=false",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			status:   http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "list members with all query params",
			query:    fmt.Sprintf("offset=1&limit=1&channel_id=%s&connected=true&status=%s&metadata=%s&permission=%s&list_perms=true", validID, clients.EnabledStatus, "%7B%22domain%22%3A%20%22example.com%22%7D", "view"),
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			listMembersResponse: clients.MembersPage{
				Page: clients.Page{
					Offset: 1,
					Limit:  1,
					Total:  1,
				},
				Members: []clients.Client{client},
			},
			status: http.StatusOK,

			err: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodGet,
				url:         ts.URL + fmt.Sprintf("/%s/channels/%s/clients?", tc.domainID, tc.groupID) + tc.query,
				contentType: contentType,
				token:       tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ListClientsByGroup", mock.Anything, tc.authnRes, mock.Anything, mock.Anything).Return(tc.listMembersResponse, tc.err)
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
			authCall.Unset()
		})
	}
}

type respBody struct {
	Err         string         `json:"error"`
	Message     string         `json:"message"`
	Total       int            `json:"total"`
	Permissions []string       `json:"permissions"`
	ID          string         `json:"id"`
	Tags        []string       `json:"tags"`
	Status      clients.Status `json:"status"`
}

type groupReqBody struct {
	Relation  string   `json:"relation"`
	UserIDs   []string `json:"user_ids"`
	GroupIDs  []string `json:"group_ids"`
	ChannelID string   `json:"channel_id"`
	ClientID  string   `json:"client_id"`
}
