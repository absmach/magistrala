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
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	"github.com/absmach/magistrala/things"
	httpapi "github.com/absmach/magistrala/things/api/http"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	secret         = "strongsecret"
	validCMetadata = things.Metadata{"role": "client"}
	ID             = testsutil.GenerateUUID(&testing.T{})
	client         = things.Client{
		ID:          ID,
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: things.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validCMetadata,
		Status:      things.EnabledStatus,
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

func newThingsServer() (*httptest.Server, *mocks.Service, *gmocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	gsvc := new(gmocks.Service)
	authn := new(authnmocks.Authentication)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	httpapi.MakeHandler(svc, gsvc, authn, mux, logger, "")

	return httptest.NewServer(mux), svc, gsvc, authn
}

func TestCreateThing(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		client      things.Client
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "register  a new thing with a valid token",
			client:      client,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "register an existing thing",
			client:      client,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusConflict,
			err:         svcerr.ErrConflict,
		},
		{
			desc:        "register a new thing with an empty token",
			client:      client,
			domainID:    domainID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc: "register a thing with an  invalid ID",
			client: things.Client{
				ID: inValid,
				Credentials: things.Credentials{
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
			desc: "register a thing that can't be marshalled",
			client: things.Client{
				Credentials: things.Credentials{
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
			desc: "register thing with invalid status",
			client: things.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: things.Credentials{
					Identity: "newclientwithinvalidstatus@example.com",
					Secret:   secret,
				},
				Status: things.AllStatus,
			},
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrInvalidStatus,
		},
		{
			desc: "create thing with invalid contentype",
			client: things.Client{
				ID: testsutil.GenerateUUID(t),
				Credentials: things.Credentials{
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
				url:         fmt.Sprintf("%s/%s/things/", ts.URL, tc.domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("CreateClients", mock.Anything, tc.authnRes, tc.client).Return([]things.Client{tc.client}, tc.err)
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

func TestCreateThings(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	num := 3
	var items []things.Client
	for i := 0; i < num; i++ {
		client := things.Client{
			ID:   testsutil.GenerateUUID(t),
			Name: namesgen.Generate(),
			Credentials: things.Credentials{
				Identity: fmt.Sprintf("%s@example.com", namesgen.Generate()),
				Secret:   secret,
			},
			Metadata: things.Metadata{},
			Status:   things.EnabledStatus,
		}
		items = append(items, client)
	}

	cases := []struct {
		desc        string
		client      []things.Client
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
			desc:        "create things with valid token",
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
			desc:        "create things with invalid token",
			client:      items,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
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
			client:      []things.Client{},
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
			len:         0,
		},
		{
			desc: "create things with invalid IDs",
			client: []things.Client{
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
			desc: "create things with invalid contentype",
			client: []things.Client{
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
			desc: "create a thing that can't be marshalled",
			client: []things.Client{
				{
					ID: testsutil.GenerateUUID(t),
					Credentials: things.Credentials{
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
			desc:        "create things with service error",
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
				url:         fmt.Sprintf("%s/%s/things/bulk", ts.URL, domainID),
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

func TestListThings(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc               string
		query              string
		domainID           string
		token              string
		listThingsResponse things.ClientsPage
		status             int
		authnRes           mgauthn.Session
		authnErr           error
		err                error
	}{
		{
			desc:     "list things as admin with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			status:   http.StatusOK,
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			err: nil,
		},
		{
			desc:     "list things as non admin with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			status:   http.StatusOK,
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			err: nil,
		},
		{
			desc:     "list things with empty token",
			domainID: domainID,
			token:    "",
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list things with invalid token",
			domainID: domainID,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list things with offset",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Offset: 1,
					Total:  1,
				},
				Clients: []things.Client{client},
			},
			query:  "offset=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list things with invalid offset",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "offset=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with limit",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Limit: 1,
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			query:  "limit=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list things with invalid limit",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "limit=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with limit greater than max",
			token:    validToken,
			domainID: domainID,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with name",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			query:  "name=clientname",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list things with invalid name",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "name=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with duplicate name",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "name=1&name=2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list things with status",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			query:  "status=enabled",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list things with invalid status",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "status=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with duplicate status",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "status=enabled&status=disabled",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list things with tags",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			query:  "tag=tag1,tag2",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list things with invalid tags",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "tag=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with duplicate tags",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "tag=tag1&tag=tag2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list things with metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			query:  "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list things with invalid metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "metadata=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with duplicate metadata",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list things with permissions",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			query:  "permission=view",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list things with invalid permissions",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "permission=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with duplicate permissions",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "permission=view&permission=view",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list things with list perms",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			listThingsResponse: things.ClientsPage{
				Page: things.Page{
					Total: 1,
				},
				Clients: []things.Client{client},
			},
			query:  "list_perms=true",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list things with invalid list perms",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{UserID: validID, DomainID: domainID, DomainUserID: domainID + "_" + validID, SuperAdmin: false},
			query:    "list_perms=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list things with duplicate list perms",
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
				url:         ts.URL + "/" + tc.domainID + "/things?" + tc.query,
				contentType: contentType,
				token:       tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ListClients", mock.Anything, tc.authnRes, "", mock.Anything).Return(tc.listThingsResponse, tc.err)
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

func TestViewThing(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
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
				url:    fmt.Sprintf("%s/%s/things/%s", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("View", mock.Anything, tc.authnRes, tc.id).Return(things.Client{}, tc.err)
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

func TestViewThingPerms(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		domainID string
		token    string
		thingID  string
		response []string
		status   int
		authnRes mgauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:     "view thing permissions with valid token",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			thingID:  client.ID,
			response: []string{"view", "delete", "membership"},
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "view thing permissions with invalid token",
			domainID: domainID,
			token:    inValidToken,
			thingID:  client.ID,
			response: []string{},
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view thing permissions with empty token",
			domainID: domainID,
			token:    "",
			thingID:  client.ID,
			response: []string{},
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view thing permissions with invalid id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			thingID:  inValid,
			response: []string{},
			status:   http.StatusForbidden,

			err: svcerr.ErrAuthorization,
		},
		{
			desc:     "view thing permissions with empty id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			thingID:  "",
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
				url:    fmt.Sprintf("%s/%s/things/%s/permissions", ts.URL, tc.domainID, tc.thingID),
				token:  tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ViewPerms", mock.Anything, tc.authnRes, tc.thingID).Return(tc.response, tc.err)
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

func TestUpdateThing(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	newName := "newname"
	newTag := "newtag"
	newMetadata := things.Metadata{"newkey": "newvalue"}

	cases := []struct {
		desc           string
		id             string
		data           string
		clientResponse things.Client
		domainID       string
		token          string
		contentType    string
		status         int
		authnRes       mgauthn.Session
		authnErr       error
		err            error
	}{
		{
			desc:        "update thing with valid token",
			domainID:    domainID,
			id:          client.ID,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			contentType: contentType,
			clientResponse: things.Client{
				ID:       client.ID,
				Name:     newName,
				Tags:     []string{newTag},
				Metadata: newMetadata,
			},
			status: http.StatusOK,

			err: nil,
		},
		{
			desc:        "update thing with invalid token",
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
			desc:        "update thing with empty token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			domainID:    domainID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update thing with invalid contentype",
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
			desc:        "update thing with malformed data",
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
			desc:        "update thing with empty id",
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
			desc:           "update thing with name that is too long",
			id:             client.ID,
			authnRes:       mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			data:           fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, strings.Repeat("a", api.MaxNameSize+1), newTag, toJSON(newMetadata)),
			domainID:       domainID,
			token:          validToken,
			contentType:    contentType,
			clientResponse: things.Client{},
			status:         http.StatusBadRequest,
			err:            apiutil.ErrNameSize,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/things/%s", ts.URL, tc.domainID, tc.id),
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

func TestUpdateThingsTags(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	newTag := "newtag"

	cases := []struct {
		desc           string
		id             string
		data           string
		contentType    string
		clientResponse things.Client
		domainID       string
		token          string
		status         int
		authnRes       mgauthn.Session
		authnErr       error
		err            error
	}{
		{
			desc:        "update thing tags with valid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			clientResponse: things.Client{
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
			desc:        "update thing tags with empty token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			domainID:    domainID,
			token:       "",
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update thing tags with invalid token",
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
			desc:        "update thing tags with invalid id",
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
			desc:        "update thing tags with invalid contentype",
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
			desc:        "update things tags with empty id",
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
			desc:        "update things with malfomed data",
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
				url:         fmt.Sprintf("%s/%s/things/%s/tags", ts.URL, tc.domainID, tc.id),
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
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		client      things.Client
		contentType string
		domainID    string
		token       string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc: "update thing secret with valid token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: things.Client{
				ID: client.ID,
				Credentials: things.Credentials{
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
			desc: "update thing secret with empty token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: things.Client{
				ID: client.ID,
				Credentials: things.Credentials{
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
			desc: "update thing secret with invalid token",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: things.Client{
				ID: client.ID,
				Credentials: things.Credentials{
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
			desc: "update thing secret with empty id",
			data: fmt.Sprintf(`{"secret": "%s"}`, "strongersecret"),
			client: things.Client{
				ID: "",
				Credentials: things.Credentials{
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
			desc: "update thing secret with empty secret",
			data: fmt.Sprintf(`{"secret": "%s"}`, ""),
			client: things.Client{
				ID: client.ID,
				Credentials: things.Credentials{
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
			desc: "update thing secret with invalid contentype",
			data: fmt.Sprintf(`{"secret": "%s"}`, ""),
			client: things.Client{
				ID: client.ID,
				Credentials: things.Credentials{
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
			desc: "update thing secret with malformed data",
			data: fmt.Sprintf(`{"secret": %s}`, "invalid"),
			client: things.Client{
				ID: client.ID,
				Credentials: things.Credentials{
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
				url:         fmt.Sprintf("%s/%s/things/%s/secret", ts.URL, tc.domainID, tc.client.ID),
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

func TestEnableThing(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		client   things.Client
		response things.Client
		domainID string
		token    string
		status   int
		authnRes mgauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:   "enable thing with valid token",
			client: client,
			response: things.Client{
				ID:     client.ID,
				Status: things.EnabledStatus,
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "enable thing with invalid token",
			client:   client,
			domainID: domainID,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "enable thing with empty id",
			client: things.Client{
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
				url:         fmt.Sprintf("%s/%s/things/%s/enable", ts.URL, tc.domainID, tc.client.ID),
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

func TestDisableThing(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		client   things.Client
		response things.Client
		domainID string
		token    string
		status   int
		authnRes mgauthn.Session
		authnErr error
		err      error
	}{
		{
			desc:   "disable thing with valid token",
			client: client,
			response: things.Client{
				ID:     client.ID,
				Status: things.DisabledStatus,
			},
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusOK,

			err: nil,
		},
		{
			desc:     "disable thing with invalid token",
			client:   client,
			domainID: domainID,
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "disable thing with empty id",
			client: things.Client{
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
				url:         fmt.Sprintf("%s/%s/things/%s/disable", ts.URL, tc.domainID, tc.client.ID),
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

func TestShareThing(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		thingID     string
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "share thing with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusCreated,

			err: nil,
		},
		{
			desc:        "share thing with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "share thing with empty token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "share thing with empty id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     " ",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingID,
		},
		{
			desc:        "share thing with missing relation",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingRelation,
		},
		{
			desc:        "share thing with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [%s, "%s"]}`, "editor", "invalid", validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "share thing with empty thing id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     "",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "share thing with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingRelation,
		},
		{
			desc:        "share thing with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [" ", " "]}`, "editor"),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "share thing with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
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
				url:         fmt.Sprintf("%s/%s/things/%s/share", ts.URL, tc.domainID, tc.thingID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Share", mock.Anything, tc.authnRes, tc.thingID, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUnShareThing(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		thingID     string
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "unshare thing with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusNoContent,

			err: nil,
		},
		{
			desc:        "unshare thing with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "unshare thing with empty token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "unshare thing with empty id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     " ",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingID,
		},
		{
			desc:        "unshare thing with missing relation",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingRelation,
		},
		{
			desc:        "unshare thing with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [%s, "%s"]}`, "editor", "invalid", validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with empty thing id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     "",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrMissingRelation,
		},
		{
			desc:        "unshare thing with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [" ", " "]}`, "editor"),
			thingID:     client.ID,
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
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
				url:         fmt.Sprintf("%s/%s/things/%s/unshare", ts.URL, tc.domainID, tc.thingID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("Unshare", mock.Anything, tc.authnRes, tc.thingID, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteThing(t *testing.T) {
	ts, svc, _, authn := newThingsServer()
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
			desc:     "delete thing with valid token",
			id:       client.ID,
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			status:   http.StatusNoContent,

			err: nil,
		},
		{
			desc:     "delete thing with invalid token",
			id:       client.ID,
			domainID: domainID,
			token:    inValidToken,
			authnRes: mgauthn.Session{},
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "delete thing with empty token",
			id:       client.ID,
			domainID: domainID,
			token:    "",
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "delete thing with empty id",
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
				url:    fmt.Sprintf("%s/%s/things/%s", ts.URL, tc.domainID, tc.id),
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
	ts, svc, _, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc                string
		query               string
		groupID             string
		domainID            string
		token               string
		listMembersResponse things.MembersPage
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
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Total: 1,
				},
				Members: []things.Client{client},
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
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Offset: 1,
					Total:  1,
				},
				Members: []things.Client{client},
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
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Limit: 1,
					Total: 1,
				},
				Members: []things.Client{client},
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
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Total: 1,
				},
				Members: []things.Client{client},
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
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Total: 1,
				},
				Members: []things.Client{client},
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
			query: fmt.Sprintf("status=%s", things.EnabledStatus),
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Total: 1,
				},
				Members: []things.Client{client},
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
			query:    fmt.Sprintf("status=%s&status=%s", things.EnabledStatus, things.DisabledStatus),
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
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Total: 1,
				},
				Members: []things.Client{client},
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
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Total: 1,
				},
				Members: []things.Client{client},
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
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Total: 1,
				},
				Members: []things.Client{client},
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
			query:    fmt.Sprintf("offset=1&limit=1&channel_id=%s&connected=true&status=%s&metadata=%s&permission=%s&list_perms=true", validID, things.EnabledStatus, "%7B%22domain%22%3A%20%22example.com%22%7D", "view"),
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  client.ID,
			listMembersResponse: things.MembersPage{
				Page: things.Page{
					Offset: 1,
					Limit:  1,
					Total:  1,
				},
				Members: []things.Client{client},
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
				url:         ts.URL + fmt.Sprintf("/%s/channels/%s/things?", tc.domainID, tc.groupID) + tc.query,
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

func TestAssignUsers(t *testing.T) {
	ts, _, gsvc, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		groupID     string
		reqBody     interface{}
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:     "assign users to a group successfully",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusCreated,

			err: nil,
		},
		{
			desc:     "assign users to a group with invalid token",
			domainID: domainID,
			token:    inValidToken,
			authnRes: mgauthn.Session{},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "assign users to a group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:     "assign users to a group with empty group id",
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  "",
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "assign users to a group with empty relation",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "assign users to a group with empty user ids",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "assign users to a group with invalid request body",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: map[string]interface{}{
				"relation": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: nil,
		},
		{
			desc:     "assign users to a group with invalid content type",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,

			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.reqBody)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/users/assign", ts.URL, tc.domainID, tc.groupID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.authnRes, tc.groupID, mock.Anything, "users", mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUnassignUsers(t *testing.T) {
	ts, _, gsvc, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		groupID     string
		reqBody     interface{}
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:     "unassign users from a group successfully",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusNoContent,

			err: nil,
		},
		{
			desc:     "unassign users from a group with invalid token",
			domainID: domainID,
			token:    inValidToken,
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "unassign users from a group with empty token",
			domainID: domainID,
			token:    "",
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:     "unassign users from a group with empty group id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  "",
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "unassign users from a group with empty relation",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "unassign users from a group with empty user ids",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "unassign users from a group with invalid request body",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: map[string]interface{}{
				"relation": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: nil,
		},
		{
			desc:     "unassign users from a group with invalid content type",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,

			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.reqBody)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/users/unassign", ts.URL, tc.domainID, tc.groupID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.authnRes, tc.groupID, mock.Anything, "users", mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAssignGroupsToChannel(t *testing.T) {
	ts, _, gsvc, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		groupID     string
		reqBody     interface{}
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:     "assign groups to a channel successfully",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusCreated,

			err: nil,
		},
		{
			desc:     "assign groups to a channel with invalid token",
			domainID: domainID,
			token:    inValidToken,
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "assign groups to a channel with empty token",
			domainID: domainID,
			token:    "",
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:     "assign groups to a channel with empty group id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  "",
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "assign groups to a channel with empty group ids",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "assign groups to a channel with invalid request body",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: map[string]interface{}{
				"group_ids": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "assign groups to a channel with invalid content type",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,

			err: apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.reqBody)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/groups/assign", ts.URL, tc.domainID, tc.groupID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.authnRes, tc.groupID, mock.Anything, "channels", mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUnassignGroupsFromChannel(t *testing.T) {
	ts, _, gsvc, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		groupID     string
		reqBody     interface{}
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:     "unassign groups from a channel successfully",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusNoContent,

			err: nil,
		},
		{
			desc:     "unassign groups from a channel with invalid token",
			domainID: domainID,
			token:    inValidToken,
			authnRes: mgauthn.Session{},
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "unassign groups from a channel with empty token",
			domainID: domainID,
			token:    "",
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:     "unassign groups from a channel with empty group id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  "",
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "unassign groups from a channel with empty group ids",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "unassign groups from a channel with invalid request body",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: map[string]interface{}{
				"group_ids": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "unassign groups from a channel with invalid content type",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			groupID:  validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,

			err: apiutil.ErrValidation,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.reqBody)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/groups/unassign", ts.URL, tc.domainID, tc.groupID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.authnRes, tc.groupID, mock.Anything, "channels", mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestConnectThingToChannel(t *testing.T) {
	ts, _, gsvc, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		channelID   string
		thingID     string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "connect thing to a channel successfully",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			channelID:   validID,
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "connect thing to a channel with invalid token",
			domainID:    domainID,
			token:       inValidToken,
			channelID:   validID,
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "connect thing to a channel with empty channel id",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: validID, UserID: validID, DomainID: domainID},
			channelID:   "",
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "connect thing to a channel with empty thing id",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			channelID:   validID,
			thingID:     "",
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/things/%s/connect", ts.URL, tc.domainID, tc.channelID, tc.thingID),
				token:       tc.token,
				contentType: tc.contentType,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.authnRes, tc.channelID, "group", "things", []string{tc.thingID}).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisconnectThingFromChannel(t *testing.T) {
	ts, _, gsvc, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		channelID   string
		thingID     string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:        "disconnect thing from a channel successfully",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			channelID:   validID,
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusNoContent,

			err: nil,
		},
		{
			desc:        "disconnect thing from a channel with invalid token",
			domainID:    domainID,
			token:       inValidToken,
			channelID:   validID,
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "disconnect thing from a channel with empty channel id",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			channelID:   "",
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:        "disconnect thing from a channel with empty thing id",
			domainID:    domainID,
			token:       validToken,
			authnRes:    mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			channelID:   validID,
			thingID:     "",
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/things/%s/disconnect", ts.URL, tc.domainID, tc.channelID, tc.thingID),
				token:       tc.token,
				contentType: tc.contentType,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.authnRes, tc.channelID, "group", "things", []string{tc.thingID}).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestConnect(t *testing.T) {
	ts, _, gsvc, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		reqBody     interface{}
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:     "connect thing to a channel successfully",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusCreated,

			err: nil,
		},
		{
			desc:     "connect thing to a channel with invalid token",
			domainID: domainID,
			token:    inValidToken,
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "connect thing to a channel with empty channel id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: groupReqBody{
				ChannelID: "",
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "connect thing to a channel with empty thing id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   "",
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "connect thing to a channel with invalid request body",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: map[string]interface{}{
				"channel_id": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "connect thing to a channel with invalid content type",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.reqBody)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/connect", ts.URL, tc.domainID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.authnRes, mock.Anything, "group", "things", mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisconnect(t *testing.T) {
	ts, _, gsvc, authn := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		reqBody     interface{}
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		err         error
	}{
		{
			desc:     "Disconnect thing from a channel successfully",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusNoContent,

			err: nil,
		},
		{
			desc:     "Disconnect thing from a channel with invalid token",
			domainID: domainID,
			token:    inValidToken,
			authnRes: mgauthn.Session{},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "Disconnect thing from a channel with empty channel id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: groupReqBody{
				ChannelID: "",
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusBadRequest,

			err: apiutil.ErrValidation,
		},
		{
			desc:     "Disconnect thing from a channel with empty thing id",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   "",
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:     "Disconnect thing from a channel with invalid request body",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: map[string]interface{}{
				"channel_id": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:     "Disconnect thing from a channel with invalid content type",
			domainID: domainID,
			token:    validToken,
			authnRes: mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.reqBody)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/disconnect", ts.URL, tc.domainID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.authnRes, mock.Anything, "group", "things", mock.Anything).Return(tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

type respBody struct {
	Err         string        `json:"error"`
	Message     string        `json:"message"`
	Total       int           `json:"total"`
	Permissions []string      `json:"permissions"`
	ID          string        `json:"id"`
	Tags        []string      `json:"tags"`
	Status      things.Status `json:"status"`
}

type groupReqBody struct {
	Relation  string   `json:"relation"`
	UserIDs   []string `json:"user_ids"`
	GroupIDs  []string `json:"group_ids"`
	ChannelID string   `json:"channel_id"`
	ThingID   string   `json:"thing_id"`
}
