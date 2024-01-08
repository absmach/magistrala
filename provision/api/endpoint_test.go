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

	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/provision"
	"github.com/absmach/magistrala/provision/api"
	"github.com/absmach/magistrala/provision/mocks"
	"github.com/stretchr/testify/assert"
)

var (
	validToken      = "valid"
	validContenType = "application/json"
	validID         = testsutil.GenerateUUID(&testing.T{})
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

func newProvisionServer() (*httptest.Server, *mocks.Service) {
	svc := new(mocks.Service)

	logger := mglog.NewMock()
	mux := api.MakeHandler(svc, logger, "test")
	return httptest.NewServer(mux), svc
}

func TestProvision(t *testing.T) {
	is, svc := newProvisionServer()

	cases := []struct {
		desc        string
		token       string
		data        string
		contentType string
		status      int
		svcErr      error
	}{
		{
			desc:        "valid request",
			token:       validToken,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s", "external_key": "%s"}`, validID, validID),
			status:      http.StatusCreated,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "request with empty external id",
			token:       validToken,
			data:        fmt.Sprintf(`{"name": "test", "external_key": "%s"}`, validID),
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "request with empty external key",
			token:       validToken,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s"}`, validID),
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "empty token",
			token:       "",
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s", "external_key": "%s"}`, validID, validID),
			status:      http.StatusCreated,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid content type",
			token:       validToken,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s", "external_key": "%s"}`, validID, validID),
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			svcErr:      nil,
		},
		{
			desc:        "invalid request",
			token:       validToken,
			data:        `data`,
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "service error",
			token:       validToken,
			data:        fmt.Sprintf(`{"name": "test", "external_id": "%s", "external_key": "%s"}`, validID, validID),
			status:      http.StatusForbidden,
			contentType: validContenType,
			svcErr:      errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := svc.On("Provision", tc.token, "test", validID, validID).Return(provision.Result{}, tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         is.URL + "/mapping",
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}

			resp, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, resp.StatusCode, tc.desc)
			repocall.Unset()
		})
	}
}

func TestMapping(t *testing.T) {
	is, svc := newProvisionServer()

	cases := []struct {
		desc        string
		token       string
		contentType string
		status      int
		svcErr      error
	}{
		{
			desc:        "valid request",
			token:       validToken,
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "empty token",
			token:       "",
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid content type",
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			svcErr:      nil,
		},
		{
			desc:        "service error",
			token:       validToken,
			status:      http.StatusForbidden,
			contentType: validContenType,
			svcErr:      errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repocall := svc.On("Mapping", tc.token).Return(map[string]interface{}{}, tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodGet,
				url:         is.URL + "/mapping",
				token:       tc.token,
				contentType: tc.contentType,
			}

			resp, err := req.make()
			assert.Nil(t, err, tc.desc)
			assert.Equal(t, tc.status, resp.StatusCode, tc.desc)
			repocall.Unset()
		})
	}
}
