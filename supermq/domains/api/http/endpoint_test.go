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

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/domains"
	domainsapi "github.com/absmach/supermq/domains/api/http"
	"github.com/absmach/supermq/domains/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	smqlog "github.com/absmach/supermq/logger"
	"github.com/absmach/supermq/pkg/authn"
	authnmock "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/uuid"
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
	validID      = testsutil.GenerateUUID(&testing.T{})
	domainID     = testsutil.GenerateUUID(&testing.T{})
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
	logger := smqlog.NewMock()
	svc := new(mocks.Service)
	authn := new(authnmock.Authentication)
	mux := chi.NewMux()
	idp := uuid.NewMock()
	domainsapi.MakeHandler(svc, authn, mux, logger, "", idp)
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
			svcCall := svc.On("CreateDomain", mock.Anything, tc.session, tc.domain).Return(tc.domain, []roles.RoleProvision{}, tc.svcErr)
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
			desc:  "list domains  with role name",
			token: validToken,
			query: "role_name=view",
			page: domains.Page{
				Offset:   api.DefOffset,
				Limit:    api.DefLimit,
				Order:    api.DefOrder,
				Dir:      api.DefDir,
				RoleName: "view",
			},
			listDomainsResp: domains.DomainsPage{
				Total:   1,
				Domains: []domains.Domain{domain},
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "list domains  with invalid role name",
			token:  validToken,
			query:  "role_name= ",
			status: http.StatusBadRequest,
			err:    apiutil.ErrValidation,
		},
		{
			desc:   "list domains  with duplicate role name",
			token:  validToken,
			query:  "role_name=view&role_name=view",
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

func TestSendInvitation(t *testing.T) {
	is, svc, auth := newDomainsServer()

	cases := []struct {
		desc        string
		token       string
		domainID    string
		data        string
		session     authn.Session
		contentType string
		status      int
		authnErr    error
		svcErr      error
	}{
		{
			desc:        "send invitation with valid request",
			token:       validToken,
			domainID:    domainID,
			data:        fmt.Sprintf(`{"invitee_user_id": "%s","role_id": "%s"}`, validID, validID),
			status:      http.StatusCreated,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "send invitation with invalid token",
			token:       "",
			domainID:    domainID,
			data:        fmt.Sprintf(`{"invitee_user_id": "%s","role_id": "%s"}`, validID, validID),
			status:      http.StatusUnauthorized,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "send invitation with empty domain_id",
			token:       validToken,
			domainID:    "",
			data:        fmt.Sprintf(`{"invitee_user_id": "%s","role_id": "%s"}`, validID, validID),
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "send invitation with invalid content type",
			token:       validToken,
			domainID:    domainID,
			data:        fmt.Sprintf(`{"invitee_user_id": "%s","role_id": "%s"}`, validID, validID),
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			svcErr:      nil,
		},
		{
			desc:        "send invitation with invalid data",
			token:       validToken,
			domainID:    domainID,
			data:        `data`,
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "send invitation with service error",
			token:       validToken,
			domainID:    domainID,
			data:        fmt.Sprintf(`{"invitee_user_id": "%s","role_id": "%s"}`, validID, validID),
			status:      http.StatusForbidden,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: domainID, DomainUserID: domainID + "_" + userID}
			}
			authnCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			repoCall := svc.On("SendInvitation", mock.Anything, tc.session, mock.Anything).Return(tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/domains/%s/invitations", is.URL, tc.domainID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}

			res, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, res.StatusCode, tc.desc)
			repoCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestListInvitation(t *testing.T) {
	is, svc, auth := newDomainsServer()

	cases := []struct {
		desc        string
		token       string
		session     authn.Session
		query       string
		contentType string
		status      int
		svcErr      error
		authnErr    error
	}{
		{
			desc:        "list invitations with valid request",
			token:       validToken,
			status:      http.StatusOK,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with invalid token",
			token:       "",
			status:      http.StatusUnauthorized,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with offset",
			token:       validToken,
			query:       "offset=1",
			status:      http.StatusOK,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with invalid offset",
			token:       validToken,
			query:       "offset=invalid",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with limit",
			token:       validToken,
			query:       "limit=1",
			status:      http.StatusOK,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with invalid limit",
			token:       validToken,
			query:       "limit=invalid",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with invitee_user_id",
			token:       validToken,
			query:       fmt.Sprintf("invitee_user_id=%s", validID),
			status:      http.StatusOK,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with duplicate invitee_user_id",
			token:       validToken,
			query:       "invitee_user_id=1&invitee_user_id=2",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with invited_by",
			token:       validToken,
			query:       fmt.Sprintf("invited_by=%s", validID),
			status:      http.StatusOK,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with duplicate invited_by",
			token:       validToken,
			query:       "invited_by=1&invited_by=2",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with state",
			token:       validToken,
			query:       "state=pending",
			status:      http.StatusOK,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with invalid state",
			token:       validToken,
			query:       "state=invalid",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with duplicate state",
			token:       validToken,
			query:       "state=all&state=all",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "list invitations with service error",
			token:       validToken,
			status:      http.StatusForbidden,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID}
			}
			authnCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			repoCall := svc.On("ListInvitations", mock.Anything, tc.session, mock.Anything).Return(domains.InvitationPage{}, tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodGet,
				url:         is.URL + "/invitations?" + tc.query,
				token:       tc.token,
				contentType: tc.contentType,
			}
			res, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, res.StatusCode, tc.desc)
			repoCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestViewInvitation(t *testing.T) {
	is, svc, auth := newDomainsServer()

	cases := []struct {
		desc        string
		token       string
		session     authn.Session
		domainID    string
		userID      string
		contentType string
		status      int
		svcErr      error
		authnErr    error
	}{
		{
			desc:        "view invitation with valid request",
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			status:      http.StatusOK,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "view invitation with invalid token",
			token:       "",
			userID:      validID,
			domainID:    domainID,
			status:      http.StatusUnauthorized,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "view invitation with service error",
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      svcerr.ErrViewEntity,
		},
		{
			desc:        "view invitation with empty domain",
			token:       validToken,
			userID:      validID,
			domainID:    "",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "view invitation with empty invitee_user_id and domain_id",
			token:       validToken,
			userID:      "",
			domainID:    "",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: domainID, DomainUserID: domainID + "_" + userID}
			}
			authnCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			repoCall := svc.On("ViewInvitation", mock.Anything, tc.session, tc.userID, tc.domainID).Return(domains.Invitation{}, tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodGet,
				url:         fmt.Sprintf("%s/domains/%s/invitations/%s", is.URL, tc.domainID, tc.userID),
				token:       tc.token,
				contentType: tc.contentType,
			}

			res, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, res.StatusCode, tc.desc)
			repoCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestDeleteInvitation(t *testing.T) {
	is, svc, auth := newDomainsServer()

	cases := []struct {
		desc        string
		token       string
		session     authn.Session
		domainID    string
		userID      string
		contentType string
		status      int
		svcErr      error
		authnErr    error
	}{
		{
			desc:        "delete invitation with valid request",
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			status:      http.StatusNoContent,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "delete invitation with invalid token",
			token:       "",
			userID:      validID,
			domainID:    domainID,
			status:      http.StatusUnauthorized,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "delete invitation with service error",
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			status:      http.StatusForbidden,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
		},
		{
			desc:        "delete invitation with empty invitee_user_id",
			token:       validToken,
			userID:      "",
			domainID:    domainID,
			status:      http.StatusMethodNotAllowed,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "delete invitation with empty domain_id",
			token:       validToken,
			userID:      validID,
			domainID:    "",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "delete invitation with empty invitee_user_id and domain_id",
			token:       validToken,
			userID:      "",
			domainID:    "",
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: domainID, DomainUserID: domainID + "_" + userID}
			}
			authnCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			repoCall := svc.On("DeleteInvitation", mock.Anything, tc.session, tc.userID, tc.domainID).Return(tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodDelete,
				url:         fmt.Sprintf("%s/domains/%s/invitations/%s", is.URL, tc.domainID, tc.userID),
				token:       tc.token,
				contentType: tc.contentType,
			}

			res, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, res.StatusCode, tc.desc)
			repoCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestAcceptInvitation(t *testing.T) {
	is, svc, auth := newDomainsServer()

	cases := []struct {
		desc        string
		token       string
		session     authn.Session
		data        string
		contentType string
		status      int
		svcErr      error
		authnErr    error
	}{
		{
			desc:        "accept invitation with valid request",
			data:        fmt.Sprintf(`{"domain_id": "%s"}`, validID),
			token:       validToken,
			status:      http.StatusNoContent,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "accept invitation with invalid token",
			token:       "",
			data:        fmt.Sprintf(`{"domain_id": "%s"}`, validID),
			status:      http.StatusUnauthorized,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "accept invitation with service error",
			token:       validToken,
			data:        fmt.Sprintf(`{"domain_id": "%s"}`, validID),
			status:      http.StatusForbidden,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
		},
		{
			desc:        "accept invitation with invalid content type",
			token:       validToken,
			data:        fmt.Sprintf(`{"domain_id": "%s"}`, validID),
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			svcErr:      nil,
		},
		{
			desc:        "accept invitation with invalid data",
			token:       validToken,
			data:        `data`,
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: domainID}
			}
			authnCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			repoCall := svc.On("AcceptInvitation", mock.Anything, tc.session, mock.Anything).Return(tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         is.URL + "/invitations/accept",
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}

			res, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, res.StatusCode, tc.desc)
			repoCall.Unset()
			authnCall.Unset()
		})
	}
}

func TestRejectInvitation(t *testing.T) {
	is, svc, auth := newDomainsServer()

	cases := []struct {
		desc        string
		token       string
		session     authn.Session
		data        string
		contentType string
		status      int
		svcErr      error
		authnErr    error
	}{
		{
			desc:        "reject invitation with valid request",
			token:       validToken,
			data:        fmt.Sprintf(`{"domain_id": "%s"}`, validID),
			status:      http.StatusNoContent,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "reject invitation with invalid token",
			token:       "",
			data:        fmt.Sprintf(`{"domain_id": "%s"}`, validID),
			status:      http.StatusUnauthorized,
			contentType: contentType,
			svcErr:      nil,
		},
		{
			desc:        "reject invitation with unauthorized error",
			token:       validToken,
			data:        fmt.Sprintf(`{"domain_id": "%s"}`, "invalid"),
			status:      http.StatusForbidden,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
		},
		{
			desc:        "reject invitation with invalid content type",
			token:       validToken,
			data:        fmt.Sprintf(`{"domain_id": "%s"}`, validID),
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			svcErr:      nil,
		},
		{
			desc:        "reject invitation with invalid data",
			token:       validToken,
			data:        `data`,
			status:      http.StatusBadRequest,
			contentType: contentType,
			svcErr:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = authn.Session{UserID: userID, DomainID: domainID}
			}
			authnCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			repoCall := svc.On("RejectInvitation", mock.Anything, tc.session, mock.Anything).Return(tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         is.URL + "/invitations/reject",
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}

			res, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, res.StatusCode, tc.desc)
			repoCall.Unset()
			authnCall.Unset()
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
