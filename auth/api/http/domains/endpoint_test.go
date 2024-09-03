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

	"github.com/absmach/magistrala/auth"
	httpapi "github.com/absmach/magistrala/auth/api/http/domains"
	"github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validCMetadata = mgclients.Metadata{"role": "client"}
	ID             = testsutil.GenerateUUID(&testing.T{})
	domain         = auth.Domain{
		ID:       ID,
		Name:     "domainname",
		Tags:     []string{"tag1", "tag2"},
		Metadata: validCMetadata,
		Status:   auth.EnabledStatus,
		Alias:    "mydomain",
	}
	validToken   = "token"
	inValidToken = "invalid"
	validID      = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"

	id = "testID"
)

const (
	contentType     = "application/json"
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

func newDomainsServer() (*httptest.Server, *mocks.Service) {
	logger := mglog.NewMock()
	mux := chi.NewRouter()
	svc := new(mocks.Service)
	httpapi.MakeHandler(svc, mux, logger)
	return httptest.NewServer(mux), svc
}

func TestCreateDomain(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc        string
		domain      auth.Domain
		token       string
		contentType string
		svcErr      error
		status      int
		err         error
	}{
		{
			desc: "register  a new domain successfully",
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc: "register  a new domain with empty token",
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			token:       "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc: "register  a new domain with invalid token",
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			svcErr:      svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "register  a new domain with an empty name",
			domain: auth.Domain{
				ID:       ID,
				Name:     "",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingName,
		},
		{
			desc: "register a new domain with an empty alias",
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "",
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingAlias,
		},
		{
			desc: "register a  new domain with invalid content type",
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			token:       validToken,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc: "register a  new domain that cant be marshalled",
			domain: auth.Domain{
				ID:   ID,
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
				Tags:  []string{"tag1", "tag2"},
				Alias: "test",
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.domain)
		req := testRequest{
			client:      ds.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/domains", ds.URL),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}

		svcCall := svc.On("CreateDomain", mock.Anything, mock.Anything, mock.Anything).Return(auth.Domain{}, tc.svcErr)
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

func TestListDomains(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc               string
		token              string
		query              string
		listDomainsRequest auth.DomainsPage
		status             int
		svcErr             error
		err                error
	}{
		{
			desc:   "list domains with valid token",
			token:  validToken,
			status: http.StatusOK,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			err: nil,
		},
		{
			desc:   "list domains  with empty token",
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "list domains  with invalid token",
			token:  inValidToken,
			status: http.StatusUnauthorized,
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:  "list domains  with offset",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "offset=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains  with invalid offset",
			token:  validToken,
			query:  "offset=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:  "list domains  with limit",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "limit=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains  with invalid limit",
			token:  validToken,
			query:  "limit=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:  "list domains  with name",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "name=domainname",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains with empty name",
			token:  validToken,
			query:  "name= ",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list domains with duplicate name",
			token:  validToken,
			query:  "name=1&name=2",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list domains with status",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "status=enabled",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains with invalid status",
			token:  validToken,
			query:  "status=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list domains  with duplicate status",
			token:  validToken,
			query:  "status=enabled&status=disabled",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list domains  with tags",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "tag=tag1,tag2",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains  with empty tags",
			token:  validToken,
			query:  "tag= ",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list domains  with duplicate tags",
			token:  validToken,
			query:  "tag=tag1&tag=tag2",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list domains  with metadata",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains  with invalid metadata",
			token:  validToken,
			query:  "metadata=invalid",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list domains  with duplicate metadata",
			token:  validToken,
			query:  "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list domains  with permissions",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "permission=view",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains  with invalid permissions",
			token:  validToken,
			query:  "permission= ",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list domains  with duplicate permissions",
			token:  validToken,
			query:  "permission=view&permission=view",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list domains  with order",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "order=name",
			status: http.StatusOK,
		},
		{
			desc:   "list domains  with invalid order",
			token:  validToken,
			query:  "order= ",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list domains  with duplicate order",
			token:  validToken,
			query:  "order=name&order=name",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
		{
			desc:  "list domains  with dir",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "dir=asc",
			status: http.StatusOK,
		},
		{
			desc:   "list domains  with invalid dir",
			token:  validToken,
			query:  "dir= ",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list domains  with duplicate dir",
			token:  validToken,
			query:  "dir=asc&dir=asc",
			status: http.StatusBadRequest,
			err:    apiutil.ErrInvalidQueryParams,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ds.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/domains?", ds.URL) + tc.query,
			token:  tc.token,
		}

		svcCall := svc.On("ListDomains", mock.Anything, mock.Anything, mock.Anything).Return(tc.listDomainsRequest, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestViewDomain(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc     string
		token    string
		domainID string
		status   int
		svcErr   error
		err      error
	}{
		{
			desc:     "view domain successfully",
			token:    validToken,
			domainID: id,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "view domain with empty token",
			token:    "",
			domainID: id,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view domain with invalid token",
			token:    inValidToken,
			domainID: id,
			status:   http.StatusUnauthorized,
			svcErr:   svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ds.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/domains/%s", ds.URL, tc.domainID),
			token:  tc.token,
		}

		svcCall := svc.On("RetrieveDomain", mock.Anything, mock.Anything, mock.Anything).Return(auth.Domain{}, tc.svcErr)
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

func TestViewDomainPermissions(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc     string
		token    string
		domainID string
		status   int
		svcErr   error
		err      error
	}{
		{
			desc:     "view domain permissions successfully",
			token:    validToken,
			domainID: id,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "view domain permissions with empty token",
			token:    "",
			domainID: id,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view domain permissions with invalid token",
			token:    inValidToken,
			domainID: id,
			status:   http.StatusUnauthorized,
			svcErr:   svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view domain permissions with empty domainID",
			token:    validToken,
			domainID: "",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ds.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/domains/%s/permissions", ds.URL, tc.domainID),
			token:  tc.token,
		}

		svcCall := svc.On("RetrieveDomainPermissions", mock.Anything, mock.Anything, mock.Anything).Return(auth.Permissions{}, tc.svcErr)
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

func TestUpdateDomain(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc        string
		token       string
		domain      auth.Domain
		contentType string
		status      int
		svcErr      error
		err         error
	}{
		{
			desc:  "update domain successfully",
			token: validToken,
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:  "update domain with empty token",
			token: "",
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:  "update domain with invalid token",
			token: inValidToken,
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			svcErr:      svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:  "update domain with invalid content type",
			token: validToken,
			domain: auth.Domain{
				ID:       ID,
				Name:     "test",
				Metadata: mgclients.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:  "update domain with data that cant be marshalled",
			token: validToken,
			domain: auth.Domain{
				ID:   ID,
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
				Tags:  []string{"tag1", "tag2"},
				Alias: "test",
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.domain)
		req := testRequest{
			client:      ds.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/domains/%s", ds.URL, tc.domain.ID),
			body:        strings.NewReader(data),
			contentType: tc.contentType,
			token:       tc.token,
		}

		svcCall := svc.On("UpdateDomain", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(auth.Domain{}, tc.svcErr)
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

func TestEnableDomain(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	disabledDomain := domain
	disabledDomain.Status = auth.DisabledStatus

	cases := []struct {
		desc     string
		domain   auth.Domain
		response auth.Domain
		token    string
		status   int
		svcErr   error
		err      error
	}{
		{
			desc:   "enable domain with valid token",
			domain: disabledDomain,
			response: auth.Domain{
				ID:     domain.ID,
				Status: auth.EnabledStatus,
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "enable domain with invalid token",
			domain: disabledDomain,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "enable domain with empty token",
			domain: disabledDomain,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc: "enable domain with empty id",
			domain: auth.Domain{
				ID: "",
			},
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
		{
			desc: "enable domain with invalid id",
			domain: auth.Domain{
				ID: "invalid",
			},
			token:  validToken,
			status: http.StatusForbidden,
			svcErr: svcerr.ErrAuthorization,
			err:    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.domain)
		req := testRequest{
			client:      ds.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/domains/%s/enable", ds.URL, tc.domain.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}
		svcCall := svc.On("ChangeDomainStatus", mock.Anything, tc.token, tc.domain.ID, mock.Anything).Return(tc.response, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestDisableDomain(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc     string
		domain   auth.Domain
		response auth.Domain
		token    string
		status   int
		svcErr   error
		err      error
	}{
		{
			desc:   "disable domain with valid token",
			domain: domain,
			response: auth.Domain{
				ID:     domain.ID,
				Status: auth.DisabledStatus,
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "disable domain with invalid token",
			domain: domain,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "disable domain with empty token",
			domain: domain,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc: "disable domain with empty id",
			domain: auth.Domain{
				ID: "",
			},
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
		{
			desc: "disable domain with invalid id",
			domain: auth.Domain{
				ID: "invalid",
			},
			token:  validToken,
			status: http.StatusForbidden,
			svcErr: svcerr.ErrAuthorization,
			err:    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.domain)
		req := testRequest{
			client:      ds.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/domains/%s/disable", ds.URL, tc.domain.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}
		svcCall := svc.On("ChangeDomainStatus", mock.Anything, tc.token, tc.domain.ID, mock.Anything).Return(tc.response, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestFreezeDomain(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc     string
		domain   auth.Domain
		response auth.Domain
		token    string
		status   int
		svcErr   error
		err      error
	}{
		{
			desc:   "freeze domain with valid token",
			domain: domain,
			response: auth.Domain{
				ID:     domain.ID,
				Status: auth.FreezeStatus,
			},
			token:  validToken,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "freeze domain with invalid token",
			domain: domain,
			token:  inValidToken,
			status: http.StatusUnauthorized,
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "freeze domain with empty token",
			domain: domain,
			token:  "",
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc: "freeze domain with empty id",
			domain: auth.Domain{
				ID: "",
			},
			token:  validToken,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingID,
		},
		{
			desc: "freeze domain with invalid id",
			domain: auth.Domain{
				ID: "invalid",
			},
			token:  validToken,
			status: http.StatusForbidden,
			svcErr: svcerr.ErrAuthorization,
			err:    svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		data := toJSON(tc.domain)
		req := testRequest{
			client:      ds.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/domains/%s/freeze", ds.URL, tc.domain.ID),
			contentType: contentType,
			token:       tc.token,
			body:        strings.NewReader(data),
		}
		svcCall := svc.On("ChangeDomainStatus", mock.Anything, tc.token, tc.domain.ID, mock.Anything).Return(tc.response, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestAssignDomainUsers(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc        string
		data        string
		domainID    string
		contentType string
		token       string
		status      int
		err         error
	}{
		{
			desc:        "assign domain users with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       validToken,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "assign domain users with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       inValidToken,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "assign domain users with empty token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "assign domain users with empty id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			domainID:    "",
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "assign domain users with invalid id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			domainID:    "invalid",
			contentType: contentType,
			token:       validToken,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "assign domain users with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", user_ids : ["%s", "%s"]}`, "editor", validID, validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "assign domain users with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "editor", validID, validID),
			domainID:    domain.ID,
			contentType: "application/xml",
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "assign domain users with empty user ids",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : []}`, "editor"),
			domainID:    domain.ID,
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "assign domain users with empty relation",
			data:        fmt.Sprintf(`{"relation": "%s", "user_ids" : ["%s", "%s"]}`, "", validID, validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingRelation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ds.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/domains/%s/users/assign", ds.URL, tc.domainID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		svcCall := svc.On("AssignUsers", mock.Anything, tc.token, tc.domainID, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestUnassignDomainUser(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc        string
		data        string
		domainID    string
		contentType string
		token       string
		status      int
		err         error
	}{
		{
			desc:        "unassign domain user with valid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_id" : "%s"}`, "editor", validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       validToken,
			status:      http.StatusNoContent,
			err:         nil,
		},
		{
			desc:        "unassign domain user with invalid token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_id" : "%s"}`, "editor", validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       inValidToken,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "unassign domain user with empty token",
			data:        fmt.Sprintf(`{"relation": "%s", "user_id" : "%s"}`, "editor", validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "unassign domain user with empty domain id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_id" : "%s"}`, "editor", validID),
			domainID:    "",
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "unassign domain user with invalid id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_id" : "%s"}`, "editor", validID),
			domainID:    "invalid",
			contentType: contentType,
			token:       validToken,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "unassign domain user with malformed data",
			data:        fmt.Sprintf(`{"relation": "%s", "user_id" : "%s}`, "editor", validID),
			domainID:    domain.ID,
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "unassign domain user with invalid content type",
			data:        fmt.Sprintf(`{"relation": "%s", "user_id" : "%s"}`, "editor", validID),
			domainID:    domain.ID,
			contentType: "application/xml",
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "unassign domain user with empty user id",
			data:        fmt.Sprintf(`{"relation": "%s", "user_id" : ""}`, "editor"),
			domainID:    domain.ID,
			contentType: contentType,
			token:       validToken,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ds.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/domains/%s/users/unassign", ds.URL, tc.domainID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.data),
		}

		svcCall := svc.On("UnassignUser", mock.Anything, tc.token, tc.domainID, mock.Anything, mock.Anything).Return(tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestListDomainsByUserID(t *testing.T) {
	ds, svc := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc               string
		token              string
		query              string
		listDomainsRequest auth.DomainsPage
		userID             string
		status             int
		svcErr             error
		err                error
	}{
		{
			desc:   "list domains by user id with valid token",
			token:  validToken,
			status: http.StatusOK,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			userID: validID,
			err:    nil,
		},
		{
			desc:   "list domains by user id with empty token",
			token:  "",
			userID: validID,
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "list domains by user id with invalid token",
			token:  inValidToken,
			userID: validID,
			status: http.StatusUnauthorized,
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:  "list domains by user id with offset",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "&offset=1",
			userID: validID,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains by user id with invalid offset",
			token:  validToken,
			query:  "&offset=invalid",
			status: http.StatusBadRequest,
			userID: validID,
			err:    apiutil.ErrValidation,
		},
		{
			desc:  "list domains by user id with limit",
			token: validToken,
			listDomainsRequest: auth.DomainsPage{
				Total:   1,
				Domains: []auth.Domain{domain},
			},
			query:  "&limit=1",
			userID: validID,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains by user id with invalid limit",
			token:  validToken,
			query:  "&limit=invalid",
			userID: validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: ds.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/domains?user=%s", ds.URL, tc.userID) + tc.query,
			token:  tc.token,
		}
		svcCall := svc.On("ListUserDomains", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.listDomainsRequest, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
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
