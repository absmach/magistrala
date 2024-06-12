// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/certs"
	httpapi "github.com/absmach/magistrala/certs/api"
	"github.com/absmach/magistrala/certs/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	contentType = "application/json"
	valid       = "valid"
	invalid     = "invalid"
	thingID     = testsutil.GenerateUUID(&testing.T{})
	serial      = testsutil.GenerateUUID(&testing.T{})
	ttl         = "1h"
	cert        = certs.Cert{
		OwnerID:        testsutil.GenerateUUID(&testing.T{}),
		ThingID:        thingID,
		ClientCert:     valid,
		IssuingCA:      valid,
		CAChain:        []string{valid},
		ClientKey:      valid,
		PrivateKeyType: valid,
		Serial:         serial,
		Expire:         time.Now().Add(time.Hour),
	}
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

	return tr.client.Do(req)
}

func newCertServer() (*httptest.Server, *mocks.Service) {
	svc := new(mocks.Service)
	logger := mglog.NewMock()

	mux := httpapi.MakeHandler(svc, logger, "")
	return httptest.NewServer(mux), svc
}

func TestIssueCert(t *testing.T) {
	cs, svc := newCertServer()
	defer cs.Close()

	validReqString := `{"thing_id": "%s","ttl": "%s"}`
	invalidReqString := `{"thing_id": "%s","ttl": %s}`

	cases := []struct {
		desc        string
		token       string
		contentType string
		thingID     string
		ttl         string
		request     string
		status      int
		svcRes      certs.Cert
		svcErr      error
		err         error
	}{
		{
			desc:        "issue cert successfully",
			token:       valid,
			contentType: contentType,
			thingID:     thingID,
			ttl:         ttl,
			request:     fmt.Sprintf(validReqString, thingID, ttl),
			status:      http.StatusCreated,
			svcRes:      cert,
			svcErr:      nil,
			err:         nil,
		},
		{
			desc:        "issue with invalid token",
			token:       invalid,
			contentType: contentType,
			thingID:     thingID,
			ttl:         ttl,
			request:     fmt.Sprintf(validReqString, thingID, ttl),
			status:      http.StatusUnauthorized,
			svcRes:      certs.Cert{},
			svcErr:      svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "issue with empty token",
			token:       "",
			contentType: contentType,
			request:     fmt.Sprintf(validReqString, thingID, ttl),
			status:      http.StatusUnauthorized,
			svcRes:      certs.Cert{},
			svcErr:      nil,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "issue with empty thing id",
			token:       valid,
			contentType: contentType,
			request:     fmt.Sprintf(validReqString, "", ttl),
			status:      http.StatusBadRequest,
			svcRes:      certs.Cert{},
			svcErr:      nil,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "issue with empty ttl",
			token:       valid,
			contentType: contentType,
			request:     fmt.Sprintf(validReqString, thingID, ""),
			status:      http.StatusBadRequest,
			svcRes:      certs.Cert{},
			svcErr:      nil,
			err:         apiutil.ErrMissingCertData,
		},
		{
			desc:        "issue with invalid ttl",
			token:       valid,
			contentType: contentType,
			request:     fmt.Sprintf(validReqString, thingID, invalid),
			status:      http.StatusBadRequest,
			svcRes:      certs.Cert{},
			svcErr:      nil,
			err:         apiutil.ErrInvalidCertData,
		},
		{
			desc:        "issue with invalid content type",
			token:       valid,
			contentType: "application/xml",
			request:     fmt.Sprintf(validReqString, thingID, ttl),
			status:      http.StatusUnsupportedMediaType,
			svcRes:      certs.Cert{},
			svcErr:      nil,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "issue with invalid request body",
			token:       valid,
			contentType: contentType,
			request:     fmt.Sprintf(invalidReqString, thingID, ttl),
			status:      http.StatusInternalServerError,
			svcRes:      certs.Cert{},
			svcErr:      nil,
			err:         apiutil.ErrValidation,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      cs.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/certs", cs.URL),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.request),
		}
		svcCall := svc.On("IssueCert", mock.Anything, tc.token, tc.thingID, tc.ttl).Return(tc.svcRes, tc.svcErr)
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

func TestViewCert(t *testing.T) {
	cs, svc := newCertServer()
	defer cs.Close()

	cases := []struct {
		desc     string
		token    string
		serialID string
		status   int
		svcRes   certs.Cert
		svcErr   error
		err      error
	}{
		{
			desc:     "view cert successfully",
			token:    valid,
			serialID: serial,
			status:   http.StatusOK,
			svcRes:   cert,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "view with invalid token",
			token:    invalid,
			serialID: serial,
			status:   http.StatusUnauthorized,
			svcRes:   certs.Cert{},
			svcErr:   svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view with empty token",
			token:    "",
			serialID: serial,
			status:   http.StatusUnauthorized,
			svcRes:   certs.Cert{},
			svcErr:   nil,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "view non-existing cert",
			token:    valid,
			serialID: invalid,
			status:   http.StatusNotFound,
			svcRes:   certs.Cert{},
			svcErr:   svcerr.ErrNotFound,
			err:      svcerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: cs.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/certs/%s", cs.URL, tc.serialID),
			token:  tc.token,
		}
		svcCall := svc.On("ViewCert", mock.Anything, tc.token, tc.serialID).Return(tc.svcRes, tc.svcErr)
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

func TestRevokeCert(t *testing.T) {
	cs, svc := newCertServer()
	defer cs.Close()

	cases := []struct {
		desc     string
		token    string
		serialID string
		status   int
		svcRes   certs.Revoke
		svcErr   error
		err      error
	}{
		{
			desc:     "revoke cert successfully",
			token:    valid,
			serialID: serial,
			status:   http.StatusOK,
			svcRes:   certs.Revoke{RevocationTime: time.Now()},
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "revoke with invalid token",
			token:    invalid,
			serialID: serial,
			status:   http.StatusUnauthorized,
			svcRes:   certs.Revoke{},
			svcErr:   svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "revoke with empty token",
			token:    "",
			serialID: serial,
			status:   http.StatusUnauthorized,
			svcErr:   nil,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "revoke non-existing cert",
			token:    valid,
			serialID: invalid,
			status:   http.StatusNotFound,
			svcRes:   certs.Revoke{},
			svcErr:   svcerr.ErrNotFound,
			err:      svcerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: cs.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/certs/%s", cs.URL, tc.serialID),
			token:  tc.token,
		}
		svcCall := svc.On("RevokeCert", mock.Anything, tc.token, tc.serialID).Return(tc.svcRes, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var errRes respBody
		err = json.NewDecoder(res.Body).Decode(&errRes)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if errRes.Err != "" || errRes.Message != "" {
			err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n ", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestListSerials(t *testing.T) {
	cs, svc := newCertServer()
	defer cs.Close()

	cases := []struct {
		desc    string
		token   string
		thingID string
		offset  uint64
		limit   uint64
		query   string
		status  int
		svcRes  certs.Page
		svcErr  error
		err     error
	}{
		{
			desc:    "list certs successfully with default limit",
			token:   valid,
			thingID: thingID,
			offset:  0,
			limit:   10,
			query:   "",
			status:  http.StatusOK,
			svcRes: certs.Page{
				Total:  1,
				Offset: 0,
				Limit:  10,
				Certs:  []certs.Cert{cert},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:    "list certs successfully with limit",
			token:   valid,
			thingID: thingID,
			offset:  0,
			limit:   5,
			query:   "?limit=5",
			status:  http.StatusOK,
			svcRes: certs.Page{
				Total:  1,
				Offset: 0,
				Limit:  5,
				Certs:  []certs.Cert{cert},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:    "list certs successfully with offset",
			token:   valid,
			thingID: thingID,
			offset:  1,
			limit:   10,
			query:   "?offset=1",
			status:  http.StatusOK,
			svcRes: certs.Page{
				Total:  1,
				Offset: 1,
				Limit:  10,
				Certs:  []certs.Cert{},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:    "list certs successfully with offset and limit",
			token:   valid,
			thingID: thingID,
			offset:  1,
			limit:   5,
			query:   "?offset=1&limit=5",
			status:  http.StatusOK,
			svcRes: certs.Page{
				Total:  1,
				Offset: 1,
				Limit:  5,
				Certs:  []certs.Cert{},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:    "list with invalid token",
			token:   invalid,
			thingID: thingID,
			offset:  0,
			limit:   10,
			query:   "",
			status:  http.StatusUnauthorized,
			svcRes:  certs.Page{},
			svcErr:  svcerr.ErrAuthentication,
			err:     svcerr.ErrAuthentication,
		},
		{
			desc:    "list with empty token",
			token:   "",
			thingID: thingID,
			offset:  0,
			limit:   10,
			query:   "",
			status:  http.StatusUnauthorized,
			svcRes:  certs.Page{},
			svcErr:  nil,
			err:     apiutil.ErrBearerToken,
		},
		{
			desc:    "list with limit exceeding max limit",
			token:   valid,
			thingID: thingID,
			query:   "?limit=1000",
			status:  http.StatusBadRequest,
			svcRes:  certs.Page{},
			svcErr:  nil,
			err:     apiutil.ErrLimitSize,
		},
		{
			desc:    "list with invalid offset",
			token:   valid,
			thingID: thingID,
			query:   "?offset=invalid",
			status:  http.StatusBadRequest,
			svcRes:  certs.Page{},
			svcErr:  nil,
			err:     apiutil.ErrValidation,
		},
		{
			desc:    "list with invalid limit",
			token:   valid,
			thingID: thingID,
			query:   "?limit=invalid",
			status:  http.StatusBadRequest,
			svcRes:  certs.Page{},
			svcErr:  nil,
			err:     apiutil.ErrValidation,
		},
		{
			desc:    "list with invalid thing id",
			token:   valid,
			thingID: invalid,
			offset:  0,
			limit:   10,
			query:   "",
			status:  http.StatusNotFound,
			svcRes:  certs.Page{},
			svcErr:  svcerr.ErrNotFound,
			err:     svcerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client: cs.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/serials/%s", cs.URL, tc.thingID) + tc.query,
			token:  tc.token,
		}
		svcCall := svc.On("ListSerials", mock.Anything, tc.token, tc.thingID, tc.offset, tc.limit).Return(tc.svcRes, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var errRes respBody
		err = json.NewDecoder(res.Body).Decode(&errRes)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
		if errRes.Err != "" || errRes.Message != "" {
			err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n ", tc.desc, tc.err, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

type respBody struct {
	Err     string `json:"error"`
	Message string `json:"message"`
}
