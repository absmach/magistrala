// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/eventlogs"
	"github.com/absmach/magistrala/eventlogs/api"
	"github.com/absmach/magistrala/eventlogs/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validToken = "valid"
	validID    = testsutil.GenerateUUID(&testing.T{})
)

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

func newEventsServer() (*httptest.Server, *mocks.Service) {
	svc := new(mocks.Service)

	logger := mglog.NewMock()
	mux := api.MakeHandler(svc, logger, "event-logs", "test")
	return httptest.NewServer(mux), svc
}

func TestListEventsEndpoint(t *testing.T) {
	es, svc := newEventsServer()

	cases := []struct {
		desc        string
		token       string
		url         string
		contentType string
		status      int
		svcErr      error
	}{
		{
			desc:   "successful",
			token:  validToken,
			url:    validID + "/" + auth.UserType,
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:  "empty token",
			token: "",
			url:   validID + "/" + auth.UserType,

			status: http.StatusUnauthorized,
			svcErr: nil,
		},
		{
			desc:   "with service error",
			token:  validToken,
			url:    validID + "/" + auth.UserType,
			status: http.StatusForbidden,
			svcErr: svcerr.ErrAuthorization,
		},
		{
			desc:   "with offset",
			token:  validToken,
			url:    validID + "/" + auth.UserType + "?offset=10",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid offset",
			token:  validToken,
			url:    validID + "/" + auth.UserType + "?offset=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with limit",
			token:  validToken,
			url:    validID + "/" + auth.UserType + "?limit=10",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid limit",
			token:  validToken,
			url:    validID + "/" + auth.UserType + "?limit=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with operation",
			token:  validToken,
			url:    validID + "/" + auth.UserType + "?operation=user.create",
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid operation",
			token:  validToken,
			url:    validID + "/" + auth.UserType + "?operation=user.create&operation=user.update",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with from",
			token:  validToken,
			url:    validID + "/" + auth.UserType + fmt.Sprintf("?to=%d", time.Now().UnixNano()),
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid from",
			token:  validToken,
			url:    validID + "/" + auth.UserType + "?from=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with to",
			token:  validToken,
			url:    validID + "/" + auth.UserType + fmt.Sprintf("?to=%d", time.Now().UnixNano()),
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "with invalid to",
			token:  validToken,
			url:    validID + "/" + auth.UserType + "?to=ten",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with empty id",
			token:  validToken,
			url:    "/" + auth.UserType,
			status: http.StatusBadRequest,
			svcErr: nil,
		},
		{
			desc:   "with empty id type",
			token:  validToken,
			url:    validID + "/",
			status: http.StatusNotFound,
			svcErr: nil,
		},
		{
			desc:   "with invalid id type",
			token:  validToken,
			url:    validID + "/invalid",
			status: http.StatusBadRequest,
			svcErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			repoCall := svc.On("ReadAll", mock.Anything, c.token, mock.Anything).Return(eventlogs.EventsPage{}, c.svcErr)
			req := testRequest{
				client: es.Client(),
				method: http.MethodGet,
				url:    es.URL + "/events/" + c.url,
				token:  c.token,
			}
			resp, err := req.make()
			assert.Nil(t, err, c.desc)
			defer resp.Body.Close()
			assert.Equal(t, c.status, resp.StatusCode, c.desc)
			repoCall.Unset()
		})
	}
}
