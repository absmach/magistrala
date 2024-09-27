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
	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	pauth "github.com/absmach/magistrala/pkg/auth"
	authmocks "github.com/absmach/magistrala/pkg/auth/mocks"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	gmocks "github.com/absmach/magistrala/pkg/groups/mocks"
	"github.com/absmach/magistrala/pkg/policies"
	httpapi "github.com/absmach/magistrala/things/api/http"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
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
	validID      = testsutil.GenerateUUID(&testing.T{})
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

func newThingsServer() (*httptest.Server, *mocks.Service, *gmocks.Service, *authmocks.AuthClient) {
	svc := new(mocks.Service)
	gsvc := new(gmocks.Service)
	auth := new(authmocks.AuthClient)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	httpapi.MakeHandler(svc, gsvc, auth, mux, logger, "")

	return httptest.NewServer(mux), svc, gsvc, auth
}

func TestCreateThing(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc         string
		client       mgclients.Client
		token        string
		session      pauth.Session
		contentType  string
		status       int
		identifyRes  *magistrala.IdentityRes
		authorizeRes *magistrala.AuthorizeRes
		identifyErr  error
		authorizeErr error
		err          error
	}{
		{
			desc:         "register  a new thing with a valid token",
			client:       client,
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  contentType,
			status:       http.StatusCreated,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc:         "register an existing thing",
			client:       client,
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  contentType,
			status:       http.StatusConflict,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          svcerr.ErrConflict,
		},
		{
			desc:        "register a new thing with an empty token",
			client:      client,
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
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
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  contentType,
			status:       http.StatusBadRequest,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          apiutil.ErrValidation,
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
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  contentType,
			status:       http.StatusBadRequest,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          errors.ErrMalformedEntity,
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
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  contentType,
			status:       http.StatusBadRequest,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          svcerr.ErrInvalidStatus,
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
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  "application/xml",
			status:       http.StatusUnsupportedMediaType,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          apiutil.ErrValidation,
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		authCall1 := auth.On("Authorize", mock.Anything, &magistrala.AuthorizeReq{
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Subject:     tc.session.DomainUserID,
			Permission:  policies.CreatePermission,
			ObjectType:  policies.DomainType,
			Object:      tc.session.DomainID,
		}).Return(tc.authorizeRes, tc.authorizeErr)
		svcCall := svc.On("CreateThings", mock.Anything, tc.session, tc.client).Return([]mgclients.Client{tc.client}, tc.err)
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
		authCall1.Unset()
	}
}

func TestCreateThings(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
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
		desc         string
		client       []mgclients.Client
		token        string
		session      pauth.Session
		contentType  string
		status       int
		identifyRes  *magistrala.IdentityRes
		authorizeRes *magistrala.AuthorizeRes
		identifyErr  error
		authorizeErr error
		err          error
		len          int
	}{
		{
			desc:         "create things with valid token",
			client:       items,
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  contentType,
			status:       http.StatusOK,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			len:          3,
		},
		{
			desc:        "create things with invalid token",
			client:      items,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
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
			desc:         "create things with empty request",
			client:       []mgclients.Client{},
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  contentType,
			status:       http.StatusBadRequest,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          apiutil.ErrValidation,
			len:          0,
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
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  contentType,
			status:       http.StatusBadRequest,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          apiutil.ErrValidation,
		},
		{
			desc: "create thing with invalid contentype",
			client: []mgclients.Client{
				{
					ID: testsutil.GenerateUUID(t),
				},
			},
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType:  "application/xml",
			status:       http.StatusUnsupportedMediaType,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          apiutil.ErrValidation,
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
			contentType:  contentType,
			token:        validToken,
			session:      pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:       http.StatusBadRequest,
			identifyRes:  &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			authorizeRes: &magistrala.AuthorizeRes{Authorized: true},
			err:          errors.ErrMalformedEntity,
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		authCall1 := auth.On("Authorize", mock.Anything, &magistrala.AuthorizeReq{
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Subject:     tc.session.DomainUserID,
			Permission:  policies.CreatePermission,
			ObjectType:  policies.DomainType,
			Object:      tc.session.DomainID,
		}).Return(tc.authorizeRes, tc.authorizeErr)
		svcCall := svc.On("CreateThings", mock.Anything, tc.session, mock.Anything, mock.Anything, mock.Anything).Return(tc.client, tc.err)
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
		authCall1.Unset()
	}
}

func TestListThings(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc               string
		query              string
		token              string
		session            pauth.Session
		listThingsResponse mgclients.ClientsPage
		status             int
		identifyRes        *magistrala.IdentityRes
		identifyErr        error
		err                error
	}{
		{
			desc:    "list things as admin with valid token",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			status:  http.StatusOK,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "list things as non admin with valid token",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			status:  http.StatusOK,
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:   "list things with empty token",
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:        "list things with invalid token",
			token:       inValidToken,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "list things with offset",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Offset: 1,
					Total:  1,
				},
				Clients: []mgclients.Client{client},
			},
			query:       "offset=1",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list things with invalid offset",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "offset=invalid",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "list things with limit",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Limit: 1,
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:       "limit=1",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list things with invalid limit",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "limit=invalid",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list things with limit greater than max",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "list things with name",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:       "name=clientname",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list things with invalid name",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "name=invalid",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list things with duplicate name",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "name=1&name=2",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "list things with status",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:       "status=enabled",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list things with invalid status",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "status=invalid",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list things with duplicate status",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "status=enabled&status=disabled",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "list things with tags",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:       "tag=tag1,tag2",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list things with invalid tags",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "tag=invalid",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list things with duplicate tags",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "tag=tag1&tag=tag2",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "list things with metadata",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:       "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list things with invalid metadata",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "metadata=invalid",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list things with duplicate metadata",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "list things with permissions",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:       "permission=view",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list things with invalid permissions",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "permission=invalid",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list things with duplicate permissions",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "permission=view&permission=view",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "list things with list perms",
			token:   validToken,
			session: pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			listThingsResponse: mgclients.ClientsPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Clients: []mgclients.Client{client},
			},
			query:       "list_perms=true",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list things with invalid list perms",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "list_perms=invalid",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list things with duplicate list perms",
			token:       validToken,
			session:     pauth.Session{UserID: validID, DomainID: validID, DomainUserID: validID, SuperAdmin: false},
			query:       "list_perms=true&listPerms=true",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrInvalidQueryParams,
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("ListClients", mock.Anything, tc.session, "", mock.Anything).Return(tc.listThingsResponse, tc.err)
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
	}
}

func TestViewThing(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		id          string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:        "view client with valid token",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			id:          client.ID,
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "view client with invalid token",
			token:       inValidToken,
			id:          client.ID,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("ViewClient", mock.Anything, tc.session, tc.id).Return(mgclients.Client{}, tc.err)
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
	}
}

func TestViewThingPerms(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		thingID     string
		response    []string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:        "view thing permissions with valid token",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			thingID:     client.ID,
			response:    []string{"view", "delete", "membership"},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "view thing permissions with invalid token",
			token:       inValidToken,
			thingID:     client.ID,
			response:    []string{},
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
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
			desc:        "view thing permissions with invalid id",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			thingID:     inValid,
			response:    []string{},
			status:      http.StatusForbidden,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s/permissions", ts.URL, tc.thingID),
			token:  tc.token,
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("ViewClientPerms", mock.Anything, tc.session, tc.thingID).Return(tc.response, tc.err)
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
	}
}

func TestUpdateThing(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
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
		session        pauth.Session
		contentType    string
		status         int
		identifyRes    *magistrala.IdentityRes
		identifyErr    error
		err            error
	}{
		{
			desc:        "update thing with valid token",
			id:          client.ID,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			contentType: contentType,
			clientResponse: mgclients.Client{
				ID:       client.ID,
				Name:     newName,
				Tags:     []string{newTag},
				Metadata: newMetadata,
			},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "update thing with invalid token",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
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
			desc:        "update thing with invalid contentype",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update thing with malformed data",
			id:          client.ID,
			data:        fmt.Sprintf(`{"name":%s}`, "invalid"),
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update thing with empty id",
			id:          " ",
			data:        fmt.Sprintf(`{"name":"%s","tags":["%s"],"metadata":%s}`, newName, newTag, toJSON(newMetadata)),
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("UpdateClient", mock.Anything, tc.session, mock.Anything).Return(tc.clientResponse, tc.err)
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
	}
}

func TestUpdateThingsTags(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	newTag := "newtag"

	cases := []struct {
		desc           string
		id             string
		data           string
		contentType    string
		clientResponse mgclients.Client
		token          string
		session        pauth.Session
		status         int
		identifyRes    *magistrala.IdentityRes
		identifyErr    error
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
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
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
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update thing tags with invalid id",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusForbidden,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "update thing tags with invalid contentype",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: "application/xml",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update things tags with empty id",
			id:          "",
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update things with malfomed data",
			id:          client.ID,
			data:        fmt.Sprintf(`{"tags":[%s]}`, newTag),
			contentType: contentType,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("UpdateClientTags", mock.Anything, tc.session, mock.Anything).Return(tc.clientResponse, tc.err)
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
	}
}

func TestUpdateClientSecret(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		client      mgclients.Client
		contentType string
		token       string
		session     pauth.Session
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
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
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
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
			identifyErr: svcerr.ErrAuthentication,
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
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
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
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
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
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
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
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("UpdateClientSecret", mock.Anything, tc.session, tc.client.ID, mock.Anything).Return(tc.client, tc.err)
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
	}
}

func TestEnableThing(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		client      mgclients.Client
		response    mgclients.Client
		token       string
		session     pauth.Session
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:   "enable thing with valid token",
			client: client,
			response: mgclients.Client{
				ID:     client.ID,
				Status: mgclients.EnabledStatus,
			},
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "enable thing with invalid token",
			client:      client,
			token:       inValidToken,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "enable thing with empty id",
			client: mgclients.Client{
				ID: "",
			},
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("EnableClient", mock.Anything, tc.session, tc.client.ID).Return(tc.response, tc.err)
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
	}
}

func TestDisableThing(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		client      mgclients.Client
		response    mgclients.Client
		token       string
		session     pauth.Session
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:   "disable thing with valid token",
			client: client,
			response: mgclients.Client{
				ID:     client.ID,
				Status: mgclients.DisabledStatus,
			},
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "disable thing with invalid token",
			client:      client,
			token:       inValidToken,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "disable thing with empty id",
			client: mgclients.Client{
				ID: "",
			},
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("DisableClient", mock.Anything, tc.session, tc.client.ID).Return(tc.response, tc.err)
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
	}
}

func TestShareThing(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		thingID     string
		token       string
		session     pauth.Session
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:        "share thing with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusCreated,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "share thing with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
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
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "share thing with missing relation",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrMissingRelation,
		},
		{
			desc:        "share thing with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [%s, "%s"]}`, "editor", "invalid", validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "share thing with empty thing id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     "",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "share thing with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrMissingRelation,
		},
		{
			desc:        "share thing with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [" ", " "]}`, "editor"),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "share thing with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("Share", mock.Anything, tc.session, tc.thingID, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestUnShareThing(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		data        string
		thingID     string
		token       string
		session     pauth.Session
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:        "unshare thing with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusNoContent,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "unshare thing with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
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
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "unshare thing with missing relation",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrMissingRelation,
		},
		{
			desc:        "unshare thing with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [%s, "%s"]}`, "editor", "invalid", validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with empty thing id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     "",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, " ", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrMissingRelation,
		},
		{
			desc:        "unshare thing with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : [" ", " "]}`, "editor"),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "unshare thing with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			thingID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
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

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("Unshare", mock.Anything, tc.session, tc.thingID, mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestDeleteThing(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		id          string
		token       string
		session     pauth.Session
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:        "delete thing with valid token",
			id:          client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusNoContent,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "delete thing with invalid token",
			id:          client.ID,
			token:       inValidToken,
			session:     pauth.Session{},
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:   "delete thing with empty token",
			id:     client.ID,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:        "delete thing with empty id",
			id:          " ",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			token:  tc.token,
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("DeleteClient", mock.Anything, tc.session, tc.id).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestListMembers(t *testing.T) {
	ts, svc, _, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc                string
		query               string
		groupID             string
		token               string
		session             pauth.Session
		listMembersResponse mgclients.MembersPage
		status              int
		identifyRes         *magistrala.IdentityRes
		identifyErr         error
		err                 error
	}{
		{
			desc:    "list members with valid token",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "list members with empty token",
			token:   "",
			groupID: client.ID,
			status:  http.StatusUnauthorized,
			err:     apiutil.ErrBearerToken,
		},
		{
			desc:        "list members with invalid token",
			token:       inValidToken,
			groupID:     client.ID,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "list members with offset",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:   "offset=1",
			groupID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Offset: 1,
					Total:  1,
				},
				Members: []mgclients.Client{client},
			},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list members with invalid offset",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:       "offset=invalid",
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "list members with limit",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:   "limit=1",
			groupID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Limit: 1,
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list members with invalid limit",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:       "limit=invalid",
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list members with limit greater than 100",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:       fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "list members with channel_id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:   fmt.Sprintf("channel_id=%s", validID),
			groupID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list members with invalid channel_id",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:       "channel_id=invalid",
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list members with duplicate channel_id",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:       fmt.Sprintf("channel_id=%s&channel_id=%s", validID, validID),
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "list members with connected set",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:   "connected=true",
			groupID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list members with invalid connected set",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:       "connected=invalid",
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list members with duplicate connected set",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:       "connected=true&connected=false",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list members with empty group id",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			query:       "",
			groupID:     "",
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
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
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID:     client.ID,
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list members with invalid status",
			query:       "status=invalid",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list members with duplicate status",
			query:       fmt.Sprintf("status=%s&status=%s", mgclients.EnabledStatus, mgclients.DisabledStatus),
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "list members with metadata",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			groupID:     client.ID,
			query:       "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list members with invalid metadata",
			query:       "metadata=invalid",
			groupID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list members with duplicate metadata",
			query:       "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			groupID:     client.ID,
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list members with permission",
			query: fmt.Sprintf("permission=%s", "view"),
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID:     client.ID,
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list members with duplicate permission",
			query:       fmt.Sprintf("permission=%s&permission=%s", "view", "edit"),
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "list members with list permission",
			query:   "list_perms=true",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Total: 1,
				},
				Members: []mgclients.Client{client},
			},
			groupID:     client.ID,
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "list members with invalid list permission",
			query:       "list_perms=invalid",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "list members with duplicate list permission",
			query:       "list_perms=true&list_perms=false",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID:     client.ID,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "list members with all query params",
			query:   fmt.Sprintf("offset=1&limit=1&channel_id=%s&connected=true&status=%s&metadata=%s&permission=%s&list_perms=true", validID, mgclients.EnabledStatus, "%7B%22domain%22%3A%20%22example.com%22%7D", "view"),
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: client.ID,
			listMembersResponse: mgclients.MembersPage{
				Page: mgclients.Page{
					Offset: 1,
					Limit:  1,
					Total:  1,
				},
				Members: []mgclients.Client{client},
			},
			status:      http.StatusOK,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodGet,
			url:         ts.URL + fmt.Sprintf("/channels/%s/things?", tc.groupID) + tc.query,
			contentType: contentType,
			token:       tc.token,
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := svc.On("ListClientsByGroup", mock.Anything, tc.session, mock.Anything, mock.Anything).Return(tc.listMembersResponse, tc.err)
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
	}
}

func TestAssignUsers(t *testing.T) {
	ts, _, gsvc, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		groupID     string
		reqBody     interface{}
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:    "assign users to a group successfully",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusCreated,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "assign users to a group with invalid token",
			token:   inValidToken,
			session: pauth.Session{},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "assign users to a group with empty token",
			token:   "",
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:    "assign users to a group with empty group id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: "",
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "assign users to a group with empty relation",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "assign users to a group with empty user ids",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "assign users to a group with invalid request body",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: map[string]interface{}{
				"relation": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "assign users to a group with invalid content type",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.reqBody)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels/%s/users/assign", ts.URL, tc.groupID),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(data),
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := gsvc.On("Assign", mock.Anything, tc.session, tc.groupID, mock.Anything, "users", mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestUnassignUsers(t *testing.T) {
	ts, _, gsvc, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		groupID     string
		reqBody     interface{}
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:    "unassign users from a group successfully",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusNoContent,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "unassign users from a group with invalid token",
			token:   inValidToken,
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "unassign users from a group with empty token",
			token:   "",
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:    "unassign users from a group with empty group id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: "",
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "unassign users from a group with empty relation",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "unassign users from a group with empty user ids",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "unassign users from a group with invalid request body",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: map[string]interface{}{
				"relation": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "unassign users from a group with invalid content type",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				Relation: "member",
				UserIDs:  []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.reqBody)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels/%s/users/unassign", ts.URL, tc.groupID),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(data),
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := gsvc.On("Unassign", mock.Anything, tc.session, tc.groupID, mock.Anything, "users", mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestAssignGroupsToChannel(t *testing.T) {
	ts, _, gsvc, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		groupID     string
		reqBody     interface{}
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:    "assign groups to a channel successfully",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusCreated,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "assign groups to a channel with invalid token",
			token:   inValidToken,
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "assign groups to a channel with empty token",
			token:   "",
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:    "assign groups to a channel with empty group id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: "",
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "assign groups to a channel with empty group ids",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "assign groups to a channel with invalid request body",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: map[string]interface{}{
				"group_ids": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "assign groups to a channel with invalid content type",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.reqBody)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels/%s/groups/assign", ts.URL, tc.groupID),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(data),
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := gsvc.On("Assign", mock.Anything, tc.session, tc.groupID, mock.Anything, "channels", mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestUnassignGroupsFromChannel(t *testing.T) {
	ts, _, gsvc, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		groupID     string
		reqBody     interface{}
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:    "unassign groups from a channel successfully",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusNoContent,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "unassign groups from a channel with invalid token",
			token:   inValidToken,
			session: pauth.Session{},
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "unassign groups from a channel with empty token",
			token:   "",
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:    "unassign groups from a channel with empty group id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: "",
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "unassign groups from a channel with empty group ids",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "unassign groups from a channel with invalid request body",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: map[string]interface{}{
				"group_ids": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "unassign groups from a channel with invalid content type",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			groupID: validID,
			reqBody: groupReqBody{
				GroupIDs: []string{testsutil.GenerateUUID(t), testsutil.GenerateUUID(t)},
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
	}
	for _, tc := range cases {
		data := toJSON(tc.reqBody)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels/%s/groups/unassign", ts.URL, tc.groupID),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(data),
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := gsvc.On("Unassign", mock.Anything, tc.session, tc.groupID, mock.Anything, "channels", mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestConnectThingToChannel(t *testing.T) {
	ts, _, gsvc, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		channelID   string
		thingID     string
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:        "connect thing to a channel successfully",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			channelID:   validID,
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusCreated,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "connect thing to a channel with invalid token",
			token:       inValidToken,
			channelID:   validID,
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "connect thing to a channel with empty channel id",
			token:       validToken,
			channelID:   "",
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "connect thing to a channel with empty thing id",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			channelID:   validID,
			thingID:     "",
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels/%s/things/%s/connect", ts.URL, tc.channelID, tc.thingID),
			token:       tc.token,
			contentType: tc.contentType,
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := gsvc.On("Assign", mock.Anything, tc.session, tc.channelID, "group", "things", []string{tc.thingID}).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestDisconnectThingFromChannel(t *testing.T) {
	ts, _, gsvc, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		channelID   string
		thingID     string
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:        "disconnect thing from a channel successfully",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			channelID:   validID,
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusNoContent,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:        "disconnect thing from a channel with invalid token",
			token:       inValidToken,
			channelID:   validID,
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "disconnect thing from a channel with empty channel id",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			channelID:   "",
			thingID:     validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "disconnect thing from a channel with empty thing id",
			token:       validToken,
			session:     pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			channelID:   validID,
			thingID:     "",
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels/%s/things/%s/disconnect", ts.URL, tc.channelID, tc.thingID),
			token:       tc.token,
			contentType: tc.contentType,
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := gsvc.On("Unassign", mock.Anything, tc.session, tc.channelID, "group", "things", []string{tc.thingID}).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestConnect(t *testing.T) {
	ts, _, gsvc, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		reqBody     interface{}
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:    "connect thing to a channel successfully",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusCreated,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:  "connect thing to a channel with invalid token",
			token: inValidToken,
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "connect thing to a channel with empty channel id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: groupReqBody{
				ChannelID: "",
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "connect thing to a channel with empty thing id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   "",
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "connect thing to a channel with invalid request body",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: map[string]interface{}{
				"channel_id": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "connect thing to a channel with invalid content type",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.reqBody)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/connect", ts.URL),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(data),
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := gsvc.On("Assign", mock.Anything, tc.session, mock.Anything, "group", "things", mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
	}
}

func TestDisconnect(t *testing.T) {
	ts, _, gsvc, auth := newThingsServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		token       string
		session     pauth.Session
		reqBody     interface{}
		contentType string
		status      int
		identifyRes *magistrala.IdentityRes
		identifyErr error
		err         error
	}{
		{
			desc:    "Disconnect thing from a channel successfully",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusNoContent,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         nil,
		},
		{
			desc:    "Disconnect thing from a channel with invalid token",
			token:   inValidToken,
			session: pauth.Session{},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "Disconnect thing from a channel with empty channel id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: groupReqBody{
				ChannelID: "",
				ThingID:   validID,
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "Disconnect thing from a channel with empty thing id",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   "",
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "Disconnect thing from a channel with invalid request body",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: map[string]interface{}{
				"channel_id": make(chan int),
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
		{
			desc:    "Disconnect thing from a channel with invalid content type",
			token:   validToken,
			session: pauth.Session{DomainUserID: validID, UserID: validID, DomainID: validID},
			reqBody: groupReqBody{
				ChannelID: validID,
				ThingID:   validID,
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			identifyRes: &magistrala.IdentityRes{Id: validID, UserId: validID, DomainId: validID},
			err:         apiutil.ErrValidation,
		},
	}
	for _, tc := range cases {
		data := toJSON(tc.reqBody)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/disconnect", ts.URL),
			token:       tc.token,
			contentType: tc.contentType,
			body:        strings.NewReader(data),
		}

		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(tc.identifyRes, tc.identifyErr)
		svcCall := gsvc.On("Unassign", mock.Anything, tc.session, mock.Anything, "group", "things", mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
		authCall.Unset()
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

type groupReqBody struct {
	Relation  string   `json:"relation"`
	UserIDs   []string `json:"user_ids"`
	GroupIDs  []string `json:"group_ids"`
	ChannelID string   `json:"channel_id"`
	ThingID   string   `json:"thing_id"`
}
