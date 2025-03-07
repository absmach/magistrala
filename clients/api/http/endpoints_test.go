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
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/clients"
	clientsapi "github.com/absmach/supermq/clients/api/http"
	"github.com/absmach/supermq/clients/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/uuid"
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

	logger := smqlog.NewMock()
	mux := chi.NewRouter()
	idp := uuid.NewMock()
	clientsapi.MakeHandler(svc, authn, mux, logger, "", idp)

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
		authnRes    smqauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "register  a new client with a valid token",
			client:      client,
			domainID:    domainID,
			token:       validToken,
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "register an existing client",
			client:      client,
			domainID:    domainID,
			token:       validToken,
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			svcCall := svc.On("CreateClients", mock.Anything, tc.authnRes, tc.client).Return([]clients.Client{tc.client}, []roles.RoleProvision{}, tc.err)
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
		authnRes    smqauthn.Session
		authnErr    error
		err         error
		len         int
	}{
		{
			desc:        "create clients with valid token",
			client:      items,
			domainID:    domainID,
			token:       validToken,
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc:        "create clients with service error",
			client:      items,
			contentType: contentType,
			domainID:    domainID,
			token:       validToken,
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			svcCall := svc.On("CreateClients", mock.Anything, tc.authnRes, mock.Anything, mock.Anything, mock.Anything).Return(tc.client, []roles.RoleProvision{}, tc.err)
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
		authnRes            smqauthn.Session
		authnErr            error
		err                 error
	}{
		{
			desc:     "list clients as admin with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "offset=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with limit",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "limit=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with limit greater than max",
			token:    validToken,
			domainID: domainID,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with name",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "name=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate name",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "name=1&name=2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with status",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "status=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate status",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "status=enabled&status=disabled",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with tags",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "tag=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate tags",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "tag=tag1&tag=tag2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "metadata=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with permissions",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "permission=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate permissions",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "permission=view&permission=view",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list clients with list perms",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "list_perms=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list clients with duplicate list perms",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
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
			svcCall := svc.On("ListClients", mock.Anything, tc.authnRes, mock.Anything).Return(tc.listClientsResponse, tc.err)
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
		authnRes smqauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:     "view client with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
		authnRes       smqauthn.Session
		authnErr       error
		err            error
	}{
		{
			desc:        "update client with valid token",
			domainID:    domainID,
			id:          client.ID,
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingID,
		},
		{
			desc:           "update client with name that is too long",
			id:             client.ID,
			authnRes:       smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
		authnRes       smqauthn.Session
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
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
		authnRes    smqauthn.Session
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes:    smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
		authnRes smqauthn.Session
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
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
		authnRes smqauthn.Session
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
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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

func TestDeleteClient(t *testing.T) {
	ts, svc, authn := newClientsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		id       string
		domainID string
		token    string
		status   int
		authnRes smqauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:     "delete client with valid token",
			id:       client.ID,
			domainID: domainID,
			token:    validToken,
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusNoContent,

			err: nil,
		},
		{
			desc:     "delete client with invalid token",
			id:       client.ID,
			domainID: domainID,
			token:    inValidToken,
			authnRes: smqauthn.Session{},
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
			authnRes: smqauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
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

func TestSetClientParentGroupEndpoint(t *testing.T) {
	gs, svc, authn := newClientsServer()
	defer gs.Close()

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		data        string
		contentType string
		session     smqauthn.Session
		svcErr      error
		resp        clients.Client
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "set client parent group successfully",
			token:       validToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "set client parent group with invalid token",
			token:       inValidToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "set client parent group with empty token",
			token:       "",
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "set client parent group with empty domainID",
			token:       validToken,
			id:          validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "set client parent group with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "set client parent group with empty id",
			token:       validToken,
			id:          "",
			domainID:    validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "set client parent group with empty parent group id",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        `{"parent_group_id":""}`,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingParentGroupID,
		},
		{
			desc:        "set client parent group with malformed request",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"`, validID),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc:        "set client parent group with service error",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/clients/%s/parent", gs.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("SetParentGroup", mock.Anything, tc.session, validID, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveClientParentGroupEndpoint(t *testing.T) {
	gs, svc, authn := newClientsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcErr   error
		resp     clients.Client
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "remove client parent group successfully",
			token:    validToken,
			id:       validID,
			domainID: validID,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "remove client parent group with invalid token",
			token:    inValidToken,
			session:  smqauthn.Session{},
			id:       validID,
			domainID: validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:   "remove client parent group with empty token",
			token:  "",
			id:     validID,
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "remove client parent group with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "remove client parent group with empty id",
			token:    validToken,
			id:       "",
			domainID: validID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "remove client parent group with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/clients/%s/parent", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RemoveParentGroup", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
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
