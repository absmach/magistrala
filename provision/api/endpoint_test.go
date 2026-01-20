// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/provision"
	"github.com/absmach/magistrala/provision/api"
	mocks "github.com/absmach/magistrala/provision/mocks"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/auth"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validToken      = "valid"
	validContenType = "application/json"
	validID         = testsutil.GenerateUUID(&testing.T{})
	userID          = testsutil.GenerateUUID(&testing.T{})
	domainID        = testsutil.GenerateUUID(&testing.T{})
	validSession    = smqauthn.Session{
		DomainUserID: auth.EncodeDomainUserID(domainID, userID),
		UserID:       userID,
		DomainID:     domainID,
	}
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	token       string
	contentType string
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

	return tr.client.Do(req)
}

func newProvisionServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)

	logger := smqlog.NewMock()
	authn := new(authnmocks.Authentication)
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithAllowUnverifiedUser(true))
	mux := api.MakeHandler(svc, am, logger, "test")
	return httptest.NewServer(mux), svc, authn
}

func TestProvision(t *testing.T) {
	is, svc, authn := newProvisionServer()

	cases := []struct {
		desc        string
		token       string
		domainID    string
		data        string
		contentType string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		svcErr      error
	}{
		{
			desc:        "valid request",
			token:       validToken,
			domainID:    validID,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s", "external_key": "%s"}`, validID, validID),
			status:      http.StatusCreated,
			contentType: validContenType,
			authnRes:    validSession,
			svcErr:      nil,
		},
		{
			desc:        "request with empty external id",
			token:       validToken,
			domainID:    validID,
			data:        fmt.Sprintf(`{"name": "test", "external_key": "%s"}`, validID),
			status:      http.StatusBadRequest,
			contentType: validContenType,
			authnRes:    validSession,
		},
		{
			desc:        "request with empty external key",
			token:       validToken,
			domainID:    validID,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s"}`, validID),
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			authnRes:    validSession,
			svcErr:      nil,
		},
		{
			desc:        "empty token",
			token:       "",
			domainID:    validID,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s", "external_key": "%s"}`, validID, validID),
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			authnRes:    smqauthn.Session{},
			authnErr:    errors.ErrAuthentication,
			svcErr:      nil,
		},
		{
			desc:        "invalid content type",
			token:       validToken,
			domainID:    validID,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s", "external_key": "%s"}`, validID, validID),
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			authnRes:    validSession,
			svcErr:      nil,
		},
		{
			desc:        "invalid request",
			token:       validToken,
			domainID:    validID,
			data:        `data`,
			status:      http.StatusBadRequest,
			contentType: validContenType,
			authnRes:    validSession,
			svcErr:      nil,
		},
		{
			desc:        "service error",
			token:       validToken,
			domainID:    validID,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s", "external_key": "%s"}`, validID, validID),
			status:      http.StatusForbidden,
			contentType: validContenType,
			authnRes:    validSession,
			svcErr:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repocall := svc.On("Provision", mock.Anything, validID, tc.token, "test", validID, validID).Return(provision.Result{}, tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         is.URL + fmt.Sprintf("/%s/mapping", tc.domainID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}

			resp, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, resp.StatusCode, tc.desc)
			authCall.Unset()
			repocall.Unset()
		})
	}
}

func TestMapping(t *testing.T) {
	is, svc, authn := newProvisionServer()

	cases := []struct {
		desc        string
		token       string
		domainID    string
		contentType string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		svcErr      error
	}{
		{
			desc:        "valid request",
			token:       validToken,
			domainID:    validID,
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
			authnRes:    validSession,
			authnErr:    nil,
		},
		{
			desc:        "empty token",
			token:       "",
			domainID:    validID,
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			svcErr:      nil,
			authnRes:    smqauthn.Session{},
			authnErr:    errors.ErrAuthentication,
		},
		{
			desc:        "service error",
			token:       validToken,
			domainID:    validID,
			status:      http.StatusForbidden,
			contentType: validContenType,
			authnRes:    validSession,
			authnErr:    nil,
			svcErr:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repocall := svc.On("Mapping").Return(map[string]any{}, tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodGet,
				url:         is.URL + fmt.Sprintf("/%s/mapping", tc.domainID),
				token:       tc.token,
				contentType: tc.contentType,
			}

			resp, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, resp.StatusCode, tc.desc)
			authCall.Unset()
			repocall.Unset()
		})
	}
}

func TestCert(t *testing.T) {
	is, svc, authn := newProvisionServer()

	cases := []struct {
		desc        string
		token       string
		domainID    string
		data        string
		contentType string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		svcErr      error
	}{
		{
			desc:        "valid request",
			token:       validToken,
			domainID:    validID,
			data:        fmt.Sprintf(`{"client_id": "%s", "ttl": "1h"}`, validID),
			status:      http.StatusCreated,
			contentType: validContenType,
			authnRes:    validSession,
			svcErr:      nil,
		},
		{
			desc:        "empty token",
			token:       "",
			domainID:    validID,
			data:        fmt.Sprintf(`{"client_id": "%s", "ttl": "1h"}`, validID),
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			authnRes:    smqauthn.Session{},
			authnErr:    errors.ErrAuthentication,
			svcErr:      nil,
		},
		{
			desc:        "invalid content type",
			token:       validToken,
			domainID:    validID,
			data:        fmt.Sprintf(`{"client_id": "%s", "ttl": "1h"}`, validID),
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			authnRes:    validSession,
			svcErr:      nil,
		},
		{
			desc:        "invalid request",
			token:       validToken,
			domainID:    validID,
			data:        `data`,
			status:      http.StatusBadRequest,
			contentType: validContenType,
			authnRes:    validSession,
			svcErr:      nil,
		},
		{
			desc:        "service error",
			token:       validToken,
			domainID:    validID,
			data:        fmt.Sprintf(`{"client_id": "%s", "ttl": "1h"}`, validID),
			status:      http.StatusForbidden,
			contentType: validContenType,
			authnRes:    validSession,
			svcErr:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repocall := svc.On("Cert", mock.Anything, validID, tc.token, validID, "1h").Return("cert", "key", tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         is.URL + fmt.Sprintf("/%s/cert", tc.domainID),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}

			resp, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, resp.StatusCode, tc.desc)
			authCall.Unset()
			repocall.Unset()
		})
	}
}
