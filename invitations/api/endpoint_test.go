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
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/invitations/api"
	"github.com/absmach/magistrala/invitations/mocks"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validToken      = "valid"
	validContenType = "application/json"
	validID         = testsutil.GenerateUUID(&testing.T{})
	domainID        = testsutil.GenerateUUID(&testing.T{})
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

func newIvitationsServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	logger := mglog.NewMock()
	authn := new(authnmocks.Authentication)
	mux := api.MakeHandler(svc, logger, authn, "test")
	return httptest.NewServer(mux), svc, authn
}

func TestSendInvitation(t *testing.T) {
	is, svc, authn := newIvitationsServer()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		data        string
		contentType string
		status      int
		authnRes    mgauthn.Session
		authnErr    error
		svcErr      error
	}{
		{
			desc:        "valid request",
			domainID:    domainID,
			token:       validToken,
			data:        fmt.Sprintf(`{"user_id": "%s", "relation": "%s"}`, validID, "domain"),
			authnRes:    mgauthn.Session{UserID: validID, DomainID: validID},
			status:      http.StatusCreated,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid token",
			domainID:    domainID,
			token:       "",
			data:        fmt.Sprintf(`{"user_id": "%s", "relation": "%s"}`, validID, "domain"),
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "empty domain_id",
			domainID:    "",
			token:       validToken,
			data:        fmt.Sprintf(`{"user_id": "%s", "relation": "%s"}`, validID, "domain"),
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid content type",
			domainID:    domainID,
			token:       validToken,
			data:        fmt.Sprintf(`{"user_id": "%s", "relation": "%s"}`, validID, "domain"),
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			svcErr:      nil,
		},
		{
			desc:        "invalid data",
			domainID:    domainID,
			token:       validToken,
			data:        `data`,
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with service error",
			domainID:    domainID,
			token:       validToken,
			data:        fmt.Sprintf(`{"user_id": "%s", "relation": "%s"}`, validID, "domain"),
			status:      http.StatusForbidden,
			contentType: validContenType,
			svcErr:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repoCall := svc.On("SendInvitation", mock.Anything, tc.authnRes, mock.Anything).Return(tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         is.URL + "/" + tc.domainID + "/invitations",
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
	is, svc, authn := newIvitationsServer()

	cases := []struct {
		desc        string
		domainID    string
		token       string
		query       string
		contentType string
		status      int
		svcErr      error
		authnRes    mgauthn.Session
		authnErr    error
	}{
		{
			desc:        "valid request",
			domainID:    domainID,
			token:       validToken,
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid token",
			domainID:    domainID,
			token:       "",
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with offset",
			domainID:    domainID,
			token:       validToken,
			query:       "offset=1",
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with invalid offset",
			domainID:    domainID,
			token:       validToken,
			query:       "offset=invalid",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with limit",
			domainID:    domainID,
			token:       validToken,
			query:       "limit=1",
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with invalid limit",
			domainID:    domainID,
			token:       validToken,
			query:       "limit=invalid",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with user_id",
			domainID:    domainID,
			token:       validToken,
			query:       fmt.Sprintf("user_id=%s", validID),
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with duplicate user_id",
			domainID:    domainID,
			token:       validToken,
			query:       "user_id=1&user_id=2",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with invited_by",
			domainID:    domainID,
			token:       validToken,
			query:       fmt.Sprintf("invited_by=%s", validID),
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with duplicate invited_by",
			domainID:    domainID,
			token:       validToken,
			query:       "invited_by=1&invited_by=2",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with relation",
			domainID:    domainID,
			token:       validToken,
			query:       fmt.Sprintf("relation=%s", "relation"),
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with duplicate relation",
			domainID:    domainID,
			token:       validToken,
			query:       "relation=1&relation=2",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with empty domain_id",
			domainID:    "",
			token:       validToken,
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with state",
			domainID:    domainID,
			token:       validToken,
			query:       "state=pending",
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with invalid state",
			domainID:    domainID,
			token:       validToken,
			query:       "state=invalid",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with duplicate state",
			domainID:    domainID,
			token:       validToken,
			query:       "state=all&state=all",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with service error",
			domainID:    domainID,
			token:       validToken,
			status:      http.StatusForbidden,
			contentType: validContenType,
			svcErr:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repoCall := svc.On("ListInvitations", mock.Anything, tc.authnRes, mock.Anything).Return(invitations.InvitationPage{}, tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodGet,
				url:         is.URL + "/" + tc.domainID + "/invitations?" + tc.query,
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
	is, svc, authn := newIvitationsServer()

	cases := []struct {
		desc        string
		token       string
		domainID    string
		userID      string
		contentType string
		status      int
		svcErr      error
		authnRes    mgauthn.Session
		authnErr    error
	}{
		{
			desc:        "valid request",
			token:       validToken,
			userID:      validID,
			domainID:    validID,
			status:      http.StatusOK,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid token",
			token:       "",
			userID:      validID,
			domainID:    validID,
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with service error",
			token:       validToken,
			userID:      validID,
			domainID:    validID,
			status:      http.StatusForbidden,
			contentType: validContenType,
			svcErr:      svcerr.ErrAuthorization,
		},
		{
			desc:        "with empty user_id",
			token:       validToken,
			userID:      "",
			domainID:    validID,
			status:      http.StatusNotFound,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with empty domain",
			token:       validToken,
			userID:      validID,
			domainID:    "",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with empty user_id and domain_id",
			token:       validToken,
			userID:      "",
			domainID:    "",
			status:      http.StatusNotFound,
			contentType: validContenType,
			svcErr:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repoCall := svc.On("ViewInvitation", mock.Anything, tc.authnRes, tc.userID, tc.domainID).Return(invitations.Invitation{}, tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodGet,
				url:         is.URL + "/" + tc.domainID + "/invitations/users/" + tc.userID,
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
	is, svc, authn := newIvitationsServer()
	_ = authn

	cases := []struct {
		desc        string
		token       string
		domainID    string
		userID      string
		contentType string
		status      int
		svcErr      error
		authnRes    mgauthn.Session
		authnErr    error
	}{
		{
			desc:        "valid request",
			token:       validToken,
			userID:      validID,
			domainID:    validID,
			status:      http.StatusNoContent,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid token",
			token:       "",
			userID:      validID,
			domainID:    validID,
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with service error",
			token:       validToken,
			userID:      validID,
			domainID:    validID,
			status:      http.StatusForbidden,
			contentType: validContenType,
			svcErr:      svcerr.ErrAuthorization,
		},
		{
			desc:        "with empty user_id",
			token:       validToken,
			userID:      "",
			domainID:    validID,
			status:      http.StatusNotFound,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with empty domain_id",
			token:       validToken,
			userID:      validID,
			domainID:    "",
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with empty user_id and domain_id",
			token:       validToken,
			userID:      "",
			domainID:    "",
			status:      http.StatusNotFound,
			contentType: validContenType,
			svcErr:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repoCall := svc.On("DeleteInvitation", mock.Anything, tc.authnRes, tc.userID, tc.domainID).Return(tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodDelete,
				url:         is.URL + "/" + tc.domainID + "/invitations/users/" + tc.userID,
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
	is, svc, authn := newIvitationsServer()
	_ = authn
	cases := []struct {
		desc        string
		domainID    string
		token       string
		data        string
		contentType string
		status      int
		svcErr      error
		authnRes    mgauthn.Session
		authnErr    error
	}{
		{
			desc:        "valid request",
			domainID:    domainID,
			token:       validToken,
			status:      http.StatusNoContent,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid token",
			domainID:    domainID,
			token:       "",
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "with service error",
			domainID:    domainID,
			token:       validToken,
			status:      http.StatusForbidden,
			contentType: validContenType,
			svcErr:      svcerr.ErrAuthorization,
		},
		{
			desc:        "invalid content type",
			domainID:    domainID,
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			svcErr:      nil,
		},
		{
			desc:        "empty domain",
			domainID:    "",
			token:       validToken,
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repoCall := svc.On("AcceptInvitation", mock.Anything, tc.authnRes, mock.Anything).Return(tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         is.URL + "/" + tc.domainID + "/invitations/accept",
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

func TestRejectInvitation(t *testing.T) {
	is, svc, authn := newIvitationsServer()
	_ = authn

	cases := []struct {
		desc        string
		domainID    string
		token       string
		contentType string
		status      int
		svcErr      error
		authnRes    mgauthn.Session
		authnErr    error
	}{
		{
			desc:        "valid request",
			domainID:    domainID,
			token:       validToken,
			status:      http.StatusNoContent,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "invalid token",
			domainID:    domainID,
			token:       "",
			status:      http.StatusUnauthorized,
			contentType: validContenType,
			svcErr:      nil,
		},
		{
			desc:        "unauthorized error",
			domainID:    domainID,
			token:       validToken,
			status:      http.StatusForbidden,
			contentType: validContenType,
			svcErr:      svcerr.ErrAuthorization,
		},
		{
			desc:        "invalid content type",
			domainID:    domainID,
			token:       validToken,
			status:      http.StatusUnsupportedMediaType,
			contentType: "text/plain",
			svcErr:      nil,
		},
		{
			desc:        "empty domain",
			domainID:    "",
			token:       validToken,
			status:      http.StatusBadRequest,
			contentType: validContenType,
			svcErr:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authnCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			repoCall := svc.On("RejectInvitation", mock.Anything, tc.authnRes, mock.Anything).Return(tc.svcErr)
			req := testRequest{
				client:      is.Client(),
				method:      http.MethodPost,
				url:         is.URL + "/" + tc.domainID + "/invitations/reject",
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
