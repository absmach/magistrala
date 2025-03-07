// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/journal"
	"github.com/absmach/supermq/journal/api"
	"github.com/absmach/supermq/journal/mocks"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var validToken = "valid"

type testRequest struct {
	client *http.Client
	method string
	url    string
	token  string
	body   io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}

	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}

	return tr.client.Do(req)
}

func newjournalServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)

	logger := smqlog.NewMock()
	authn := new(authnmocks.Authentication)
	mux := api.MakeHandler(svc, authn, logger, "journal-log", "test")
	return httptest.NewServer(mux), svc, authn
}

func TestListUserJournalsEndpoint(t *testing.T) {
	es, svc, authn := newjournalServer()

	cases := []struct {
		desc        string
		token       string
		session     smqauthn.Session
		url         string
		contentType string
		status      int
		svcErr      error
	}{
		{
			desc:   "successful",
			token:  validToken,
			url:    "/user/123",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "empty token",
			token:  "",
			url:    "/user/123",
			status: http.StatusUnauthorized,
			svcErr: nil,
		},
		{
			desc:   "with service error",
			token:  validToken,
			url:    "/user/123",
			status: http.StatusForbidden,
			svcErr: svcerr.ErrAuthorization,
		},
		{
			desc:   "with offset",
			token:  validToken,
			url:    "/user/123?offset=10",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid offset",
			token:  validToken,
			url:    "/user/123?offset=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with limit",
			token:  validToken,
			url:    "/user/123?limit=10",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid limit",
			token:  validToken,
			url:    "/user/123?limit=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with operation",
			token:  validToken,
			url:    "/user/123?operation=user.create",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with malformed operation",
			token:  validToken,
			url:    "/user/123?operation=user.create&operation=user.update",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with from",
			token:  validToken,
			url:    fmt.Sprintf("/user/123?from=%d", time.Now().Unix()),
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid from",
			token:  validToken,
			url:    "/user/123?from=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with invalid from as UnixNano",
			token:  validToken,
			url:    fmt.Sprintf("/user/123?from=%d", time.Now().UnixNano()),
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with to",
			token:  validToken,
			url:    fmt.Sprintf("/user/123?to=%d", time.Now().Unix()),
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid to",
			token:  validToken,
			url:    "/user/123?to=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with invalid to as UnixNano",
			token:  validToken,
			url:    fmt.Sprintf("/user/123?to=%d", time.Now().UnixNano()),
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with attributes",
			token:  validToken,
			url:    fmt.Sprintf("/user/123?with_attributes=%s", strconv.FormatBool(true)),
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid attributes",
			token:  validToken,
			url:    "/user/123?with_attributes=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with metadata",
			token:  validToken,
			url:    fmt.Sprintf("/user/123?with_metadata=%s", strconv.FormatBool(true)),
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid metadata",
			token:  validToken,
			url:    "/user/123?with_metadata=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with asc direction",
			token:  validToken,
			url:    "/user/123?dir=asc",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with desc direction",
			token:  validToken,
			url:    "/user/123?dir=desc",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid direction",
			token:  validToken,
			url:    "/user/123?dir=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with malformed direction",
			token:  validToken,
			url:    "/user/123?dir=invalid&dir=invalid2",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with empty url",
			token:  validToken,
			url:    "",
			status: http.StatusNotFound,
			svcErr: nil,
		},
		{
			desc:   "with empty entity type",
			token:  validToken,
			url:    "//123",
			status: http.StatusNotFound,
			svcErr: nil,
		},
		{
			desc:   "with empty entity ID",
			token:  validToken,
			url:    "/user/",
			status: http.StatusNotFound,
			svcErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if c.token == validToken {
				c.session = smqauthn.Session{
					UserID: testsutil.GenerateUUID(t),
				}
			}
			authCall := authn.On("Authenticate", mock.Anything, c.token).Return(c.session, nil)
			svcCall := svc.On("RetrieveAll", mock.Anything, c.session, mock.Anything).Return(journal.JournalsPage{}, c.svcErr)
			req := testRequest{
				client: es.Client(),
				method: http.MethodGet,
				url:    es.URL + "/journal" + c.url,
				token:  c.token,
			}

			resp, err := req.make()
			assert.Nil(t, err, c.desc)
			defer resp.Body.Close()
			assert.Equal(t, c.status, resp.StatusCode, c.desc)
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListEntityJournalsEndpoint(t *testing.T) {
	es, svc, authn := newjournalServer()

	domainID := testsutil.GenerateUUID(t)
	userID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc        string
		token       string
		session     smqauthn.Session
		domainID    string
		url         string
		contentType string
		status      int
		authnErr    error
		svcErr      error
	}{
		{
			desc:     "with group type successful",
			token:    validToken,
			domainID: domainID,
			url:      "/group/123",
			status:   http.StatusOK,
			svcErr:   nil,
		},
		{
			desc:     "with channel type successful",
			token:    validToken,
			domainID: domainID,
			url:      "/channel/123",
			status:   http.StatusOK,
			svcErr:   nil,
		},
		{
			desc:     "with client type successful",
			token:    validToken,
			domainID: domainID,
			url:      "/client/123",
			status:   http.StatusOK,
			svcErr:   nil,
		},
		{
			desc:     "with service error",
			token:    validToken,
			domainID: domainID,
			url:      "/client/123",
			status:   http.StatusForbidden,
			svcErr:   svcerr.ErrAuthorization,
		},
		{
			desc:     "with operation",
			token:    validToken,
			domainID: domainID,
			url:      "/channel/123?operation=channel.create",
			status:   http.StatusOK,
			svcErr:   nil,
		},
		{
			desc:     "with malformed operation",
			token:    validToken,
			domainID: domainID,
			url:      "/user/123?operation=user.create&operation=user.update",
			status:   http.StatusBadRequest,
			svcErr:   nil,
		},
		{
			desc:     "with invalid entity type",
			token:    validToken,
			domainID: domainID,
			url:      "/invalid/123",
			status:   http.StatusBadRequest,
			svcErr:   nil,
		},
		{
			desc:     "with all query params",
			token:    validToken,
			domainID: domainID,
			url:      "/group/123?offset=10&limit=10&operation=group.create&from=0&to=10&with_attributes=true&with_metadata=true&dir=asc",
			status:   http.StatusOK,
			svcErr:   nil,
		},
		{
			desc:     " with empty token",
			url:      "/group/123",
			domainID: domainID,
			status:   http.StatusUnauthorized,
			svcErr:   nil,
		},
		{
			desc:   "with empty domain ID",
			token:  validToken,
			url:    "/group/",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if c.token == validToken {
				c.session = smqauthn.Session{
					UserID:       userID,
					DomainID:     domainID,
					DomainUserID: domainID + "_" + userID,
				}
			}
			authCall := authn.On("Authenticate", mock.Anything, c.token).Return(c.session, c.authnErr)
			svcCall := svc.On("RetrieveAll", mock.Anything, c.session, mock.Anything).Return(journal.JournalsPage{}, c.svcErr)
			req := testRequest{
				client: es.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/journal%s", es.URL, c.domainID, c.url),
				token:  c.token,
			}
			resp, err := req.make()
			assert.Nil(t, err, c.desc)
			defer resp.Body.Close()
			assert.Equal(t, c.status, resp.StatusCode, c.desc)
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRetrieveClientTelemetryEndpoint(t *testing.T) {
	es, svc, authn := newjournalServer()

	clientID := testsutil.GenerateUUID(t)
	userID := testsutil.GenerateUUID(t)
	domanID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc        string
		token       string
		session     smqauthn.Session
		clientID    string
		domainID    string
		url         string
		contentType string
		status      int
		authnErr    error
		svcErr      error
	}{
		{
			desc:     "successful",
			token:    validToken,
			clientID: clientID,
			domainID: domanID,
			url:      fmt.Sprintf("/client/%s/telemetry", clientID),
			status:   http.StatusOK,
			svcErr:   nil,
		},
		{
			desc:     "with service error",
			token:    validToken,
			clientID: clientID,
			domainID: domanID,
			url:      fmt.Sprintf("/client/%s/telemetry", clientID),
			status:   http.StatusForbidden,
			svcErr:   svcerr.ErrAuthorization,
		},
		{
			desc:     "with empty token",
			clientID: clientID,
			domainID: domanID,
			url:      fmt.Sprintf("/client/%s/telemetry", clientID),
			status:   http.StatusUnauthorized,
			svcErr:   nil,
		},
		{
			desc:     "with invalid client ID",
			token:    validToken,
			domainID: domanID,
			clientID: "invalid",
			url:      "/client/invalid/telemetry",
			status:   http.StatusNotFound,
			svcErr:   svcerr.ErrNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if c.token == validToken {
				c.session = smqauthn.Session{
					UserID:       userID,
					DomainID:     c.domainID,
					DomainUserID: c.domainID + "_" + userID,
				}
			}
			authCall := authn.On("Authenticate", mock.Anything, c.token).Return(c.session, c.authnErr)
			svcCall := svc.On("RetrieveClientTelemetry", mock.Anything, c.session, c.clientID).Return(journal.ClientTelemetry{}, c.svcErr)
			req := testRequest{
				client: es.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/journal%s", es.URL, c.domainID, c.url),
				token:  c.token,
			}
			resp, err := req.make()
			assert.Nil(t, err, c.desc)
			defer resp.Body.Close()
			assert.Equal(t, c.status, resp.StatusCode, c.desc)
			svcCall.Unset()
			authCall.Unset()
		})
	}
}
