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
	"time"

	"github.com/absmach/magistrala/domains"
	httpapi "github.com/absmach/magistrala/domains/api/http"
	"github.com/absmach/magistrala/domains/mocks"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	authnmock "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validMetadata = domains.Metadata{"role": "client"}
	ID            = testsutil.GenerateUUID(&testing.T{})
	domain        = domains.Domain{
		ID:       ID,
		Name:     "domainname",
		Tags:     []string{"tag1", "tag2"},
		Metadata: validMetadata,
		Status:   domains.EnabledStatus,
		Alias:    "mydomain",
	}
	validToken   = "token"
	inValidToken = "invalid"
	invalid      = "invalid"
	userID       = testsutil.GenerateUUID(&testing.T{})
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

func newDomainsServer() (*httptest.Server, *mocks.Service, *authnmock.Authentication) {
	logger := mglog.NewMock()
	svc := new(mocks.Service)
	authn := new(authnmock.Authentication)
	mux := chi.NewMux()
	httpapi.MakeHandler(svc, authn, mux, logger, "")
	return httptest.NewServer(mux), svc, authn
}

func TestCreateDomain(t *testing.T) {
	ds, svc, auth := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc        string
		domain      domains.Domain
		token       string
		session     authn.Session
		contentType string
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc: "register a new domain successfully",
			domain: domains.Domain{
				Name:     "test",
				Metadata: domains.Metadata{"role": "domain"},
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
			domain: domains.Domain{
				Name:     "test",
				Metadata: domains.Metadata{"role": "domain"},
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
			domain: domains.Domain{
				Name:     "test",
				Metadata: domains.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			token:       inValidToken,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc: "register  a new domain with an empty name",
			domain: domains.Domain{
				Name:     "",
				Metadata: domains.Metadata{"role": "domain"},
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
			domain: domains.Domain{
				Name:     "test",
				Metadata: domains.Metadata{"role": "domain"},
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
			domain: domains.Domain{
				Name:     "test",
				Metadata: domains.Metadata{"role": "domain"},
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
			domain: domains.Domain{
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
		{
			desc: "register domain with service error",
			domain: domains.Domain{
				Name:     "test",
				Metadata: domains.Metadata{"role": "domain"},
				Tags:     []string{"tag1", "tag2"},
				Alias:    "test",
			},
			token:       validToken,
			contentType: contentType,
			status:      http.StatusUnprocessableEntity,
			svcErr:      svcerr.ErrCreateEntity,
			err:         svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.domain)
			req := testRequest{
				client:      ds.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/domains", ds.URL),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("CreateDomain", mock.Anything, tc.session, tc.domain).Return(tc.domain, tc.svcErr)
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

func TestListDomains(t *testing.T) {
	ds, svc, auth := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc            string
		token           string
		session         authn.Session
		query           string
		page            domains.Page
		listDomainsResp domains.DomainsPage
		status          int
		svcErr          error
		authnErr        error
		err             error
	}{
		{
			desc:  "list domains with valid token",
			token: validToken,
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  api.DefLimit,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			status: http.StatusOK,
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
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
			desc:     "list domains  with invalid token",
			token:    inValidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:  "list domains  with offset",
			token: validToken,
			query: "offset=1",
			page: domains.Page{
				Offset: 1,
				Limit:  api.DefLimit,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
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
			query: "limit=1",
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  1,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
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
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
			query: "name=domainname",
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  api.DefLimit,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Name:   "domainname",
			},
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
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
			query: "status=enabled",
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  api.DefLimit,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Status: domains.EnabledStatus,
			},
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
			desc:  "list domains with tags",
			token: validToken,
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
			query: "tag=tag1",
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  api.DefLimit,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Tag:    "tag1",
			},
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
			query: "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  api.DefLimit,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
				Metadata: domains.Metadata{
					"domain": "example.com",
				},
			},
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
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
			query: "permission=view",
			page: domains.Page{
				Offset:     api.DefOffset,
				Limit:      api.DefLimit,
				Order:      api.DefOrder,
				Dir:        api.DefDir,
				Permission: "view",
			},
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
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
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  api.DefLimit,
				Order:  "name",
				Dir:    api.DefDir,
			},
			query: "order=name",
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
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
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  api.DefLimit,
				Order:  api.DefOrder,
				Dir:    "asc",
			},
			query: "dir=asc",
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
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
		{
			desc:  "list domains with service error",
			token: validToken,
			page: domains.Page{
				Offset: api.DefOffset,
				Limit:  api.DefLimit,
				Order:  api.DefOrder,
				Dir:    api.DefDir,
			},
			status:          http.StatusBadRequest,
			listDomainsResp: domains.DomainsPage{},
			svcErr:          svcerr.ErrViewEntity,
			err:             svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ds.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/domains?", ds.URL) + tc.query,
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ListDomains", mock.Anything, tc.session, tc.page).Return(tc.listDomainsResp, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewDomain(t *testing.T) {
	ds, svc, auth := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc     string
		token    string
		session  authn.Session
		domainID string
		status   int
		svcRes   domains.Domain
		svcErr   error
		authnErr error
		err      error
	}{
		{
			desc:     "view domain successfully",
			token:    validToken,
			domainID: domain.ID,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "view domain with empty token",
			token:    "",
			domainID: domain.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view domain with invalid token",
			token:    inValidToken,
			domainID: domain.ID,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view domain with invalid id",
			token:    validToken,
			domainID: invalid,
			status:   http.StatusBadRequest,
			svcErr:   svcerr.ErrViewEntity,
			err:      svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ds.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/domains/%s", ds.URL, tc.domainID),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: tc.domainID, DomainUserID: tc.domainID + "_" + userID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RetrieveDomain", mock.Anything, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
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

func TestUpdateDomain(t *testing.T) {
	ds, svc, auth := newDomainsServer()
	defer ds.Close()

	updatedName := "test"
	updatedMetadata := domains.Metadata{"role": "domain"}
	updatedTags := []string{"tag1", "tag2"}
	updatedAlias := "test"
	updatedDomain := domains.Domain{
		ID:       ID,
		Name:     updatedName,
		Metadata: updatedMetadata,
		Tags:     updatedTags,
		Alias:    updatedAlias,
	}
	unMetadata := domains.Metadata{
		"test": make(chan int),
	}

	cases := []struct {
		desc        string
		token       string
		session     authn.Session
		domainID    string
		updateReq   domains.DomainReq
		contentType string
		status      int
		svcRes      domains.Domain
		svcErr      error
		authnErr    error
		err         error
	}{
		{
			desc:     "update domain successfully",
			token:    validToken,
			domainID: domain.ID,
			updateReq: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
				Tags:     &updatedTags,
				Alias:    &updatedAlias,
			},
			contentType: contentType,
			status:      http.StatusOK,
			svcRes:      updatedDomain,
			err:         nil,
		},
		{
			desc:     "update domain with empty token",
			token:    "",
			domainID: domain.ID,
			updateReq: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
				Tags:     &updatedTags,
				Alias:    &updatedAlias,
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:     "update domain with invalid token",
			token:    inValidToken,
			domainID: domain.ID,
			updateReq: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
				Tags:     &updatedTags,
				Alias:    &updatedAlias,
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "update domain with invalid content type",
			token:    validToken,
			domainID: domain.ID,
			updateReq: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
				Tags:     &updatedTags,
				Alias:    &updatedAlias,
			},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:     "update domain with data that cant be marshalled",
			token:    validToken,
			domainID: domain.ID,
			updateReq: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &unMetadata,
				Tags:     &updatedTags,
				Alias:    &updatedAlias,
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:     "update domain with invalid id",
			token:    validToken,
			domainID: invalid,
			updateReq: domains.DomainReq{
				Name:     &updatedName,
				Metadata: &updatedMetadata,
				Tags:     &updatedTags,
				Alias:    &updatedAlias,
			},
			contentType: contentType,
			status:      http.StatusUnprocessableEntity,
			svcErr:      svcerr.ErrUpdateEntity,
			err:         svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.updateReq)
			req := testRequest{
				client:      ds.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/domains/%s", ds.URL, tc.domainID),
				body:        strings.NewReader(data),
				contentType: tc.contentType,
				token:       tc.token,
			}
			fmt.Println("req url", req.url)

			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: tc.domainID, DomainUserID: tc.domainID + "_" + userID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateDomain", mock.Anything, tc.session, tc.domainID, tc.updateReq).Return(tc.svcRes, tc.svcErr)
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

func TestEnableDomain(t *testing.T) {
	ds, svc, auth := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc     string
		token    string
		session  authn.Session
		domainID string
		status   int
		svcErr   error
		svcRes   domains.Domain
		authnErr error
		err      error
	}{
		{
			desc:     "enable domain with valid token",
			token:    validToken,
			domainID: domain.ID,
			status:   http.StatusOK,
			svcRes:   domain,
			err:      nil,
		},
		{
			desc:     "enable domain with invalid token",
			token:    inValidToken,
			domainID: domain.ID,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "enable domain with empty token",
			token:    "",
			domainID: domain.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "enable domain with empty id",
			token:    validToken,
			domainID: "",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "enable domain with invalid id",
			token:    validToken,
			domainID: invalid,
			status:   http.StatusUnprocessableEntity,
			svcErr:   svcerr.ErrUpdateEntity,
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ds.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/domains/%s/enable", ds.URL, tc.domainID),
				contentType: contentType,
				token:       tc.token,
			}
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: tc.domainID, DomainUserID: tc.domainID + "_" + userID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("EnableDomain", mock.Anything, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableDomain(t *testing.T) {
	ds, svc, auth := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc     string
		token    string
		session  authn.Session
		domainID string
		status   int
		svcErr   error
		svcRes   domains.Domain
		authnErr error
		err      error
	}{
		{
			desc:     "disable domain with valid token",
			token:    validToken,
			domainID: domain.ID,
			status:   http.StatusOK,
			svcRes:   domain,
			err:      nil,
		},
		{
			desc:     "disable domain with invalid token",
			token:    inValidToken,
			domainID: domain.ID,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "disable domain with empty token",
			token:    "",
			domainID: domain.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "disable domain with empty id",
			token:    validToken,
			domainID: "",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "disable domain with invalid id",
			token:    validToken,
			domainID: invalid,
			status:   http.StatusUnprocessableEntity,
			svcErr:   svcerr.ErrUpdateEntity,
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ds.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/domains/%s/disable", ds.URL, tc.domainID),
				contentType: contentType,
				token:       tc.token,
			}
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: tc.domainID, DomainUserID: tc.domainID + "_" + userID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("DisableDomain", mock.Anything, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestFreezeDomain(t *testing.T) {
	ds, svc, auth := newDomainsServer()
	defer ds.Close()

	cases := []struct {
		desc     string
		token    string
		session  authn.Session
		domainID string
		status   int
		svcErr   error
		svcRes   domains.Domain
		authnErr error
		err      error
	}{
		{
			desc:     "freeze domain with valid token",
			token:    validToken,
			domainID: domain.ID,
			status:   http.StatusOK,
			svcRes:   domain,
			err:      nil,
		},
		{
			desc:     "freeze domain with invalid token",
			token:    inValidToken,
			domainID: domain.ID,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "freeze domain with empty token",
			token:    "",
			domainID: domain.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "freeze domain with empty id",
			token:    validToken,
			domainID: "",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "freeze domain with invalid id",
			token:    validToken,
			domainID: invalid,
			status:   http.StatusUnprocessableEntity,
			svcErr:   svcerr.ErrUpdateEntity,
			err:      svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ds.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/domains/%s/freeze", ds.URL, tc.domainID),
				contentType: contentType,
				token:       tc.token,
			}
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: tc.domainID, DomainUserID: tc.domainID + "_" + userID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("FreezeDomain", mock.Anything, tc.session, tc.domainID).Return(tc.svcRes, tc.svcErr)
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
	Status      domains.Status `json:"status"`
}
